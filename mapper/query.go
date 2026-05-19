package mapper

import (
	"encoding/json"
	"fmt"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/matcher"
	"github.com/KorAP/Koral-Mapper/parser"
)

// ApplyQueryMappings transforms a JSON query object using the mapping rules
// identified by mappingID. The input may be a bare query node or a wrapper
// object containing a "query" field; both forms are accepted.
func (m *Mapper) ApplyQueryMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	if _, exists := m.mappingLists[mappingID]; !exists {
		return nil, fmt.Errorf("mapping list with ID %s not found", mappingID)
	}

	if m.mappingLists[mappingID].IsCorpus() {
		return m.applyCorpusQueryMappings(mappingID, opts, jsonData)
	}

	rules := m.parsedQueryRules[mappingID]

	// Detect wrapper: input may be {"query": ...} or a bare koral:token
	var queryData any
	var hasQueryWrapper bool

	if jsonMap, ok := jsonData.(map[string]any); ok {
		if query, exists := jsonMap["query"]; exists {
			queryData = query
			hasQueryWrapper = true
		}
	}

	if !hasQueryWrapper {
		if !isValidQueryObject(jsonData) {
			return jsonData, nil
		}
		queryData = jsonData
	} else if queryData == nil || !isValidQueryObject(queryData) {
		return jsonData, nil
	}

	// Strip pre-existing rewrites before AST conversion so they do not
	// interfere with matching. They are restored after transformation.
	var oldRewrites any
	if queryMap, ok := queryData.(map[string]any); ok {
		if rewrites, exists := queryMap["rewrites"]; exists {
			oldRewrites = rewrites
			delete(queryMap, "rewrites")
		}
	}

	jsonBytes, err := json.Marshal(queryData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input JSON: %w", err)
	}

	node, err := parser.ParseJSON(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON into AST: %w", err)
	}

	// Unwrap Token so matching operates on the inner node; re-wrapped later.
	isToken := false
	var tokenWrap ast.Node
	if token, ok := node.(*ast.Token); ok {
		isToken = true
		tokenWrap = token.Wrap
		node = tokenWrap
	}

	// Resolve foundry/layer overrides per direction once, before the rule loop.
	var patternFoundry, patternLayer, replacementFoundry, replacementLayer string
	if opts.Direction {
		patternFoundry, patternLayer = opts.FoundryA, opts.LayerA
		replacementFoundry, replacementLayer = opts.FoundryB, opts.LayerB
	} else {
		patternFoundry, patternLayer = opts.FoundryB, opts.LayerB
		replacementFoundry, replacementLayer = opts.FoundryA, opts.LayerA
	}

	// patternCache avoids redundant Clone+Override for the same rule index
	// and foundry/layer combination across repeated calls.
	type patternCacheKey struct {
		ruleIndex     int
		foundry       string
		layer         string
		isReplacement bool
	}
	patternCache := make(map[patternCacheKey]ast.Node)

	for i, rule := range rules {
		var pattern, replacement ast.Node
		if opts.Direction {
			pattern = rule.Upper
			replacement = rule.Lower
		} else {
			pattern = rule.Lower
			replacement = rule.Upper
		}

		if token, ok := pattern.(*ast.Token); ok {
			pattern = token.Wrap
		}
		if token, ok := replacement.(*ast.Token); ok {
			replacement = token.Wrap
		}

		patternKey := patternCacheKey{ruleIndex: i, foundry: patternFoundry, layer: patternLayer, isReplacement: false}
		processedPattern, exists := patternCache[patternKey]
		if !exists {
			processedPattern = pattern.Clone()
			if patternFoundry != "" || patternLayer != "" {
				ast.ApplyFoundryAndLayerOverrides(processedPattern, patternFoundry, patternLayer)
			}
			patternCache[patternKey] = processedPattern
		}

		// Probe for a match before cloning the replacement (lazy evaluation)
		tempMatcher, err := matcher.NewMatcher(ast.Pattern{Root: processedPattern}, ast.Replacement{Root: &ast.Term{}})
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary matcher: %w", err)
		}
		if !tempMatcher.Match(node) {
			continue
		}

		replacementKey := patternCacheKey{ruleIndex: i, foundry: replacementFoundry, layer: replacementLayer, isReplacement: true}
		processedReplacement, exists := patternCache[replacementKey]
		if !exists {
			processedReplacement = replacement.Clone()
			if replacementFoundry != "" || replacementLayer != "" {
				ast.ApplyFoundryAndLayerOverrides(processedReplacement, replacementFoundry, replacementLayer)
			}
			patternCache[replacementKey] = processedReplacement
		}

		var beforeNode ast.Node
		if opts.AddRewrites {
			beforeNode = node.Clone()
		}

		actualMatcher, err := matcher.NewMatcher(ast.Pattern{Root: processedPattern}, ast.Replacement{Root: processedReplacement})
		if err != nil {
			return nil, fmt.Errorf("failed to create matcher: %w", err)
		}
		node = actualMatcher.Replace(node)

		if opts.AddRewrites {
			recordRewrites(node, beforeNode)
		}
	}

	var result ast.Node
	if isToken {
		result = &ast.Token{Wrap: node}
	} else {
		result = node
	}

	resultBytes, err := parser.SerializeToJSON(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize AST to JSON: %w", err)
	}

	var resultData any
	if err := json.Unmarshal(resultBytes, &resultData); err != nil {
		return nil, fmt.Errorf("failed to parse result JSON: %w", err)
	}

	// Restore pre-existing rewrites. The round-trip through ast.Rewrite
	// normalizes legacy field names (e.g. "source" -> "editor") so the
	// output always uses the modern schema.
	if oldRewrites != nil {
		if rewritesList, ok := oldRewrites.([]any); ok {
			processedRewrites := make([]any, len(rewritesList))
			for i, rewriteData := range rewritesList {
				rewriteBytes, err := json.Marshal(rewriteData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal old rewrite %d: %w", i, err)
				}
				var rewrite ast.Rewrite
				if err := json.Unmarshal(rewriteBytes, &rewrite); err != nil {
					return nil, fmt.Errorf("failed to unmarshal old rewrite %d: %w", i, err)
				}
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
			if resultMap, ok := resultData.(map[string]any); ok {
				resultMap["rewrites"] = oldRewrites
			}
		}
	}

	if hasQueryWrapper {
		if wrapper, ok := jsonData.(map[string]any); ok {
			wrapper["query"] = resultData
			return wrapper, nil
		}
	}

	return resultData, nil
}

// recordRewrites compares the new node against the before-snapshot and
// attaches rewrite entries to any changed nodes. It handles both simple
// nodes (Term, TermGroup) and container nodes (CatchallNode with operands).
func recordRewrites(newNode, beforeNode ast.Node) {
	if ast.NodesEqual(newNode, beforeNode) {
		return
	}

	// For CatchallNodes with operands (e.g. token sequences), attach
	// per-operand rewrites so each changed token gets its own annotation.
	if newCatchall, ok := newNode.(*ast.CatchallNode); ok {
		if oldCatchall, ok := beforeNode.(*ast.CatchallNode); ok && len(newCatchall.Operands) > 0 {
			for i, newOp := range newCatchall.Operands {
				if i >= len(oldCatchall.Operands) {
					break
				}
				oldOp := oldCatchall.Operands[i]
				recordRewritesForOperand(newOp, oldOp)
			}
			return
		}
	}

	addRewriteToNode(newNode, beforeNode)
}

// recordRewritesForOperand handles rewrite recording for a single operand,
// unwrapping Token nodes so the rewrite attaches to the inner term/termGroup
// rather than the token wrapper.
func recordRewritesForOperand(newOp, oldOp ast.Node) {
	if ast.NodesEqual(newOp, oldOp) {
		return
	}

	newInner := newOp
	oldInner := oldOp
	if tok, ok := newOp.(*ast.Token); ok {
		newInner = tok.Wrap
	}
	if tok, ok := oldOp.(*ast.Token); ok {
		oldInner = tok.Wrap
	}

	if newInner == nil || ast.NodesEqual(newInner, oldInner) {
		return
	}

	addRewriteToNode(newInner, oldInner)
}

// addRewriteToNode creates and attaches a rewrite entry to a node,
// recording what the node looked like before the change.
func addRewriteToNode(newNode, originalNode ast.Node) {
	rw := buildRewrite(originalNode, newNode)
	ast.AppendRewrite(newNode, rw)
}

// buildRewrite creates a Rewrite describing what changed between
// originalNode and newNode. For simple term-level changes (just foundry,
// layer, key, or value), it uses a scoped rewrite. For structural changes,
// it stores the full original as an object.
func buildRewrite(originalNode, newNode ast.Node) ast.Rewrite {
	if term, ok := originalNode.(*ast.Term); ok && ast.IsTermNode(newNode) && originalNode.Type() == newNode.Type() {
		newTerm := newNode.(*ast.Term)
		if term.Foundry != newTerm.Foundry {
			return ast.Rewrite{Editor: RewriteEditor, Scope: "foundry", Original: term.Foundry}
		}
		if term.Layer != newTerm.Layer {
			return ast.Rewrite{Editor: RewriteEditor, Scope: "layer", Original: term.Layer}
		}
		if term.Key != newTerm.Key {
			return ast.Rewrite{Editor: RewriteEditor, Scope: "key", Original: term.Key}
		}
		if term.Value != newTerm.Value {
			return ast.Rewrite{Editor: RewriteEditor, Scope: "value", Original: term.Value}
		}
	}

	// Structural change: serialize the original as the rewrite value
	originalBytes, err := parser.SerializeToJSON(originalNode)
	if err != nil {
		return ast.Rewrite{Editor: RewriteEditor}
	}
	var originalJSON any
	if err := json.Unmarshal(originalBytes, &originalJSON); err != nil {
		return ast.Rewrite{Editor: RewriteEditor}
	}
	return ast.Rewrite{Editor: RewriteEditor, Original: originalJSON}
}

// isValidQueryObject returns true if data is a JSON object with an @type field.
func isValidQueryObject(data any) bool {
	queryMap, ok := data.(map[string]any)
	if !ok {
		return false
	}
	_, ok = queryMap["@type"]
	return ok
}
