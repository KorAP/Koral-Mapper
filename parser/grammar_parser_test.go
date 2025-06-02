package parser

import (
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
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
		{
			name:           "Special symbol",
			input:          "[$\\(]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				SimpleKey: &KeyTerm{
					Key: "$(",
				},
			},
		},
		{
			name:           "Multiple escaped characters",
			input:          "[\\&\\|\\=]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				SimpleKey: &KeyTerm{
					Key: "&|=",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewGrammarParser(tt.defaultFoundry, tt.defaultLayer)
			require.NoError(t, err)

			grammar, err := parser.tokenParser.ParseString("", tt.input)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, grammar.Token, "Expected token expression")

			// For testing purposes, unescape the key in the simple term
			if grammar.Token.Expr.First.Simple.SimpleKey != nil {
				grammar.Token.Expr.First.Simple.SimpleKey.Key = unescapeString(grammar.Token.Expr.First.Simple.SimpleKey.Key)
			}

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

func TestMappingRules(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *MappingResult
		wantErr  bool
	}{
		{
			name:  "Simple PIDAT mapping",
			input: "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PIDAT",
						Match: ast.MatchEqual,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.TermGroup{
						Relation: ast.AndRelation,
						Operands: []ast.Node{
							&ast.Term{
								Foundry: "opennlp",
								Layer:   "p",
								Key:     "PIDAT",
								Match:   ast.MatchEqual,
							},
							&ast.Term{
								Foundry: "opennlp",
								Layer:   "p",
								Key:     "AdjType",
								Value:   "Pdt",
								Match:   ast.MatchEqual,
							},
						},
					},
				},
			},
		},
		{
			name:  "PAV mapping",
			input: "[PAV] <> [ADV & PronType:Dem]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PAV",
						Match: ast.MatchEqual,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.TermGroup{
						Relation: ast.AndRelation,
						Operands: []ast.Node{
							&ast.Term{
								Key:   "ADV",
								Match: ast.MatchEqual,
							},
							&ast.Term{
								Key:   "PronType",
								Value: "Dem",
								Match: ast.MatchEqual,
							},
						},
					},
				},
			},
		},
		{
			name:    "Invalid mapping syntax",
			input:   "[PAV] -> [ADV]",
			wantErr: true,
		},
		{
			name:    "Missing closing bracket",
			input:   "[PAV <> [ADV]",
			wantErr: true,
		},
	}

	parser, err := NewGrammarParser("", "")
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseMapping(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
