package ast

import (
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
