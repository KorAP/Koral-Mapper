package matcher

import (
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/stretchr/testify/assert"
)

func TestMatchSimplePattern(t *testing.T) {
	// Create a simple pattern: match a term with DET
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
		},
	}

	// Create a simple replacement
	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "COMBINED_DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
		},
	}

	m := NewMatcher(pattern, replacement)

	tests := []struct {
		name     string
		input    ast.Node
		expected bool
	}{
		{
			name: "Exact match",
			input: &ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			expected: true,
		},
		{
			name: "Different key",
			input: &ast.Term{
				Foundry: "opennlp",
				Key:     "NOUN",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			expected: false,
		},
		{
			name: "Different foundry",
			input: &ast.Term{
				Foundry: "different",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			expected: false,
		},
		{
			name: "Different match type",
			input: &ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchNotEqual,
			},
			expected: false,
		},
		{
			name: "Wrong node type",
			input: &ast.Token{
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Match(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchComplexPattern(t *testing.T) {
	// Create a complex pattern: DET AND (AdjType=Pdt OR PronType=Ind)
	pattern := ast.Pattern{
		Root: &ast.Token{
			Wrap: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "DET",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
					&ast.TermGroup{
						Operands: []ast.Node{
							&ast.Term{
								Foundry: "opennlp",
								Key:     "AdjType",
								Layer:   "m",
								Match:   ast.MatchEqual,
								Value:   "Pdt",
							},
							&ast.Term{
								Foundry: "opennlp",
								Key:     "PronType",
								Layer:   "m",
								Match:   ast.MatchEqual,
								Value:   "Ind",
							},
						},
						Relation: ast.OrRelation,
					},
				},
				Relation: ast.AndRelation,
			},
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Token{
			Wrap: &ast.Term{
				Foundry: "opennlp",
				Key:     "COMBINED_DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
		},
	}

	m := NewMatcher(pattern, replacement)

	tests := []struct {
		name     string
		input    ast.Node
		expected bool
	}{
		{
			name: "Match with AdjType=Pdt",
			input: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "DET",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "AdjType",
							Layer:   "m",
							Match:   ast.MatchEqual,
							Value:   "Pdt",
						},
					},
					Relation: ast.AndRelation,
				},
			},
			expected: true,
		},
		{
			name: "Match with PronType=Ind",
			input: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "DET",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "PronType",
							Layer:   "m",
							Match:   ast.MatchEqual,
							Value:   "Ind",
						},
					},
					Relation: ast.AndRelation,
				},
			},
			expected: true,
		},
		{
			name: "No match - missing DET",
			input: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "NOUN",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "AdjType",
							Layer:   "m",
							Match:   ast.MatchEqual,
							Value:   "Pdt",
						},
					},
					Relation: ast.AndRelation,
				},
			},
			expected: false,
		},
		{
			name: "No match - wrong value",
			input: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "DET",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "AdjType",
							Layer:   "m",
							Match:   ast.MatchEqual,
							Value:   "Wrong",
						},
					},
					Relation: ast.AndRelation,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Match(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplace(t *testing.T) {
	// Create pattern and replacement
	pattern := ast.Pattern{
		Root: &ast.TermGroup{
			Operands: []ast.Node{
				&ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
				&ast.Term{
					Foundry: "opennlp",
					Key:     "AdjType",
					Layer:   "m",
					Match:   ast.MatchEqual,
					Value:   "Pdt",
				},
			},
			Relation: ast.AndRelation,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "COMBINED_DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
		},
	}

	m := NewMatcher(pattern, replacement)

	tests := []struct {
		name     string
		input    ast.Node
		expected ast.Node
	}{
		{
			name: "Replace matching pattern",
			input: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "DET",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
					&ast.Term{
						Foundry: "opennlp",
						Key:     "AdjType",
						Layer:   "m",
						Match:   ast.MatchEqual,
						Value:   "Pdt",
					},
				},
				Relation: ast.AndRelation,
			},
			expected: &ast.Term{
				Foundry: "opennlp",
				Key:     "COMBINED_DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
		},
		{
			name: "No replacement for non-matching pattern",
			input: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "NOUN",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
					&ast.Term{
						Foundry: "opennlp",
						Key:     "AdjType",
						Layer:   "m",
						Match:   ast.MatchEqual,
						Value:   "Pdt",
					},
				},
				Relation: ast.AndRelation,
			},
			expected: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "NOUN",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
					&ast.Term{
						Foundry: "opennlp",
						Key:     "AdjType",
						Layer:   "m",
						Match:   ast.MatchEqual,
						Value:   "Pdt",
					},
				},
				Relation: ast.AndRelation,
			},
		},
		{
			name: "Replace in nested structure",
			input: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.TermGroup{
							Operands: []ast.Node{
								&ast.Term{
									Foundry: "opennlp",
									Key:     "DET",
									Layer:   "p",
									Match:   ast.MatchEqual,
								},
								&ast.Term{
									Foundry: "opennlp",
									Key:     "AdjType",
									Layer:   "m",
									Match:   ast.MatchEqual,
									Value:   "Pdt",
								},
							},
							Relation: ast.AndRelation,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "NOUN",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
					},
					Relation: ast.AndRelation,
				},
			},
			expected: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "COMBINED_DET",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "NOUN",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
					},
					Relation: ast.AndRelation,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Replace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchNodeOrder(t *testing.T) {
	// Test that operands can match in any order
	pattern := ast.Pattern{
		Root: &ast.TermGroup{
			Operands: []ast.Node{
				&ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
				&ast.Term{
					Foundry: "opennlp",
					Key:     "AdjType",
					Layer:   "m",
					Match:   ast.MatchEqual,
					Value:   "Pdt",
				},
			},
			Relation: ast.AndRelation,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "COMBINED_DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
		},
	}

	m := NewMatcher(pattern, replacement)

	// Test with operands in different orders
	input1 := &ast.TermGroup{
		Operands: []ast.Node{
			&ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			&ast.Term{
				Foundry: "opennlp",
				Key:     "AdjType",
				Layer:   "m",
				Match:   ast.MatchEqual,
				Value:   "Pdt",
			},
		},
		Relation: ast.AndRelation,
	}

	input2 := &ast.TermGroup{
		Operands: []ast.Node{
			&ast.Term{
				Foundry: "opennlp",
				Key:     "AdjType",
				Layer:   "m",
				Match:   ast.MatchEqual,
				Value:   "Pdt",
			},
			&ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
		},
		Relation: ast.AndRelation,
	}

	assert.True(t, m.Match(input1), "Should match with original order")
	assert.True(t, m.Match(input2), "Should match with reversed order")
}
