package mapper

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/KorAP/KoralPipe-TermMapper/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapper(t *testing.T) {
	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-mapper",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
			"[DET] <> [opennlp/p=DET]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	tests := []struct {
		name        string
		mappingID   string
		opts        MappingOptions
		input       string
		expected    string
		expectError bool
	}{
		{
			name:      "Simple A to B mapping",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:termGroup",
					"operands": [
						{
							"@type": "koral:term",
							"foundry": "opennlp",
							"key": "PIDAT",
							"layer": "p",
							"match": "match:eq"
						},
						{
							"@type": "koral:term",
							"foundry": "opennlp",
							"key": "AdjType",
							"layer": "p",
							"match": "match:eq",
							"value": "Pdt"
						}
					],
					"relation": "relation:and"
				}
			}`,
		},
		{
			name:      "B to A direction",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: BtoA,
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expectError: false,
		},
		{
			name:      "Mapping with foundry override",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: AtoB,
				FoundryB:  "custom",
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:termGroup",
					"operands": [
						{
							"@type": "koral:term",
							"foundry": "custom",
							"key": "PIDAT",
							"layer": "p",
							"match": "match:eq"
						},
						{
							"@type": "koral:term",
							"foundry": "custom",
							"key": "AdjType",
							"layer": "p",
							"match": "match:eq",
							"value": "Pdt"
						}
					],
					"relation": "relation:and"
				}
			}`,
		},
		{
			name:      "Invalid mapping ID",
			mappingID: "nonexistent",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expectError: true,
		},
		{
			name:      "Invalid direction",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: Direction(false),
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse input JSON
			var inputData interface{}
			err := json.Unmarshal([]byte(tt.input), &inputData)
			require.NoError(t, err)

			// Apply mappings
			result, err := m.ApplyMappings(tt.mappingID, tt.opts, inputData)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Parse expected JSON
			var expectedData interface{}
			err = json.Unmarshal([]byte(tt.expected), &expectedData)
			require.NoError(t, err)

			// Compare results
			assert.Equal(t, expectedData, result)
		})
	}
}

func TestMatchComplexPatterns(t *testing.T) {
	tests := []struct {
		name        string
		pattern     ast.Pattern
		replacement ast.Replacement
		input       ast.Node
		expected    ast.Node
	}{
		{
			name: "Deep nested pattern with mixed operators",
			pattern: ast.Pattern{
				Root: &ast.TermGroup{
					Operands: []ast.Node{
						&ast.Term{
							Key:   "A",
							Match: ast.MatchEqual,
						},
						&ast.TermGroup{
							Operands: []ast.Node{
								&ast.Term{
									Key:   "B",
									Match: ast.MatchEqual,
								},
								&ast.TermGroup{
									Operands: []ast.Node{
										&ast.Term{
											Key:   "C",
											Match: ast.MatchEqual,
										},
										&ast.Term{
											Key:   "D",
											Match: ast.MatchEqual,
										},
									},
									Relation: ast.AndRelation,
								},
							},
							Relation: ast.OrRelation,
						},
					},
					Relation: ast.AndRelation,
				},
			},
			replacement: ast.Replacement{
				Root: &ast.Term{
					Key:   "RESULT",
					Match: ast.MatchEqual,
				},
			},
			input: &ast.TermGroup{
				Operands: []ast.Node{
					&ast.Term{
						Key:   "A",
						Match: ast.MatchEqual,
					},
					&ast.TermGroup{
						Operands: []ast.Node{
							&ast.Term{
								Key:   "C",
								Match: ast.MatchEqual,
							},
							&ast.Term{
								Key:   "D",
								Match: ast.MatchEqual,
							},
						},
						Relation: ast.AndRelation,
					},
				},
				Relation: ast.AndRelation,
			},
			expected: &ast.Term{
				Key:   "RESULT",
				Match: ast.MatchEqual,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := matcher.NewMatcher(tt.pattern, tt.replacement)
			require.NoError(t, err)
			result := m.Replace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInvalidPatternReplacement(t *testing.T) {
	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-mapper",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Invalid input - empty term group",
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:termGroup",
					"operands": [],
					"relation": "relation:and"
				}
			}`,
			expectError: true,
			errorMsg:    "failed to parse JSON into AST: error parsing wrapped node: term group must have at least one operand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputData any
			err := json.Unmarshal([]byte(tt.input), &inputData)
			require.NoError(t, err)

			result, err := m.ApplyMappings("test-mapper", MappingOptions{Direction: AtoB}, inputData)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
