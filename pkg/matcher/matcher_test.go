package matcher

// matcher is a function that takes a pattern and a node and returns true if the node matches the pattern.
// It is used to match a pattern against a node in the AST.

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/stretchr/testify/assert"
)

func TestNewMatcherValidation(t *testing.T) {
	tests := []struct {
		name          string
		pattern       ast.Pattern
		replacement   ast.Replacement
		expectedError string
	}{
		{
			name: "Valid pattern and replacement",
			pattern: ast.Pattern{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			replacement: ast.Replacement{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "COMBINED_DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expectedError: "",
		},
		{
			name: "Invalid pattern - CatchallNode",
			pattern: ast.Pattern{
				Root: &ast.CatchallNode{
					NodeType: "custom",
				},
			},
			replacement: ast.Replacement{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expectedError: "invalid pattern: catchall nodes are not allowed in pattern/replacement ASTs",
		},
		{
			name: "Invalid replacement - CatchallNode",
			pattern: ast.Pattern{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			replacement: ast.Replacement{
				Root: &ast.CatchallNode{
					NodeType: "custom",
				},
			},
			expectedError: "invalid replacement: catchall nodes are not allowed in pattern/replacement ASTs",
		},
		{
			name: "Invalid pattern - Empty TermGroup",
			pattern: ast.Pattern{
				Root: &ast.TermGroup{
					Operands: []ast.Node{},
					Relation: ast.AndRelation,
				},
			},
			replacement: ast.Replacement{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expectedError: "invalid pattern: empty term group",
		},
		{
			name: "Invalid pattern - Nested CatchallNode",
			pattern: ast.Pattern{
				Root: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "DET",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.CatchallNode{
							NodeType: "custom",
						},
					},
					Relation: ast.AndRelation,
				},
			},
			replacement: ast.Replacement{
				Root: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expectedError: "invalid pattern: invalid operand: catchall nodes are not allowed in pattern/replacement ASTs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewMatcher(tt.pattern, tt.replacement)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, matcher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
			}
		})
	}
}

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

	m, err := NewMatcher(pattern, replacement)
	assert.NoError(t, err)
	assert.NotNil(t, m)

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
			name: "Nested node",
			input: &ast.Token{
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expected: true,
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

	m, err := NewMatcher(pattern, replacement)
	assert.NoError(t, err)
	assert.NotNil(t, m)

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
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
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

	m, err := NewMatcher(pattern, replacement)
	assert.NoError(t, err)
	assert.NotNil(t, m)

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
			expected: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "COMBINED_DET",
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

	m, err := NewMatcher(pattern, replacement)
	assert.NoError(t, err)
	assert.NotNil(t, m)

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

func TestMatchWithUnknownNodes(t *testing.T) {
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "opennlp",
			Key:     "DET",
			Layer:   "p",
			Match:   ast.MatchEqual,
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

	m, err := NewMatcher(pattern, replacement)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	tests := []struct {
		name     string
		input    ast.Node
		expected bool
	}{
		{
			name: "Match term inside unknown node with wrap",
			input: &ast.CatchallNode{
				NodeType: "koral:custom",
				RawContent: json.RawMessage(`{
					"@type": "koral:custom",
					"customField": "value"
				}`),
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expected: true,
		},
		{
			name: "Match term inside unknown node's operands",
			input: &ast.CatchallNode{
				NodeType: "koral:custom",
				RawContent: json.RawMessage(`{
					"@type": "koral:custom",
					"customField": "value"
				}`),
				Operands: []ast.Node{
					&ast.Term{
						Foundry: "opennlp",
						Key:     "DET",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
				},
			},
			expected: true,
		},
		{
			name: "No match in unknown node with different term",
			input: &ast.CatchallNode{
				NodeType: "koral:custom",
				RawContent: json.RawMessage(`{
					"@type": "koral:custom",
					"customField": "value"
				}`),
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "NOUN",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			expected: false,
		},
		{
			name: "Match in deeply nested unknown nodes",
			input: &ast.CatchallNode{
				NodeType: "koral:outer",
				RawContent: json.RawMessage(`{
					"@type": "koral:outer",
					"outerField": "value"
				}`),
				Wrap: &ast.CatchallNode{
					NodeType: "koral:inner",
					RawContent: json.RawMessage(`{
						"@type": "koral:inner",
						"innerField": "value"
					}`),
					Wrap: &ast.Term{
						Foundry: "opennlp",
						Key:     "DET",
						Layer:   "p",
						Match:   ast.MatchEqual,
					},
				},
			},
			expected: true,
		},
		{
			name: "Match in mixed known and unknown nodes",
			input: &ast.Token{
				Wrap: &ast.CatchallNode{
					NodeType: "koral:custom",
					RawContent: json.RawMessage(`{
						"@type": "koral:custom",
						"customField": "value"
					}`),
					Operands: []ast.Node{
						&ast.TermGroup{
							Operands: []ast.Node{
								&ast.Term{
									Foundry: "opennlp",
									Key:     "DET",
									Layer:   "p",
									Match:   ast.MatchEqual,
								},
							},
							Relation: ast.AndRelation,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Match(tt.input)
			assert.Equal(t, tt.expected, result)

			if tt.expected {
				// Test replacement when there's a match
				replaced := m.Replace(tt.input)
				// Verify the replacement happened somewhere in the structure
				containsReplacement := false
				var checkNode func(ast.Node)
				checkNode = func(node ast.Node) {
					switch n := node.(type) {
					case *ast.Term:
						if n.Key == "COMBINED_DET" {
							containsReplacement = true
						}
					case *ast.Token:
						if n.Wrap != nil {
							checkNode(n.Wrap)
						}
					case *ast.TermGroup:
						for _, op := range n.Operands {
							checkNode(op)
						}
					case *ast.CatchallNode:
						if n.Wrap != nil {
							checkNode(n.Wrap)
						}
						for _, op := range n.Operands {
							checkNode(op)
						}
					}
				}
				checkNode(replaced)
				assert.True(t, containsReplacement, "Replacement should be found in the result")
			}
		})
	}
}
