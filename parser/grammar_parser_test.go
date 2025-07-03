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
		{
			name:           "Foundry wildcard key",
			input:          "[opennlp/*=PIDAT]",
			defaultFoundry: "opennlp",
			defaultLayer:   "p",
			expected: &SimpleTerm{
				WithFoundryWildcard: &FoundryWildcardTerm{
					Foundry: "opennlp",
					Key:     "PIDAT",
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

// TestGrammarParser was removed as the Parse method is no longer supported
// The functionality is now only available through ParseMapping method

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
			name:  "PAV mapping with special character",
			input: "[$\\(] <> [ADV & PronType:Dem]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.Term{
						Key:   "$(",
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
		// Additional tests to cover functionality from removed TestGrammarParser
		{
			name:  "Simple term with foundry and layer",
			input: "[opennlp/p=PIDAT] <> [PIDAT]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.Term{
						Foundry: "opennlp",
						Layer:   "p",
						Key:     "PIDAT",
						Match:   ast.MatchEqual,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PIDAT",
						Match: ast.MatchEqual,
					},
				},
			},
		},
		{
			name:  "Term group with and relation",
			input: "[opennlp/p=PIDAT & opennlp/p=AdjType:Pdt] <> [PIDAT]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.TermGroup{
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
						Relation: ast.AndRelation,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PIDAT",
						Match: ast.MatchEqual,
					},
				},
			},
		},
		{
			name:  "Term group with or relation",
			input: "[opennlp/p=PronType:Ind | opennlp/p=PronType:Neg] <> [PRON]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.TermGroup{
						Operands: []ast.Node{
							&ast.Term{
								Foundry: "opennlp",
								Layer:   "p",
								Key:     "PronType",
								Value:   "Ind",
								Match:   ast.MatchEqual,
							},
							&ast.Term{
								Foundry: "opennlp",
								Layer:   "p",
								Key:     "PronType",
								Value:   "Neg",
								Match:   ast.MatchEqual,
							},
						},
						Relation: ast.OrRelation,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PRON",
						Match: ast.MatchEqual,
					},
				},
			},
		},
		{
			name:  "Complex term group with nested parentheses",
			input: "[opennlp/p=PIDAT & (opennlp/p=PronType:Ind | opennlp/p=PronType:Neg)] <> [COMPLEX]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.TermGroup{
						Operands: []ast.Node{
							&ast.Term{
								Foundry: "opennlp",
								Layer:   "p",
								Key:     "PIDAT",
								Match:   ast.MatchEqual,
							},
							&ast.TermGroup{
								Operands: []ast.Node{
									&ast.Term{
										Foundry: "opennlp",
										Layer:   "p",
										Key:     "PronType",
										Value:   "Ind",
										Match:   ast.MatchEqual,
									},
									&ast.Term{
										Foundry: "opennlp",
										Layer:   "p",
										Key:     "PronType",
										Value:   "Neg",
										Match:   ast.MatchEqual,
									},
								},
								Relation: ast.OrRelation,
							},
						},
						Relation: ast.AndRelation,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.Term{
						Key:   "COMPLEX",
						Match: ast.MatchEqual,
					},
				},
			},
		},
		{
			name:  "Wildcard pattern",
			input: "[opennlp/*=PIDAT] <> [PIDAT]",
			expected: &MappingResult{
				Upper: &ast.Token{
					Wrap: &ast.Term{
						Foundry: "opennlp",
						Layer:   "",
						Key:     "PIDAT",
						Match:   ast.MatchEqual,
					},
				},
				Lower: &ast.Token{
					Wrap: &ast.Term{
						Key:   "PIDAT",
						Match: ast.MatchEqual,
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
			assert.NoError(t, err, "Input: %s", tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
