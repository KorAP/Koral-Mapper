package parser

import (
	"fmt"
	"strings"
)

// CorpusNode represents a node in a corpus mapping rule.
type CorpusNode interface {
	isCorpusNode()
	Clone() CorpusNode
	ToJSON() map[string]any
}

// CorpusField represents a single koral:doc field constraint.
type CorpusField struct {
	Key   string
	Value string
	Match string // "eq","ne","geq","leq","contains","excludes" (empty = unspecified)
	Type  string // "string","regex","date" (empty = unspecified, defaults to "string")
}

func (f *CorpusField) isCorpusNode() {}

func (f *CorpusField) Clone() CorpusNode {
	return &CorpusField{Key: f.Key, Value: f.Value, Match: f.Match, Type: f.Type}
}

// ToJSON converts the field to a koral:doc JSON map.
func (f *CorpusField) ToJSON() map[string]any {
	m := map[string]any{
		"@type": "koral:doc",
		"key":   f.Key,
		"value": f.Value,
	}
	if f.Match != "" {
		m["match"] = "match:" + f.Match
	} else {
		m["match"] = "match:eq"
	}
	if f.Type != "" {
		m["type"] = "type:" + f.Type
	} else {
		m["type"] = "type:string"
	}
	return m
}

// CorpusGroup represents a koral:docGroup boolean group.
type CorpusGroup struct {
	Operation string // "and" or "or"
	Operands  []CorpusNode
}

func (g *CorpusGroup) isCorpusNode() {}

func (g *CorpusGroup) Clone() CorpusNode {
	ops := make([]CorpusNode, len(g.Operands))
	for i, op := range g.Operands {
		ops[i] = op.Clone()
	}
	return &CorpusGroup{Operation: g.Operation, Operands: ops}
}

// ToJSON converts the group to a koral:docGroup JSON map.
func (g *CorpusGroup) ToJSON() map[string]any {
	operands := make([]map[string]any, len(g.Operands))
	for i, op := range g.Operands {
		operands[i] = op.ToJSON()
	}
	return map[string]any{
		"@type":     "koral:docGroup",
		"operation": "operation:" + g.Operation,
		"operands":  operands,
	}
}

// CorpusMappingResult represents a parsed corpus mapping rule.
type CorpusMappingResult struct {
	Upper CorpusNode // Side A
	Lower CorpusNode // Side B
}

// CorpusParser parses corpus mapping rules.
type CorpusParser struct {
	// AllowBareValues enables parsing values without a key= prefix.
	// The resulting CorpusField will have an empty Key, to be filled
	// from the mapping list header (KeyA/KeyB).
	AllowBareValues bool
}

func NewCorpusParser() *CorpusParser {
	return &CorpusParser{}
}

// ParseMapping parses a corpus mapping rule of the form "pattern <> replacement".
func (p *CorpusParser) ParseMapping(input string) (*CorpusMappingResult, error) {
	sepIdx := strings.Index(input, "<>")
	if sepIdx == -1 {
		return nil, fmt.Errorf("invalid corpus mapping rule: missing <> separator in %q", input)
	}

	leftStr := strings.TrimSpace(input[:sepIdx])
	rightStr := strings.TrimSpace(input[sepIdx+2:])

	if leftStr == "" {
		return nil, fmt.Errorf("invalid corpus mapping rule: empty left side")
	}
	if rightStr == "" {
		return nil, fmt.Errorf("invalid corpus mapping rule: empty right side")
	}

	upper, err := p.parseExpression(leftStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing left side: %w", err)
	}

	lower, err := p.parseExpression(rightStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing right side: %w", err)
	}

	return &CorpusMappingResult{Upper: upper, Lower: lower}, nil
}

// parseExpression parses a corpus expression (field or group).
func (p *CorpusParser) parseExpression(input string) (CorpusNode, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty expression")
	}

	if input[0] == '(' {
		closeIdx := findMatchingParen(input)
		if closeIdx == len(input)-1 {
			return p.parseGroupContent(input[1 : len(input)-1])
		}
	}

	return p.parseField(input)
}

// parseGroupContent parses the content inside parentheses.
func (p *CorpusParser) parseGroupContent(input string) (CorpusNode, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty group")
	}

	parts, operator, err := splitOnTopLevelOperator(input)
	if err != nil {
		return nil, err
	}

	if len(parts) == 1 {
		return p.parseExpression(parts[0])
	}

	operands := make([]CorpusNode, len(parts))
	for i, part := range parts {
		node, err := p.parseExpression(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("error in group operand %d: %w", i, err)
		}
		operands[i] = node
	}

	op := "and"
	if operator == "|" {
		op = "or"
	}

	return &CorpusGroup{Operation: op, Operands: operands}, nil
}

var validMatchTypes = map[string]bool{
	"eq": true, "ne": true, "geq": true, "leq": true,
	"contains": true, "excludes": true,
}

// parseField parses a single field expression: key=value[:match][#type].
// When AllowBareValues is true, also accepts bare values without key=.
func (p *CorpusParser) parseField(input string) (*CorpusField, error) {
	input = strings.TrimSpace(input)

	eqIdx := strings.Index(input, "=")
	if eqIdx == -1 {
		if !p.AllowBareValues {
			return nil, fmt.Errorf("invalid field expression: missing '=' in %q", input)
		}
		return p.parseBareValue(input)
	}

	key := strings.TrimSpace(input[:eqIdx])
	rest := strings.TrimSpace(input[eqIdx+1:])

	if key == "" {
		return nil, fmt.Errorf("invalid field expression: empty key")
	}
	if rest == "" {
		return nil, fmt.Errorf("invalid field expression: empty value for key %q", key)
	}

	field := &CorpusField{Key: key}

	// Split off #type first
	if hashIdx := strings.LastIndex(rest, "#"); hashIdx != -1 {
		field.Type = strings.TrimSpace(rest[hashIdx+1:])
		rest = rest[:hashIdx]
	}

	// Split off :match — only if the part after the last colon is a valid match type
	if colonIdx := strings.LastIndex(rest, ":"); colonIdx != -1 {
		candidate := strings.TrimSpace(rest[colonIdx+1:])
		if validMatchTypes[candidate] {
			field.Match = candidate
			rest = rest[:colonIdx]
		}
	}

	field.Value = strings.TrimSpace(rest)
	if field.Value == "" {
		return nil, fmt.Errorf("invalid field expression: empty value for key %q", key)
	}

	return field, nil
}

// parseBareValue parses a value without a key= prefix.
// The Key is left empty and should be filled from the mapping list header.
func (p *CorpusParser) parseBareValue(input string) (*CorpusField, error) {
	if input == "" {
		return nil, fmt.Errorf("invalid field expression: empty bare value")
	}

	field := &CorpusField{}

	if hashIdx := strings.LastIndex(input, "#"); hashIdx != -1 {
		field.Type = strings.TrimSpace(input[hashIdx+1:])
		input = input[:hashIdx]
	}

	if colonIdx := strings.LastIndex(input, ":"); colonIdx != -1 {
		candidate := strings.TrimSpace(input[colonIdx+1:])
		if validMatchTypes[candidate] {
			field.Match = candidate
			input = input[:colonIdx]
		}
	}

	field.Value = strings.TrimSpace(input)
	if field.Value == "" {
		return nil, fmt.Errorf("invalid field expression: empty bare value")
	}

	return field, nil
}

// findMatchingParen finds the index of the closing parenthesis matching the
// opening parenthesis at position 0.
func findMatchingParen(input string) int {
	depth := 0
	for i, ch := range input {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitOnTopLevelOperator splits a string on & or | operators at the top level
// (not inside parentheses). Returns the parts, the operator used, and any error.
func splitOnTopLevelOperator(input string) ([]string, string, error) {
	depth := 0
	var parts []string
	var operator string
	lastSplit := 0

	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '&', '|':
			if depth == 0 {
				op := string(input[i])
				if operator == "" {
					operator = op
				} else if op != operator {
					return nil, "", fmt.Errorf("mixed operators '&' and '|' at same level; use parentheses to disambiguate")
				}
				parts = append(parts, strings.TrimSpace(input[lastSplit:i]))
				lastSplit = i + 1
			}
		}
	}

	parts = append(parts, strings.TrimSpace(input[lastSplit:]))
	return parts, operator, nil
}
