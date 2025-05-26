package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected NodeType
	}{
		{
			name:     "Token node returns correct type",
			node:     &Token{Wrap: &Term{}},
			expected: TokenNode,
		},
		{
			name: "TermGroup node returns correct type",
			node: &TermGroup{
				Operands: []Node{&Term{}},
				Relation: AndRelation,
			},
			expected: TermGroupNode,
		},
		{
			name: "Term node returns correct type",
			node: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
			},
			expected: TermNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.Type())
		})
	}
}

func TestTermGroupConstruction(t *testing.T) {
	term1 := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}

	term2 := &Term{
		Foundry: "opennlp",
		Key:     "AdjType",
		Layer:   "m",
		Match:   MatchEqual,
		Value:   "Pdt",
	}

	group := &TermGroup{
		Operands: []Node{term1, term2},
		Relation: AndRelation,
	}

	assert.Len(t, group.Operands, 2)
	assert.Equal(t, AndRelation, group.Relation)
	assert.Equal(t, TermGroupNode, group.Type())

	// Test operands are correctly set
	assert.Equal(t, term1, group.Operands[0])
	assert.Equal(t, term2, group.Operands[1])
}

func TestTokenConstruction(t *testing.T) {
	term := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}

	token := &Token{Wrap: term}

	assert.Equal(t, TokenNode, token.Type())
	assert.Equal(t, term, token.Wrap)
}

func TestTermConstruction(t *testing.T) {
	tests := []struct {
		name     string
		term     *Term
		foundry  string
		key      string
		layer    string
		match    MatchType
		hasValue bool
		value    string
	}{
		{
			name: "Term without value",
			term: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
			},
			foundry:  "opennlp",
			key:      "DET",
			layer:    "p",
			match:    MatchEqual,
			hasValue: false,
		},
		{
			name: "Term with value",
			term: &Term{
				Foundry: "opennlp",
				Key:     "AdjType",
				Layer:   "m",
				Match:   MatchEqual,
				Value:   "Pdt",
			},
			foundry:  "opennlp",
			key:      "AdjType",
			layer:    "m",
			match:    MatchEqual,
			hasValue: true,
			value:    "Pdt",
		},
		{
			name: "Term with not equal match",
			term: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchNotEqual,
			},
			foundry:  "opennlp",
			key:      "DET",
			layer:    "p",
			match:    MatchNotEqual,
			hasValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, TermNode, tt.term.Type())
			assert.Equal(t, tt.foundry, tt.term.Foundry)
			assert.Equal(t, tt.key, tt.term.Key)
			assert.Equal(t, tt.layer, tt.term.Layer)
			assert.Equal(t, tt.match, tt.term.Match)
			if tt.hasValue {
				assert.Equal(t, tt.value, tt.term.Value)
			} else {
				assert.Empty(t, tt.term.Value)
			}
		})
	}
}

func TestPatternAndReplacement(t *testing.T) {
	// Create a simple pattern
	patternTerm := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}
	pattern := Pattern{Root: patternTerm}

	// Create a simple replacement
	replacementTerm := &Term{
		Foundry: "opennlp",
		Key:     "COMBINED_DET",
		Layer:   "p",
		Match:   MatchEqual,
	}
	replacement := Replacement{Root: replacementTerm}

	// Test pattern
	assert.NotNil(t, pattern.Root)
	assert.Equal(t, patternTerm, pattern.Root)

	// Test replacement
	assert.NotNil(t, replacement.Root)
	assert.Equal(t, replacementTerm, replacement.Root)
}

func TestCatchallNode(t *testing.T) {
	tests := []struct {
		name       string
		nodeType   string
		content    string
		wrap       Node
		operands   []Node
		expectType NodeType
	}{
		{
			name:       "CatchallNode with custom type",
			nodeType:   "customType",
			content:    `{"key": "value"}`,
			expectType: NodeType("customType"),
		},
		{
			name:     "CatchallNode with wrapped term",
			nodeType: "wrapper",
			content:  `{"key": "value"}`,
			wrap: &Term{
				Foundry: "test",
				Key:     "TEST",
				Layer:   "x",
				Match:   MatchEqual,
			},
			expectType: NodeType("wrapper"),
		},
		{
			name:     "CatchallNode with operands",
			nodeType: "custom_group",
			content:  `{"key": "value"}`,
			operands: []Node{
				&Term{Foundry: "test1", Key: "TEST1", Layer: "x", Match: MatchEqual},
				&Term{Foundry: "test2", Key: "TEST2", Layer: "y", Match: MatchEqual},
			},
			expectType: NodeType("custom_group"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawContent := json.RawMessage(tt.content)
			node := &CatchallNode{
				NodeType:   tt.nodeType,
				RawContent: rawContent,
				Wrap:       tt.wrap,
				Operands:   tt.operands,
			}

			assert.Equal(t, tt.expectType, node.Type())
			if tt.wrap != nil {
				assert.Equal(t, tt.wrap, node.Wrap)
			}
			if tt.operands != nil {
				assert.Equal(t, tt.operands, node.Operands)
			}
			assert.Equal(t, rawContent, node.RawContent)
		})
	}
}

func TestComplexNestedStructures(t *testing.T) {
	// Create a complex nested structure
	innerGroup1 := &TermGroup{
		Operands: []Node{
			&Term{Foundry: "f1", Key: "k1", Layer: "l1", Match: MatchEqual},
			&Term{Foundry: "f2", Key: "k2", Layer: "l2", Match: MatchNotEqual},
		},
		Relation: AndRelation,
	}

	innerGroup2 := &TermGroup{
		Operands: []Node{
			&Term{Foundry: "f3", Key: "k3", Layer: "l3", Match: MatchEqual},
			&Term{Foundry: "f4", Key: "k4", Layer: "l4", Match: MatchEqual, Value: "test"},
		},
		Relation: OrRelation,
	}

	topGroup := &TermGroup{
		Operands: []Node{
			innerGroup1,
			innerGroup2,
			&Token{Wrap: &Term{Foundry: "f5", Key: "k5", Layer: "l5", Match: MatchEqual}},
		},
		Relation: AndRelation,
	}

	assert.Equal(t, TermGroupNode, topGroup.Type())
	assert.Len(t, topGroup.Operands, 3)
	assert.Equal(t, AndRelation, topGroup.Relation)

	// Test inner groups
	group1 := topGroup.Operands[0].(*TermGroup)
	assert.Len(t, group1.Operands, 2)
	assert.Equal(t, AndRelation, group1.Relation)

	group2 := topGroup.Operands[1].(*TermGroup)
	assert.Len(t, group2.Operands, 2)
	assert.Equal(t, OrRelation, group2.Relation)

	// Test token wrapping
	token := topGroup.Operands[2].(*Token)
	assert.NotNil(t, token.Wrap)
	assert.Equal(t, TermNode, token.Wrap.Type())
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Empty TermGroup",
			test: func(t *testing.T) {
				group := &TermGroup{
					Operands: []Node{},
					Relation: AndRelation,
				}
				assert.Empty(t, group.Operands)
				assert.Equal(t, AndRelation, group.Relation)
			},
		},
		{
			name: "Token with nil wrap",
			test: func(t *testing.T) {
				token := &Token{Wrap: nil}
				assert.Equal(t, TokenNode, token.Type())
				assert.Nil(t, token.Wrap)
			},
		},
		{
			name: "Term with empty strings",
			test: func(t *testing.T) {
				term := &Term{
					Foundry: "",
					Key:     "",
					Layer:   "",
					Match:   MatchEqual,
					Value:   "",
				}
				assert.Equal(t, TermNode, term.Type())
				assert.Empty(t, term.Foundry)
				assert.Empty(t, term.Key)
				assert.Empty(t, term.Layer)
				assert.Empty(t, term.Value)
			},
		},
		{
			name: "Complex Pattern and Replacement",
			test: func(t *testing.T) {
				// Create a complex pattern
				patternGroup := &TermGroup{
					Operands: []Node{
						&Term{Foundry: "f1", Key: "k1", Layer: "l1", Match: MatchEqual},
						&Token{Wrap: &Term{Foundry: "f2", Key: "k2", Layer: "l2", Match: MatchNotEqual}},
					},
					Relation: OrRelation,
				}
				pattern := Pattern{Root: patternGroup}

				// Create a complex replacement
				replacementGroup := &TermGroup{
					Operands: []Node{
						&Term{Foundry: "f3", Key: "k3", Layer: "l3", Match: MatchEqual},
						&Term{Foundry: "f4", Key: "k4", Layer: "l4", Match: MatchEqual},
					},
					Relation: AndRelation,
				}
				replacement := Replacement{Root: replacementGroup}

				assert.Equal(t, TermGroupNode, pattern.Root.Type())
				assert.Equal(t, TermGroupNode, replacement.Root.Type())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
