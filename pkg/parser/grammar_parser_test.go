package parser

import (
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrammarParserSimpleTerm(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		defaultFoundry string
		defaultLayer   string
		expected       *SimpleTerm
		expectError    bool
	}{
		{
			name:           "Foundry layer key value",
			input:          "[opennlp/p=PIDAT:new]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				WithFoundryLayer: &FoundryLayerTerm{
					Foundry: "opennlp",
					Layer:   "p",
					Key:     "PIDAT",
					Value:   "new",
				},
			},
		},
		{
			name:           "Foundry layer key",
			input:          "[opennlp/p=PIDAT]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				WithFoundryLayer: &FoundryLayerTerm{
					Foundry: "opennlp",
					Layer:   "p",
					Key:     "PIDAT",
				},
			},
		},
		{
			name:           "Layer key",
			input:          "[p=PIDAT]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				WithLayer: &LayerTerm{
					Layer: "p",
					Key:   "PIDAT",
				},
			},
		},
		{
			name:           "Simple key",
			input:          "[PIDAT]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				SimpleKey: &KeyTerm{
					Key: "PIDAT",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewGrammarParser(tt.defaultFoundry, tt.defaultLayer)
			require.NoError(t, err)

			grammar, err := parser.parser.ParseString("", tt.input)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, grammar.Token.Expr.First.Simple)
		})
	}
}

func TestGrammarParser(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		defaultFoundry string
		defaultLayer   string
		expected       ast.Node
		expectError    bool
	}{
		{
			name:           "Simple term with foundry and layer",
			input:          "[opennlp/p=PIDAT]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &ast.Token{
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "PIDAT",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
		},
		{
			name:           "Term group with and relation",
			input:          "[opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "PIDAT",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "AdjType",
							Layer:   "p",
							Match:   ast.MatchEqual,
							Value:   "Pdt",
						},
					},
					Relation: ast.AndRelation,
				},
			},
		},
		{
			name:           "Term group with or relation",
			input:          "[opennlp/p=PronType:Ind | opennlp/p=PronType:Neg]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "PronType",
							Layer:   "p",
							Match:   ast.MatchEqual,
							Value:   "Ind",
						},
						&ast.Term{
							Foundry: "opennlp",
							Key:     "PronType",
							Layer:   "p",
							Match:   ast.MatchEqual,
							Value:   "Neg",
						},
					},
					Relation: ast.OrRelation,
				},
			},
		},
		{
			name:           "Complex term group",
			input:          "[opennlp/p=PIDAT & (opennlp/p=PronType:Ind | opennlp/p=PronType:Neg)]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &ast.Token{
				Wrap: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Foundry: "opennlp",
							Key:     "PIDAT",
							Layer:   "p",
							Match:   ast.MatchEqual,
						},
						&ast.TermGroup{
							Operands: []ast.Node{
								&ast.Term{
									Foundry: "opennlp",
									Key:     "PronType",
									Layer:   "p",
									Match:   ast.MatchEqual,
									Value:   "Ind",
								},
								&ast.Term{
									Foundry: "opennlp",
									Key:     "PronType",
									Layer:   "p",
									Match:   ast.MatchEqual,
									Value:   "Neg",
								},
							},
							Relation: ast.OrRelation,
						},
					},
					Relation: ast.AndRelation,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewGrammarParser(tt.defaultFoundry, tt.defaultLayer)
			require.NoError(t, err)

			result, err := parser.Parse(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
