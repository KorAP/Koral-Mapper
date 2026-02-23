package mapper

import (
	"regexp"

	"github.com/KorAP/Koral-Mapper/parser"
)

// applyCorpusQueryMappings processes corpus/collection section with corpus rules.
// Rules are applied iteratively: each rule is applied to the entire tree,
// and subsequent rules see the transformed result.
func (m *Mapper) applyCorpusQueryMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	rules := m.rulesWithFieldOverrides(m.parsedCorpusRules[mappingID], opts)

	jsonMap, ok := jsonData.(map[string]any)
	if !ok {
		return jsonData, nil
	}

	corpusKey := ""
	if _, exists := jsonMap["corpus"]; exists {
		corpusKey = "corpus"
	} else if _, exists := jsonMap["collection"]; exists {
		corpusKey = "collection"
	}

	if corpusKey == "" {
		return jsonData, nil
	}

	corpusData, ok := jsonMap[corpusKey].(map[string]any)
	if !ok {
		return jsonData, nil
	}

	result := shallowCopyMap(jsonMap)

	var current any = corpusData
	for _, rule := range rules {
		current = m.applyCorpusRule(current, rule, opts)
	}
	result[corpusKey] = current

	return result, nil
}

// applyCorpusRule applies a single corpus mapping rule to a node tree.
// It matches at the current level first, then recurses into operands
// if no match is found.
func (m *Mapper) applyCorpusRule(nodeAny any, rule *parser.CorpusMappingResult, opts MappingOptions) any {
	node, ok := nodeAny.(map[string]any)
	if !ok {
		return nodeAny
	}

	atType, _ := node["@type"].(string)
	if atType == "koral:docGroupRef" {
		return node
	}

	var pattern, replacement parser.CorpusNode
	if opts.Direction == AtoB {
		pattern, replacement = rule.Upper, rule.Lower
	} else {
		pattern, replacement = rule.Lower, rule.Upper
	}

	if matchCorpusNode(pattern, node) {
		// AND subset match: node has more operands than pattern
		if pg, ok := pattern.(*parser.CorpusGroup); ok && pg.Operation == "and" {
			operandsRaw, _ := node["operands"].([]any)
			if operandsRaw != nil && len(operandsRaw) > len(pg.Operands) {
				return m.buildSubsetANDReplacement(node, pg.Operands, replacement, opts)
			}
		}

		replaced := buildReplacementFromNode(replacement, node)
		if opts.AddRewrites {
			addCorpusRewrite(replaced, node)
		}
		return replaced
	}

	// No match at this level; recurse into operands if it's a group
	if atType == "koral:docGroup" || atType == "koral:fieldGroup" {
		return m.applyCorpusRuleToOperands(node, rule, opts)
	}

	return node
}

// applyCorpusRuleToOperands recursively applies a single rule to operands of a docGroup.
func (m *Mapper) applyCorpusRuleToOperands(node map[string]any, rule *parser.CorpusMappingResult, opts MappingOptions) any {
	result := shallowCopyMap(node)

	operandsRaw, ok := node["operands"].([]any)
	if !ok {
		return result
	}

	newOperands := make([]any, len(operandsRaw))
	for i, opRaw := range operandsRaw {
		newOperands[i] = m.applyCorpusRule(opRaw, rule, opts)
	}
	result["operands"] = newOperands

	return result
}

// buildSubsetANDReplacement handles AND patterns that match a subset of a
// group's operands. The matched operands are replaced and unmatched ones
// are preserved alongside the replacement.
func (m *Mapper) buildSubsetANDReplacement(node map[string]any, patternOps []parser.CorpusNode, replacement parser.CorpusNode, opts MappingOptions) any {
	operandsRaw, _ := node["operands"].([]any)

	used := make([]bool, len(operandsRaw))
	for _, patOp := range patternOps {
		for j, docOpRaw := range operandsRaw {
			if used[j] {
				continue
			}
			docOp, ok := docOpRaw.(map[string]any)
			if !ok {
				continue
			}
			if matchCorpusNode(patOp, docOp) {
				used[j] = true
				break
			}
		}
	}

	var remaining []any
	for j, docOpRaw := range operandsRaw {
		if !used[j] {
			remaining = append(remaining, docOpRaw)
		}
	}

	replacementNode := buildReplacementFromNode(replacement, node)
	newOperands := append([]any{replacementNode}, remaining...)

	if len(newOperands) == 1 {
		result := newOperands[0]
		if opts.AddRewrites {
			if resultMap, ok := result.(map[string]any); ok {
				addCorpusRewrite(resultMap, node)
			}
		}
		return result
	}

	result := shallowCopyMap(node)
	result["operands"] = newOperands

	if opts.AddRewrites {
		addCorpusRewrite(result, node)
	}

	return result
}

// matchCorpusNode checks if a JSON node matches a CorpusNode pattern.
// For CorpusField patterns, the node must be a koral:doc/koral:field.
// For CorpusGroup patterns, the node must be a koral:docGroup/koral:fieldGroup
// with matching operation and exactly matching operands (commutative).
func matchCorpusNode(pattern parser.CorpusNode, node map[string]any) bool {
	switch p := pattern.(type) {
	case *parser.CorpusField:
		atType, _ := node["@type"].(string)
		if atType != "koral:doc" && atType != "koral:field" {
			return false
		}
		return matchCorpusField(p, node)
	case *parser.CorpusGroup:
		return matchCorpusGroupNode(p, node)
	}
	return false
}

// matchCorpusGroupNode checks if a JSON node matches a CorpusGroup pattern.
//
// OR patterns: for leaf nodes (doc/field), any operand matching suffices.
// For group nodes, structural matching requires an OR docGroup/fieldGroup
// with exactly matching operands (commutative, exact count).
//
// AND patterns: the node must be a docGroup/fieldGroup with AND operation
// and all pattern operands must be found (subset matching — the node may
// have additional operands beyond those in the pattern).
func matchCorpusGroupNode(pattern *parser.CorpusGroup, node map[string]any) bool {
	atType, _ := node["@type"].(string)

	if pattern.Operation == "or" {
		// Leaf nodes: any-operand matching
		if atType == "koral:doc" || atType == "koral:field" {
			for _, op := range pattern.Operands {
				if matchCorpusNode(op, node) {
					return true
				}
			}
			return false
		}
		// Group nodes: structural matching (exact operand count)
		if atType != "koral:docGroup" && atType != "koral:fieldGroup" {
			return false
		}
		operation, _ := node["operation"].(string)
		if operation != "operation:or" {
			return false
		}
		return matchGroupOperands(pattern.Operands, node, true)
	}

	// AND patterns: subset matching
	if atType != "koral:docGroup" && atType != "koral:fieldGroup" {
		return false
	}
	operation, _ := node["operation"].(string)
	if operation != "operation:and" {
		return false
	}
	return matchGroupOperands(pattern.Operands, node, false)
}

// matchGroupOperands checks if a docGroup's operands match a pattern's
// operands using commutative set matching. When exactCount is true, the
// operand counts must be equal; otherwise subset matching is used (the
// node may have more operands than the pattern).
func matchGroupOperands(patternOps []parser.CorpusNode, node map[string]any, exactCount bool) bool {
	operandsRaw, ok := node["operands"].([]any)
	if !ok {
		return false
	}
	if exactCount {
		if len(operandsRaw) != len(patternOps) {
			return false
		}
	} else {
		if len(operandsRaw) < len(patternOps) {
			return false
		}
	}

	used := make([]bool, len(operandsRaw))
	for _, patOp := range patternOps {
		found := false
		for j, docOpRaw := range operandsRaw {
			if used[j] {
				continue
			}
			docOp, ok := docOpRaw.(map[string]any)
			if !ok {
				continue
			}
			if matchCorpusNode(patOp, docOp) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// matchCorpusField checks if a koral:doc JSON node matches a CorpusField pattern.
func matchCorpusField(pattern *parser.CorpusField, doc map[string]any) bool {
	docKey, _ := doc["key"].(string)
	if docKey != pattern.Key {
		return false
	}

	docValue, _ := doc["value"].(string)
	if pattern.Type == "regex" {
		re, err := regexp.Compile("^" + pattern.Value + "$")
		if err != nil {
			return false
		}
		if !re.MatchString(docValue) {
			return false
		}
	} else if docValue != pattern.Value {
		return false
	}

	if pattern.Match != "" {
		docMatch, _ := doc["match"].(string)
		expected := "match:" + pattern.Match
		if docMatch != expected {
			return false
		}
	}

	if pattern.Type != "" && pattern.Type != "regex" {
		docType, _ := doc["type"].(string)
		expected := "type:" + pattern.Type
		if docType != "" && docType != expected {
			return false
		}
	}

	return true
}

// buildReplacementFromNode builds a replacement JSON structure from a CorpusNode pattern.
// Preserves match and type from the original doc when the rule doesn't specify them.
func buildReplacementFromNode(replacement parser.CorpusNode, originalDoc map[string]any) any {
	switch r := replacement.(type) {
	case *parser.CorpusField:
		// Determine @type: use the original's type for doc/field, default to koral:doc
		atType := "koral:doc"
		if origType, _ := originalDoc["@type"].(string); origType == "koral:doc" || origType == "koral:field" {
			atType = origType
		}

		result := map[string]any{
			"@type": atType,
			"key":   r.Key,
			"value": r.Value,
		}

		if r.Match != "" {
			result["match"] = "match:" + r.Match
		} else if m, ok := originalDoc["match"]; ok {
			result["match"] = m
		}

		if r.Type != "" {
			result["type"] = "type:" + r.Type
		} else if t, ok := originalDoc["type"]; ok {
			result["type"] = t
		}

		return result

	case *parser.CorpusGroup:
		operands := make([]any, len(r.Operands))
		for i, op := range r.Operands {
			operands[i] = buildReplacementFromNode(op, originalDoc)
		}
		return map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:" + r.Operation,
			"operands":  operands,
		}

	default:
		return originalDoc
	}
}

// addCorpusRewrite adds a koral:rewrite annotation to the replaced node.
func addCorpusRewrite(replaced any, original map[string]any) {
	replacedMap, ok := replaced.(map[string]any)
	if !ok {
		return
	}

	origAtType, _ := original["@type"].(string)

	// If the original was a group, store the whole structure as the rewrite original
	if origAtType == "koral:docGroup" || origAtType == "koral:fieldGroup" {
		rewrite := newRewriteEntry("", original)
		replacedMap["rewrites"] = []any{rewrite}
		return
	}

	origKey, _ := original["key"].(string)
	newKey, _ := replacedMap["key"].(string)

	var rewrite map[string]any
	if origKey != newKey && origKey != "" {
		rewrite = newRewriteEntry("key", origKey)
	} else {
		origValue, _ := original["value"].(string)
		rewrite = newRewriteEntry("value", origValue)
	}

	replacedMap["rewrites"] = []any{rewrite}
}

// applyCorpusResponseMappings processes fields arrays with corpus rules.
func (m *Mapper) applyCorpusResponseMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	rules := m.rulesWithFieldOverrides(m.parsedCorpusRules[mappingID], opts)

	jsonMap, ok := jsonData.(map[string]any)
	if !ok {
		return jsonData, nil
	}

	fieldsRaw, exists := jsonMap["fields"]
	if !exists {
		return jsonData, nil
	}

	fields, ok := fieldsRaw.([]any)
	if !ok {
		return jsonData, nil
	}

	var newFields []any
	for _, fieldRaw := range fields {
		newFields = append(newFields, fieldRaw)

		fieldMap, ok := fieldRaw.(map[string]any)
		if !ok {
			continue
		}

		atType, _ := fieldMap["@type"].(string)
		if atType != "koral:field" && atType != "koral:doc" {
			continue
		}

		fieldKey, _ := fieldMap["key"].(string)
		fieldValue := fieldMap["value"]

		mapped := m.matchFieldAndCollect(fieldKey, fieldValue, rules, opts)
		newFields = append(newFields, mapped...)
	}

	result := shallowCopyMap(jsonMap)
	result["fields"] = newFields
	return result, nil
}

// matchFieldAndCollect matches a field's key/value against rules and returns mapped entries.
// For array values, each element is matched individually.
func (m *Mapper) matchFieldAndCollect(key string, value any, rules []*parser.CorpusMappingResult, opts MappingOptions) []any {
	var results []any

	switch v := value.(type) {
	case string:
		results = append(results, m.matchSingleValue(key, v, rules, opts)...)
	case []any:
		for _, elem := range v {
			if s, ok := elem.(string); ok {
				results = append(results, m.matchSingleValue(key, s, rules, opts)...)
			}
		}
	}

	return results
}

// matchSingleValue checks a single key+value pair against all rules and returns mapped field entries.
// Supports field patterns (direct match) and OR group patterns (any operand match).
// AND group patterns cannot match a single field and are skipped.
func (m *Mapper) matchSingleValue(key, value string, rules []*parser.CorpusMappingResult, opts MappingOptions) []any {
	var results []any

	pseudoDoc := map[string]any{
		"key":   key,
		"value": value,
	}

	for _, rule := range rules {
		var pattern, replacement parser.CorpusNode
		if opts.Direction == AtoB {
			pattern, replacement = rule.Upper, rule.Lower
		} else {
			pattern, replacement = rule.Lower, rule.Upper
		}

		if !matchCorpusFieldPattern(pattern, pseudoDoc) {
			continue
		}

		results = append(results, collectReplacementFields(replacement)...)
	}

	return results
}

// matchCorpusFieldPattern checks if a single response field matches a pattern.
// Field patterns match directly. OR group patterns match if any operand matches.
// AND group patterns cannot match a single field.
func matchCorpusFieldPattern(pattern parser.CorpusNode, doc map[string]any) bool {
	switch p := pattern.(type) {
	case *parser.CorpusField:
		return matchCorpusField(p, doc)
	case *parser.CorpusGroup:
		if p.Operation == "or" {
			for _, op := range p.Operands {
				if matchCorpusFieldPattern(op, doc) {
					return true
				}
			}
		}
	}
	return false
}

// collectReplacementFields flattens a replacement CorpusNode into individual
// mapped field entries. OR groups are skipped because response fields are flat
// key/value entries and OR semantics (one-of) cannot be represented. AND groups
// are flattened — all operands become individual fields.
func collectReplacementFields(node parser.CorpusNode) []any {
	var results []any

	switch n := node.(type) {
	case *parser.CorpusField:
		entry := map[string]any{
			"@type":  "koral:field",
			"key":    n.Key,
			"value":  n.Value,
			"mapped": true,
		}
		if n.Type != "" {
			entry["type"] = "type:" + n.Type
		} else {
			entry["type"] = "type:string"
		}
		results = append(results, entry)

	case *parser.CorpusGroup:
		if n.Operation == "or" {
			return nil
		}
		for _, op := range n.Operands {
			results = append(results, collectReplacementFields(op)...)
		}
	}

	return results
}

func shallowCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func (m *Mapper) rulesWithFieldOverrides(rules []*parser.CorpusMappingResult, opts MappingOptions) []*parser.CorpusMappingResult {
	if opts.FieldA == "" && opts.FieldB == "" {
		return rules
	}

	result := make([]*parser.CorpusMappingResult, len(rules))
	for i, rule := range rules {
		upper := rule.Upper.Clone()
		lower := rule.Lower.Clone()

		if opts.FieldA != "" {
			applyCorpusKeyOverride(upper, opts.FieldA)
		}
		if opts.FieldB != "" {
			applyCorpusKeyOverride(lower, opts.FieldB)
		}

		result[i] = &parser.CorpusMappingResult{
			Upper: upper,
			Lower: lower,
		}
	}

	return result
}

func applyCorpusKeyOverride(node parser.CorpusNode, key string) {
	switch n := node.(type) {
	case *parser.CorpusField:
		n.Key = key
	case *parser.CorpusGroup:
		for _, op := range n.Operands {
			applyCorpusKeyOverride(op, key)
		}
	}
}

