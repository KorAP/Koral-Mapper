package mapper

import (
	"regexp"

	"github.com/KorAP/Koral-Mapper/parser"
)

// applyCorpusQueryMappings processes corpus/collection section with corpus rules.
func (m *Mapper) applyCorpusQueryMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	rules := m.parsedCorpusRules[mappingID]

	jsonMap, ok := jsonData.(map[string]any)
	if !ok {
		return jsonData, nil
	}

	// Find corpus or collection attribute
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
	rewritten := m.rewriteCorpusNode(corpusData, rules, opts)
	result[corpusKey] = rewritten

	return result, nil
}

// rewriteCorpusNode recursively walks a corpus tree and applies matching rules.
func (m *Mapper) rewriteCorpusNode(node map[string]any, rules []*parser.CorpusMappingResult, opts MappingOptions) any {
	atType, _ := node["@type"].(string)

	switch atType {
	case "koral:doc", "koral:field":
		return m.rewriteCorpusDoc(node, rules, opts)
	case "koral:docGroup", "koral:fieldGroup":
		return m.rewriteCorpusDocGroup(node, rules, opts)
	case "koral:docGroupRef":
		return node
	default:
		return node
	}
}

// rewriteCorpusDoc attempts to match a koral:doc node against rules and replace it.
func (m *Mapper) rewriteCorpusDoc(node map[string]any, rules []*parser.CorpusMappingResult, opts MappingOptions) any {
	for _, rule := range rules {
		var pattern, replacement parser.CorpusNode
		if opts.Direction == AtoB {
			pattern, replacement = rule.Upper, rule.Lower
		} else {
			pattern, replacement = rule.Lower, rule.Upper
		}

		patternField, ok := pattern.(*parser.CorpusField)
		if !ok {
			continue
		}

		if !matchCorpusField(patternField, node) {
			continue
		}

		replaced := buildReplacementFromNode(replacement, node)

		if opts.AddRewrites {
			addCorpusRewrite(replaced, node)
		}

		return replaced
	}

	return node
}

// rewriteCorpusDocGroup recursively rewrites operands of a koral:docGroup.
func (m *Mapper) rewriteCorpusDocGroup(node map[string]any, rules []*parser.CorpusMappingResult, opts MappingOptions) any {
	result := shallowCopyMap(node)

	operandsRaw, ok := node["operands"].([]any)
	if !ok {
		return result
	}

	newOperands := make([]any, len(operandsRaw))
	for i, opRaw := range operandsRaw {
		opMap, ok := opRaw.(map[string]any)
		if !ok {
			newOperands[i] = opRaw
			continue
		}
		newOperands[i] = m.rewriteCorpusNode(opMap, rules, opts)
	}
	result["operands"] = newOperands

	return result
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
		result := map[string]any{
			"@type": originalDoc["@type"],
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
	rules := m.parsedCorpusRules[mappingID]

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

		patternField, ok := pattern.(*parser.CorpusField)
		if !ok {
			continue
		}

		if !matchCorpusField(patternField, pseudoDoc) {
			continue
		}

		results = append(results, collectReplacementFields(replacement)...)
	}

	return results
}

// collectReplacementFields flattens a replacement CorpusNode into individual mapped field entries.
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

