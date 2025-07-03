package mapper

import (
	"fmt"
	"strings"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/matcher"
	"github.com/KorAP/Koral-Mapper/parser"
	"github.com/rs/zerolog/log"
)

// ApplyResponseMappings applies the specified mapping rules to a JSON object
func (m *Mapper) ApplyResponseMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	// Validate mapping ID
	if _, exists := m.mappingLists[mappingID]; !exists {
		return nil, fmt.Errorf("mapping list with ID %s not found", mappingID)
	}

	// Get the parsed rules
	rules := m.parsedRules[mappingID]

	// Check if we have a snippet to process
	jsonMap, ok := jsonData.(map[string]any)
	if !ok {
		return jsonData, nil
	}

	snippetValue, exists := jsonMap["snippet"]
	if !exists {
		return jsonData, nil
	}

	snippet, ok := snippetValue.(string)
	if !ok {
		return jsonData, nil
	}

	// Process the snippet with each rule
	processedSnippet := snippet
	for ruleIndex, rule := range rules {
		// Create pattern and replacement based on direction
		var pattern, replacement ast.Node
		if opts.Direction { // true means AtoB
			pattern = rule.Upper
			replacement = rule.Lower
		} else {
			pattern = rule.Lower
			replacement = rule.Upper
		}

		// Extract the inner nodes from the pattern and replacement tokens
		if token, ok := pattern.(*ast.Token); ok {
			pattern = token.Wrap
		}
		if token, ok := replacement.(*ast.Token); ok {
			replacement = token.Wrap
		}

		// Apply foundry and layer overrides with proper precedence
		mappingList := m.mappingLists[mappingID]

		// Determine foundry and layer values based on direction
		var patternFoundry, patternLayer, replacementFoundry, replacementLayer string
		if opts.Direction { // AtoB
			patternFoundry, patternLayer = opts.FoundryA, opts.LayerA
			replacementFoundry, replacementLayer = opts.FoundryB, opts.LayerB
			// Apply mapping list defaults if not specified
			if replacementFoundry == "" {
				replacementFoundry = mappingList.FoundryB
			}
			if replacementLayer == "" {
				replacementLayer = mappingList.LayerB
			}
		} else { // BtoA
			patternFoundry, patternLayer = opts.FoundryB, opts.LayerB
			replacementFoundry, replacementLayer = opts.FoundryA, opts.LayerA
			// Apply mapping list defaults if not specified
			if replacementFoundry == "" {
				replacementFoundry = mappingList.FoundryA
			}
			if replacementLayer == "" {
				replacementLayer = mappingList.LayerA
			}
		}

		// Clone pattern and apply foundry and layer overrides
		processedPattern := pattern.Clone()
		if patternFoundry != "" || patternLayer != "" {
			ast.ApplyFoundryAndLayerOverrides(processedPattern, patternFoundry, patternLayer)
		}

		// Create snippet matcher for this rule
		snippetMatcher, err := matcher.NewSnippetMatcher(
			ast.Pattern{Root: processedPattern},
			ast.Replacement{Root: replacement},
		)
		if err != nil {
			continue // Skip this rule if we can't create a matcher
		}

		// Find matching tokens in the snippet
		matchingTokens, err := snippetMatcher.FindMatchingTokens(processedSnippet)
		if err != nil {
			continue // Skip this rule if parsing fails
		}

		if len(matchingTokens) == 0 {
			continue // No matches, try next rule
		}

		// Apply RestrictToObligatory with layer precedence logic
		restrictedReplacement := m.applyReplacementWithLayerPrecedence(
			replacement, replacementFoundry, replacementLayer,
			mappingID, ruleIndex, bool(opts.Direction))
		if restrictedReplacement == nil {
			continue // Nothing obligatory to add
		}

		// Generate annotation strings from the restricted replacement
		annotationStrings, err := m.generateAnnotationStrings(restrictedReplacement)
		if err != nil {
			continue // Skip if we can't generate annotations
		}

		if len(annotationStrings) == 0 {
			continue // Nothing to add
		}

		// Apply annotations to matching tokens in the snippet
		processedSnippet, err = m.addAnnotationsToSnippet(processedSnippet, matchingTokens, annotationStrings)
		if err != nil {
			continue // Skip if we can't apply annotations
		}
	}

	log.Debug().Str("snippet", processedSnippet).Msg("Processed snippet")

	// Create a copy of the input data and update the snippet
	result := make(map[string]any)
	for k, v := range jsonMap {
		result[k] = v
	}
	result["snippet"] = processedSnippet

	return result, nil
}

// generateAnnotationStrings converts a replacement AST node into annotation strings
func (m *Mapper) generateAnnotationStrings(node ast.Node) ([]string, error) {
	if node == nil {
		return nil, nil
	}

	switch n := node.(type) {
	case *ast.Term:
		// Create annotation string in format "foundry/layer:key" or "foundry/layer:key:value"
		annotation := n.Foundry + "/" + n.Layer + ":" + n.Key
		if n.Value != "" {
			annotation += ":" + n.Value
		}
		return []string{annotation}, nil

	case *ast.TermGroup:
		if n.Relation == ast.AndRelation {
			// For AND groups, collect all annotations
			var allAnnotations []string
			for _, operand := range n.Operands {
				annotations, err := m.generateAnnotationStrings(operand)
				if err != nil {
					return nil, err
				}
				allAnnotations = append(allAnnotations, annotations...)
			}
			return allAnnotations, nil
		} else {
			// For OR groups (should not happen with RestrictToObligatory, but handle gracefully)
			return nil, nil
		}

	case *ast.Token:
		// Handle wrapped tokens
		if n.Wrap != nil {
			return m.generateAnnotationStrings(n.Wrap)
		}
		return nil, nil

	default:
		return nil, nil
	}
}

// addAnnotationsToSnippet adds new annotations to matching tokens in the snippet
func (m *Mapper) addAnnotationsToSnippet(snippet string, matchingTokens []matcher.TokenSpan, annotationStrings []string) (string, error) {
	if len(matchingTokens) == 0 || len(annotationStrings) == 0 {
		return snippet, nil
	}

	result := snippet

	// Process each matching token
	for _, token := range matchingTokens {
		// For nested span structure, we need to find the innermost text and wrap it
		// Look for the actual token text within span tags
		tokenText := token.Text

		// Find all occurrences of the token text in the current snippet
		// We need to be careful about which occurrence to replace
		startPos := 0
		for {
			tokenStart := strings.Index(result[startPos:], tokenText)
			if tokenStart == -1 {
				break // No more occurrences
			}
			tokenStart += startPos
			tokenEnd := tokenStart + len(tokenText)

			// Check if this token text is within the expected context
			// Look backwards and forwards to see if we're in the right span context
			beforeContext := result[:tokenStart]
			afterContext := result[tokenEnd:]

			// Simple heuristic: if we're immediately preceded by a > and followed by a <
			// then we're likely at the innermost text node
			if strings.HasSuffix(beforeContext, ">") && (strings.HasPrefix(afterContext, "<") || len(afterContext) == 0 || afterContext[0] == ' ') {
				// Build the replacement with nested spans for each annotation
				replacement := tokenText
				for i := len(annotationStrings) - 1; i >= 0; i-- {
					replacement = fmt.Sprintf(`<span title="%s" class="notinindex">%s</span>`, annotationStrings[i], replacement)
				}

				// Replace this occurrence
				result = result[:tokenStart] + replacement + result[tokenEnd:]
				break // Only replace the first appropriate occurrence for this token
			}

			// Move past this occurrence
			startPos = tokenEnd
		}
	}

	return result, nil
}

// applyReplacementWithLayerPrecedence applies RestrictToObligatory with proper layer precedence
func (m *Mapper) applyReplacementWithLayerPrecedence(
	replacement ast.Node, foundry, layerOverride string,
	mappingID string, ruleIndex int, direction bool) ast.Node {

	// First, apply RestrictToObligatory without layer override to preserve explicit layers
	restricted := ast.RestrictToObligatory(replacement, foundry, "")
	if restricted == nil {
		return nil
	}

	// If no layer override is specified, we're done
	if layerOverride == "" {
		return restricted
	}

	// Apply layer override only to terms that didn't have explicit layers in the original rule
	mappingList := m.mappingLists[mappingID]
	if ruleIndex < len(mappingList.Mappings) {
		originalRule := string(mappingList.Mappings[ruleIndex])
		m.applySelectiveLayerOverrides(restricted, layerOverride, originalRule, direction)
	}

	return restricted
}

// applySelectiveLayerOverrides applies layer overrides only to terms without explicit layers
func (m *Mapper) applySelectiveLayerOverrides(node ast.Node, layerOverride, originalRule string, direction bool) {
	if node == nil {
		return
	}

	// Parse the original rule without defaults to detect explicit layers
	explicitTerms := m.getExplicitTerms(originalRule, direction)

	// Apply overrides only to terms that weren't explicit in the original rule
	termIndex := 0
	m.applyLayerOverrideToImplicitTerms(node, layerOverride, explicitTerms, &termIndex)
}

// getExplicitTerms parses the original rule without defaults to identify terms with explicit layers
func (m *Mapper) getExplicitTerms(originalRule string, direction bool) map[int]bool {
	explicitTerms := make(map[int]bool)

	// Parse without defaults to see what was explicitly specified
	parser, err := parser.NewGrammarParser("", "")
	if err != nil {
		return explicitTerms
	}

	result, err := parser.ParseMapping(originalRule)
	if err != nil {
		return explicitTerms
	}

	// Get the replacement side based on direction
	var replacement ast.Node
	if direction { // AtoB
		replacement = result.Lower.Wrap
	} else { // BtoA
		replacement = result.Upper.Wrap
	}

	// Extract terms and check which ones have explicit layers
	termIndex := 0
	m.markExplicitTerms(replacement, explicitTerms, &termIndex)
	return explicitTerms
}

// markExplicitTerms recursively marks terms that have explicit layers
func (m *Mapper) markExplicitTerms(node ast.Node, explicitTerms map[int]bool, termIndex *int) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Term:
		// A term has an explicit layer if it was specified in the original rule
		if n.Layer != "" {
			explicitTerms[*termIndex] = true
		}
		*termIndex++

	case *ast.TermGroup:
		for _, operand := range n.Operands {
			m.markExplicitTerms(operand, explicitTerms, termIndex)
		}

	case *ast.Token:
		if n.Wrap != nil {
			m.markExplicitTerms(n.Wrap, explicitTerms, termIndex)
		}
	}
}

// applyLayerOverrideToImplicitTerms applies layer override only to terms not marked as explicit
func (m *Mapper) applyLayerOverrideToImplicitTerms(node ast.Node, layerOverride string, explicitTerms map[int]bool, termIndex *int) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Term:
		// Apply override only if this term wasn't explicit in the original rule
		if !explicitTerms[*termIndex] && n.Layer != "" {
			n.Layer = layerOverride
		}
		*termIndex++

	case *ast.TermGroup:
		for _, operand := range n.Operands {
			m.applyLayerOverrideToImplicitTerms(operand, layerOverride, explicitTerms, termIndex)
		}

	case *ast.Token:
		if n.Wrap != nil {
			m.applyLayerOverrideToImplicitTerms(n.Wrap, layerOverride, explicitTerms, termIndex)
		}
	}
}
