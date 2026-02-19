package mapper // ApplyQueryMappings applies the specified mapping rules to a JSON object

import (
	"encoding/json"
	"fmt"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/matcher"
	"github.com/KorAP/Koral-Mapper/parser"
)

// ApplyQueryMappings applies the specified mapping rules to a JSON object
func (m *Mapper) ApplyQueryMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	// Validate mapping ID
	if _, exists := m.mappingLists[mappingID]; !exists {
		return nil, fmt.Errorf("mapping list with ID %s not found", mappingID)
	}

	if m.mappingLists[mappingID].IsCorpus() {
		return m.applyCorpusQueryMappings(mappingID, opts, jsonData)
	}

	// Get the parsed rules
	rules := m.parsedQueryRules[mappingID]

	// Check if we have a wrapper object with a "query" field
	var queryData any
	var hasQueryWrapper bool

	if jsonMap, ok := jsonData.(map[string]any); ok {
		if query, exists := jsonMap["query"]; exists {
			queryData = query
			hasQueryWrapper = true
		}
	}

	// If no query wrapper was found, use the entire input
	if !hasQueryWrapper {
		// If the input itself is not a valid query object, return it as is
		if !isValidQueryObject(jsonData) {
			return jsonData, nil
		}
		queryData = jsonData
	} else if queryData == nil || !isValidQueryObject(queryData) {
		// If we have a query wrapper but the query is nil or not a valid object,
		// return the original data
		return jsonData, nil
	}

	// Store rewrites if they exist
	var oldRewrites any
	if queryMap, ok := queryData.(map[string]any); ok {
		if rewrites, exists := queryMap["rewrites"]; exists {
			oldRewrites = rewrites
			delete(queryMap, "rewrites")
		}
	}

	// Convert input JSON to AST
	jsonBytes, err := json.Marshal(queryData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input JSON: %w", err)
	}

	node, err := parser.ParseJSON(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON into AST: %w", err)
	}

	// Store whether the input was a Token
	isToken := false
	var tokenWrap ast.Node
	if token, ok := node.(*ast.Token); ok {
		isToken = true
		tokenWrap = token.Wrap
		node = tokenWrap
	}

	// Store original node for rewrite if needed
	var originalNode ast.Node
	if opts.AddRewrites {
		originalNode = node.Clone()
	}

	// Pre-check foundry/layer overrides to optimize processing
	var patternFoundry, patternLayer, replacementFoundry, replacementLayer string
	if opts.Direction { // true means AtoB
		patternFoundry, patternLayer = opts.FoundryA, opts.LayerA
		replacementFoundry, replacementLayer = opts.FoundryB, opts.LayerB
	} else {
		patternFoundry, patternLayer = opts.FoundryB, opts.LayerB
		replacementFoundry, replacementLayer = opts.FoundryA, opts.LayerA
	}

	// Create a pattern cache key for memoization
	type patternCacheKey struct {
		ruleIndex     int
		foundry       string
		layer         string
		isReplacement bool
	}
	patternCache := make(map[patternCacheKey]ast.Node)

	// Apply each rule to the AST
	for i, rule := range rules {
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

		// Get or create pattern with overrides
		patternKey := patternCacheKey{ruleIndex: i, foundry: patternFoundry, layer: patternLayer, isReplacement: false}
		processedPattern, exists := patternCache[patternKey]
		if !exists {
			// Clone pattern only when needed
			processedPattern = pattern.Clone()
			// Apply foundry and layer overrides only if they're non-empty
			if patternFoundry != "" || patternLayer != "" {
				ast.ApplyFoundryAndLayerOverrides(processedPattern, patternFoundry, patternLayer)
			}
			patternCache[patternKey] = processedPattern
		}

		// Create a temporary matcher to check for actual matches
		tempMatcher, err := matcher.NewMatcher(ast.Pattern{Root: processedPattern}, ast.Replacement{Root: &ast.Term{}})
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary matcher: %w", err)
		}

		// Only proceed if there's an actual match
		if !tempMatcher.Match(node) {
			continue
		}

		// Get or create replacement with overrides (lazy evaluation)
		replacementKey := patternCacheKey{ruleIndex: i, foundry: replacementFoundry, layer: replacementLayer, isReplacement: true}
		processedReplacement, exists := patternCache[replacementKey]
		if !exists {
			// Clone replacement only when we have a match
			processedReplacement = replacement.Clone()
			// Apply foundry and layer overrides only if they're non-empty
			if replacementFoundry != "" || replacementLayer != "" {
				ast.ApplyFoundryAndLayerOverrides(processedReplacement, replacementFoundry, replacementLayer)
			}
			patternCache[replacementKey] = processedReplacement
		}

		// Create the actual matcher and apply replacement
		actualMatcher, err := matcher.NewMatcher(ast.Pattern{Root: processedPattern}, ast.Replacement{Root: processedReplacement})
		if err != nil {
			return nil, fmt.Errorf("failed to create matcher: %w", err)
		}
		node = actualMatcher.Replace(node)
	}

	// Wrap the result in a token if the input was a token
	var result ast.Node
	if isToken {
		result = &ast.Token{Wrap: node}
	} else {
		result = node
	}

	// Convert AST back to JSON
	resultBytes, err := parser.SerializeToJSON(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize AST to JSON: %w", err)
	}

	// Parse the JSON string back into
	var resultData any
	if err := json.Unmarshal(resultBytes, &resultData); err != nil {
		return nil, fmt.Errorf("failed to parse result JSON: %w", err)
	}

	// Add rewrites if enabled and node was changed
	if opts.AddRewrites && !ast.NodesEqual(node, originalNode) {
		rewrite := buildQueryRewrite(originalNode, node)

		// Add rewrite to the node
		if resultMap, ok := resultData.(map[string]any); ok {
			if wrapMap, ok := resultMap["wrap"].(map[string]any); ok {
				rewrites, exists := wrapMap["rewrites"]
				if !exists {
					rewrites = []any{}
				}
				if rewritesList, ok := rewrites.([]any); ok {
					wrapMap["rewrites"] = append(rewritesList, rewrite)
				} else {
					wrapMap["rewrites"] = []any{rewrite}
				}
			}
		}
	}

	// Restore rewrites if they existed
	if oldRewrites != nil {
		// Process old rewrites through AST to ensure backward compatibility
		if rewritesList, ok := oldRewrites.([]any); ok {
			processedRewrites := make([]any, len(rewritesList))
			for i, rewriteData := range rewritesList {
				// Marshal and unmarshal each rewrite to apply backward compatibility
				rewriteBytes, err := json.Marshal(rewriteData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal old rewrite %d: %w", i, err)
				}
				var rewrite ast.Rewrite
				if err := json.Unmarshal(rewriteBytes, &rewrite); err != nil {
					return nil, fmt.Errorf("failed to unmarshal old rewrite %d: %w", i, err)
				}
				// Marshal back to get the transformed version
				transformedBytes, err := json.Marshal(&rewrite)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal transformed rewrite %d: %w", i, err)
				}
				var transformedRewrite any
				if err := json.Unmarshal(transformedBytes, &transformedRewrite); err != nil {
					return nil, fmt.Errorf("failed to unmarshal transformed rewrite %d: %w", i, err)
				}
				processedRewrites[i] = transformedRewrite
			}
			if resultMap, ok := resultData.(map[string]any); ok {
				resultMap["rewrites"] = processedRewrites
			}
		} else {
			// If it's not a list, restore as-is
			if resultMap, ok := resultData.(map[string]any); ok {
				resultMap["rewrites"] = oldRewrites
			}
		}
	}

	// If we had a query wrapper, put the transformed data back in it
	if hasQueryWrapper {
		if wrapper, ok := jsonData.(map[string]any); ok {
			wrapper["query"] = resultData
			return wrapper, nil
		}
	}

	return resultData, nil
}

// buildQueryRewrite creates a rewrite entry for a query-level transformation
// by comparing the original and new AST nodes.
func buildQueryRewrite(originalNode, newNode ast.Node) map[string]any {
	if term, ok := originalNode.(*ast.Term); ok && ast.IsTermNode(newNode) && originalNode.Type() == newNode.Type() {
		newTerm := newNode.(*ast.Term)
		if term.Foundry != newTerm.Foundry {
			return newRewriteEntry("foundry", term.Foundry)
		}
		if term.Layer != newTerm.Layer {
			return newRewriteEntry("layer", term.Layer)
		}
		if term.Key != newTerm.Key {
			return newRewriteEntry("key", term.Key)
		}
		if term.Value != newTerm.Value {
			return newRewriteEntry("value", term.Value)
		}
	}

	originalBytes, err := parser.SerializeToJSON(originalNode)
	if err != nil {
		return newRewriteEntry("", nil)
	}
	var originalJSON any
	if err := json.Unmarshal(originalBytes, &originalJSON); err != nil {
		return newRewriteEntry("", nil)
	}
	return newRewriteEntry("", originalJSON)
}

// isValidQueryObject checks if the query data is a valid object that can be processed
func isValidQueryObject(data any) bool {
	// Check if it's a map
	queryMap, ok := data.(map[string]any)
	if !ok {
		return false
	}

	// Check if it has the required @type field
	if _, ok := queryMap["@type"]; !ok {
		return false
	}

	return true
}
