package mapper

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/config"
	"github.com/KorAP/Koral-Mapper/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
			name:      "Simple A to B mapping with rewrites",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction:   AtoB,
				AddRewrites: true,
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
					"relation": "relation:and",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"original": {
								"@type": "koral:term",
								"foundry": "opennlp",
								"key": "PIDAT",
								"layer": "p",
								"match": "match:eq"
							}
						}
					]
				}
			}`,
		},
		{
			name:      "Mapping with foundry override and rewrites",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction:   AtoB,
				FoundryB:    "custom",
				AddRewrites: true,
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
					"relation": "relation:and",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"original": {
								"@type": "koral:term",
								"foundry": "opennlp",
								"key": "PIDAT",
								"layer": "p",
								"match": "match:eq"
							}
						}
					]
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
		{
			name:      "Query with legacy rewrite field names",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@type": "koral:token",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Legacy rewrite with source instead of editor",
						"source": "LegacyEditor",
						"operation": "operation:legacy",
						"origin": "LegacySource"
					}
				],
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
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Legacy rewrite with source instead of editor",
						"editor": "LegacyEditor",
						"operation": "operation:legacy",
						"src": "LegacySource"
					}
				],
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
			name:      "Query with mixed legacy and modern rewrite fields",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@type": "koral:token",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Modern rewrite",
						"editor": "ModernEditor",
						"operation": "operation:modern",
						"original": {
							"@type": "koral:term",
							"foundry": "original",
							"key": "original-key"
						}
					},
					{
						"@type": "koral:rewrite",
						"_comment": "Legacy rewrite with precedence test",
						"editor": "PreferredEditor",
						"source": "IgnoredSource",
						"operation": "operation:precedence",
						"original": "PreferredOriginal",
						"src": "IgnoredSrc",
						"origin": "IgnoredOrigin"
					}
				],
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
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Modern rewrite",
						"editor": "ModernEditor",
						"operation": "operation:modern",
						"original": {
							"@type": "koral:term",
							"foundry": "original",
							"key": "original-key"
						}
					},
					{
						"@type": "koral:rewrite",
						"_comment": "Legacy rewrite with precedence test",
						"editor": "PreferredEditor",
						"operation": "operation:precedence",
						"original": "PreferredOriginal"
					}
				],
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse input JSON
			var inputData any
			err := json.Unmarshal([]byte(tt.input), &inputData)
			require.NoError(t, err)

			// Apply mappings
			result, err := m.ApplyQueryMappings(tt.mappingID, tt.opts, inputData)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Parse expected JSON
			var expectedData any
			err = json.Unmarshal([]byte(tt.expected), &expectedData)
			require.NoError(t, err)

			// Compare results
			assert.Equal(t, expectedData, result)
		})
	}
}

func TestTokenToTermGroupWithRewrites(t *testing.T) {
	// Create test mapping list specifically for token to termGroup test
	mappingList := config.MappingList{
		ID:       "test-token-to-termgroup",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "tt",
		LayerB:   "pos",
		Mappings: []config.MappingRule{
			"[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	input := `{
		"@type": "koral:token",
		"rewrites": [
			{
				"@type": "koral:rewrite",
				"_comment": "This rewrite should be preserved",
				"editor": "TestEditor",
				"operation": "operation:test",
				"src": "TestSource"
			}
		],
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PIDAT",
			"layer": "p",
			"match": "match:eq"
		}
	}`

	expected := `{
		"@type": "koral:token",
		"rewrites": [
			{
				"@type": "koral:rewrite",
				"_comment": "This rewrite should be preserved",
				"editor": "TestEditor",
				"operation": "operation:test",
				"src": "TestSource"
			}
		],
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
	}`

	// Parse input JSON
	var inputData any
	err = json.Unmarshal([]byte(input), &inputData)
	require.NoError(t, err)

	// Apply mappings
	result, err := m.ApplyQueryMappings("test-token-to-termgroup", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	// Parse expected JSON
	var expectedData any
	err = json.Unmarshal([]byte(expected), &expectedData)
	require.NoError(t, err)

	// Compare results
	assert.Equal(t, expectedData, result)
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

			result, err := m.ApplyQueryMappings("test-mapper", MappingOptions{Direction: AtoB}, inputData)
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

func TestMultiFieldRewritesAreReversible(t *testing.T) {
	mappingList := config.MappingList{
		ID:       "multi-field",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "pos",
		Mappings: []config.MappingRule{
			"[DET] <> [PRON]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	tests := []struct {
		name     string
		opts     MappingOptions
		input    string
		expected string
	}{
		{
			name: "Multi-field change: single rewrite with full original",
			opts: MappingOptions{
				Direction:   AtoB,
				AddRewrites: true,
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "DET",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "upos",
					"key": "PRON",
					"layer": "pos",
					"match": "match:eq",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"original": {
								"@type": "koral:term",
								"foundry": "opennlp",
								"key": "DET",
								"layer": "p",
								"match": "match:eq"
							}
						}
					]
				}
			}`,
		},
		{
			name: "Reverse direction: single rewrite with full original",
			opts: MappingOptions{
				Direction:   BtoA,
				AddRewrites: true,
			},
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "upos",
					"key": "PRON",
					"layer": "pos",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "DET",
					"layer": "p",
					"match": "match:eq",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"original": {
								"@type": "koral:term",
								"foundry": "upos",
								"key": "PRON",
								"layer": "pos",
								"match": "match:eq"
							}
						}
					]
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputData any
			err := json.Unmarshal([]byte(tt.input), &inputData)
			require.NoError(t, err)

			result, err := m.ApplyQueryMappings("multi-field", tt.opts, inputData)
			require.NoError(t, err)

			var expectedData any
			err = json.Unmarshal([]byte(tt.expected), &expectedData)
			require.NoError(t, err)

			assert.Equal(t, expectedData, result)
		})
	}
}

func TestSingleFieldRewrite(t *testing.T) {
	mappingList := config.MappingList{
		ID:       "same-fl",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "pos",
		Mappings: []config.MappingRule{
			"[DET] <> [PRON]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(`{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "DET",
			"layer": "p",
			"match": "match:eq"
		}
	}`), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyQueryMappings("same-fl", MappingOptions{
		Direction:   AtoB,
		AddRewrites: true,
	}, inputData)
	require.NoError(t, err)

	var expectedData any
	err = json.Unmarshal([]byte(`{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PRON",
			"layer": "pos",
			"match": "match:eq",
			"rewrites": [
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"original": {
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					}
				}
			]
		}
	}`), &expectedData)
	require.NoError(t, err)

	assert.Equal(t, expectedData, result)
}

func TestBuildRewritesSingleObjectRewrite(t *testing.T) {
	tests := []struct {
		name     string
		original *ast.Term
		new_     *ast.Term
	}{
		{
			name:     "All fields change",
			original: &ast.Term{Foundry: "a", Layer: "l1", Key: "k1", Value: "v1", Match: ast.MatchEqual},
			new_:     &ast.Term{Foundry: "b", Layer: "l2", Key: "k2", Value: "v2", Match: ast.MatchEqual},
		},
		{
			name:     "Single field injection: empty value becomes non-empty",
			original: &ast.Term{Foundry: "a", Layer: "l", Key: "k", Match: ast.MatchEqual},
			new_:     &ast.Term{Foundry: "a", Layer: "l", Key: "k", Value: "v", Match: ast.MatchEqual},
		},
		{
			name:     "Single field deletion: non-empty value becomes empty",
			original: &ast.Term{Foundry: "a", Layer: "l", Key: "k", Value: "v", Match: ast.MatchEqual},
			new_:     &ast.Term{Foundry: "a", Layer: "l", Key: "k", Match: ast.MatchEqual},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewrites := buildRewrites(tt.original, tt.new_)
			require.Len(t, rewrites, 1, "one rule application should produce exactly one rewrite")
			rw := rewrites[0]
			assert.Equal(t, RewriteEditor, rw.Editor)
			assert.Empty(t, rw.Scope, "object-level rewrite should have no scope")
			assert.NotNil(t, rw.Original, "rewrite should contain the full original")
			originalMap, ok := rw.Original.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "koral:term", originalMap["@type"])
		})
	}
}

func TestQueryWrapperMappings(t *testing.T) {

	mappingList := config.MappingList{
		ID:       "test-wrapper",
		FoundryA: "opennlp",
		LayerA:   "orth",
		FoundryB: "upos",
		LayerB:   "orth",
		Mappings: []config.MappingRule{
			"[opennlp/orth=Baum] <> [opennlp/orth=X]",
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
			name:      "Query wrapper case with rewrites preservation",
			mappingID: "test-wrapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"collection": {
					"@type": "koral:doc",
					"key": "availability",
					"match": "match:eq",
					"type": "type:regex",
					"value": "CC.*"
				},
				"query": {
					"@type": "koral:token",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"_comment": "Original rewrite that should be preserved",
							"editor": "Original",
							"operation": "operation:original",
							"src": "Original"
						}
					],
					"wrap": {
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "Baum",
						"layer": "orth",
						"match": "match:eq"
					}
				}
			}`,
			expected: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"collection": {
					"@type": "koral:doc",
					"key": "availability",
					"match": "match:eq",
					"type": "type:regex",
					"value": "CC.*"
				},
				"query": {
					"@type": "koral:token",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"_comment": "Original rewrite that should be preserved",
							"editor": "Original",
							"operation": "operation:original",
							"src": "Original"
						}
					],
					"wrap": {
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "X",
						"layer": "orth",
						"match": "match:eq"
					}
				}
			}`,
		},
		{
			name:      "Empty query field",
			mappingID: "test-wrapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"query": null
			}`,
			expected: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"query": null
			}`,
		},
		{
			name:      "Missing query field",
			mappingID: "test-wrapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"collection": {
					"@type": "koral:doc"
				}
			}`,
			expected: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"collection": {
					"@type": "koral:doc"
				}
			}`,
		},
		{
			name:      "Query field with non-object value",
			mappingID: "test-wrapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"query": "invalid"
			}`,
			expected: `{
				"@context": "http://korap.ids-mannheim.de/ns/KoralQuery/v0.3/context.jsonld",
				"query": "invalid"
			}`,
		},
		{
			name:      "Query with rewrites in nested token",
			mappingID: "test-wrapper",
			opts: MappingOptions{
				Direction: AtoB,
			},
			input: `{
				"@type": "koral:token",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Nested rewrite that should be preserved",
						"editor": "Nested",
						"operation": "operation:nested",
						"src": "Nested"
					}
				],
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "Baum",
					"layer": "orth",
					"match": "match:eq"
				}
			}`,
			expected: `{
				"@type": "koral:token",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Nested rewrite that should be preserved",
						"editor": "Nested",
						"operation": "operation:nested",
						"src": "Nested"
					}
				],
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "X",
					"layer": "orth",
					"match": "match:eq"
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse input JSON
			var inputData any
			err := json.Unmarshal([]byte(tt.input), &inputData)
			require.NoError(t, err)

			// Apply mappings
			result, err := m.ApplyQueryMappings(tt.mappingID, tt.opts, inputData)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Parse expected JSON
			var expectedData any
			err = json.Unmarshal([]byte(tt.expected), &expectedData)
			require.NoError(t, err)

			// Compare results
			assert.Equal(t, expectedData, result)
		})
	}
}

func TestIdenticalEffectiveFoundryLayerRejected(t *testing.T) {
	tests := []struct {
		name    string
		list    config.MappingList
		opts    MappingOptions
		wantErr string
	}{
		{
			name: "YAML defaults identical",
			list: config.MappingList{
				ID: "test", FoundryA: "opennlp", LayerA: "p",
				FoundryB: "opennlp", LayerB: "p",
				Mappings: []config.MappingRule{"[A] <> [B]"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "identical source and target",
		},
		{
			name: "Query param override makes them identical",
			list: config.MappingList{
				ID: "test", FoundryA: "opennlp", LayerA: "p",
				FoundryB: "upos", LayerB: "p",
				Mappings: []config.MappingRule{"[A] <> [B]"},
			},
			opts:    MappingOptions{Direction: AtoB, FoundryB: "opennlp"},
			wantErr: "identical source and target",
		},
		{
			name: "Query param override resolves the conflict",
			list: config.MappingList{
				ID: "test", FoundryA: "opennlp", LayerA: "p",
				FoundryB: "opennlp", LayerB: "p",
				Mappings: []config.MappingRule{"[A] <> [B]"},
			},
			opts:    MappingOptions{Direction: AtoB, FoundryB: "upos"},
			wantErr: "",
		},
		{
			name: "Different foundry same layer is allowed",
			list: config.MappingList{
				ID: "test", FoundryA: "opennlp", LayerA: "p",
				FoundryB: "upos", LayerB: "p",
				Mappings: []config.MappingRule{"[A] <> [B]"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "",
		},
		{
			name: "Both foundries empty is allowed",
			list: config.MappingList{
				ID:       "test",
				Mappings: []config.MappingRule{"[A] <> [B]"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMapper([]config.MappingList{tt.list})
			require.NoError(t, err)

			input := map[string]any{
				"@type": "koral:token",
				"wrap": map[string]any{
					"@type": "koral:term",
					"key":   "A",
				},
			}

			_, err = m.ApplyQueryMappings("test", tt.opts, input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIdenticalEffectiveFieldRejected(t *testing.T) {
	tests := []struct {
		name    string
		list    config.MappingList
		opts    MappingOptions
		wantErr string
	}{
		{
			name: "YAML defaults identical",
			list: config.MappingList{
				ID: "test", Type: "corpus",
				FieldA: "textClass", FieldB: "textClass",
				Mappings: []config.MappingRule{"novel <> fiction"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "identical source and target field",
		},
		{
			name: "Query param override makes them identical",
			list: config.MappingList{
				ID: "test", Type: "corpus",
				FieldA: "textClass", FieldB: "genre",
				Mappings: []config.MappingRule{"novel <> fiction"},
			},
			opts:    MappingOptions{Direction: AtoB, FieldB: "textClass"},
			wantErr: "identical source and target field",
		},
		{
			name: "Query param override resolves the conflict",
			list: config.MappingList{
				ID: "test", Type: "corpus",
				FieldA: "textClass", FieldB: "textClass",
				Mappings: []config.MappingRule{"novel <> fiction"},
			},
			opts:    MappingOptions{Direction: AtoB, FieldB: "genre"},
			wantErr: "",
		},
		{
			name: "Different fields is allowed",
			list: config.MappingList{
				ID: "test", Type: "corpus",
				FieldA: "textClass", FieldB: "genre",
				Mappings: []config.MappingRule{"novel <> fiction"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "",
		},
		{
			name: "Both fields empty is allowed",
			list: config.MappingList{
				ID: "test", Type: "corpus",
				Mappings: []config.MappingRule{"textClass=novel <> genre=fiction"},
			},
			opts:    MappingOptions{Direction: AtoB},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMapper([]config.MappingList{tt.list})
			require.NoError(t, err)

			input := map[string]any{
				"collection": map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "novel",
					"match": "match:eq",
				},
			}

			_, err = m.ApplyQueryMappings("test", tt.opts, input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIdenticalEffectiveValuesResponseEndpoint(t *testing.T) {
	t.Run("annotation response rejects identical effective foundry/layer", func(t *testing.T) {
		m, err := NewMapper([]config.MappingList{{
			ID: "test", FoundryA: "marmot", LayerA: "p",
			FoundryB: "marmot", LayerB: "p",
			Mappings: []config.MappingRule{"[DET] <> [PRON]"},
		}})
		require.NoError(t, err)

		input := map[string]any{
			"snippet": `<span title="marmot/p:DET">Der</span>`,
		}

		_, err = m.ApplyResponseMappings("test", MappingOptions{Direction: AtoB}, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identical source and target")
	})

	t.Run("corpus response rejects identical effective field", func(t *testing.T) {
		m, err := NewMapper([]config.MappingList{{
			ID: "test", Type: "corpus",
			FieldA: "textClass", FieldB: "textClass",
			Mappings: []config.MappingRule{"novel <> fiction"},
		}})
		require.NoError(t, err)

		input := map[string]any{
			"fields": []any{
				map[string]any{
					"@type": "koral:field",
					"key":   "textClass",
					"value": "novel",
					"type":  "type:string",
				},
			},
		}

		_, err = m.ApplyResponseMappings("test", MappingOptions{Direction: AtoB}, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identical source and target field")
	})
}

func newSTTSUPoSMapper(t *testing.T) *Mapper {
	t.Helper()
	data, err := os.ReadFile("../mappings/stts-upos.yaml")
	require.NoError(t, err, "failed to read stts-upos.yaml from disk")

	var mappingList config.MappingList
	err = yaml.Unmarshal(data, &mappingList)
	require.NoError(t, err, "failed to parse stts-upos.yaml")

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)
	return m
}

func TestFallbackRules(t *testing.T) {
	m := newSTTSUPoSMapper(t)

	t.Run("Bare ADJ (BtoA) maps to ADJA|ADJD disjunction", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "ADJ",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
		operands := wrap["operands"].([]any)
		assert.Len(t, operands, 2)
		keys := []string{
			operands[0].(map[string]any)["key"].(string),
			operands[1].(map[string]any)["key"].(string),
		}
		assert.Contains(t, keys, "ADJA")
		assert.Contains(t, keys, "ADJD")
	})

	t.Run("ADJ & Variant=Short (BtoA) maps to ADJD only", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "ADJ",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "Short",
						"layer": "Variant",
						"match": "match:eq"
					}
				],
				"relation": "relation:and"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ADJD", wrap["key"])
	})

	t.Run("Bare DET (BtoA) maps to DET subtypes disjunction", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "DET",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
		operands := wrap["operands"].([]any)
		assert.Len(t, operands, 7)
		var keys []string
		for _, op := range operands {
			keys = append(keys, op.(map[string]any)["key"].(string))
		}
		assert.Contains(t, keys, "ART")
		assert.Contains(t, keys, "PDAT")
		assert.Contains(t, keys, "PWAT")
	})

	t.Run("DET & PronType=Art (BtoA) maps to ART only", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "Art",
						"layer": "PronType",
						"match": "match:eq"
					}
				],
				"relation": "relation:and"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ART", wrap["key"])
	})

	t.Run("Bare SCONJ (BtoA) maps to KOUI|KOUS disjunction", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "SCONJ",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
		operands := wrap["operands"].([]any)
		assert.Len(t, operands, 2)
	})

	t.Run("Bare VERB (BtoA) maps to STTS verb subtypes disjunction", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "VERB",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
		operands := wrap["operands"].([]any)
		assert.Len(t, operands, 8)
	})

	t.Run("Bare AUX (BtoA) maps to AUX subtypes disjunction", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "AUX",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
		operands := wrap["operands"].([]any)
		assert.Len(t, operands, 4)
	})

	t.Run("Forward direction AtoB: ADJA maps to ADJ", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "ADJA",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: AtoB}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ADJ", wrap["key"])
	})

	t.Run("Forward direction AtoB: ART maps to DET & PronType=Art", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "ART",
				"layer": "p",
				"match": "match:eq"
			}
		}`
		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: AtoB}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:and", wrap["relation"])
	})
}

func TestOriginalProblemMultiTokenQuery(t *testing.T) {
	m := newSTTSUPoSMapper(t)

	t.Run("Multi-token [DET][ADJ][NOUN] BtoA produces correct disjunctions", func(t *testing.T) {
		// This reproduces the exact problem from the issue:
		// [upos/p=DET][upos/p=ADJ][upos/p=NOUN] mapped B->A
		input := `{
			"@type": "koral:group",
			"operation": "operation:sequence",
			"operands": [
				{
					"@type": "koral:token",
					"wrap": {
						"@type": "koral:term",
						"foundry": "upos",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					}
				},
				{
					"@type": "koral:token",
					"wrap": {
						"@type": "koral:term",
						"foundry": "upos",
						"key": "ADJ",
						"layer": "p",
						"match": "match:eq"
					}
				},
				{
					"@type": "koral:token",
					"wrap": {
						"@type": "koral:term",
						"foundry": "upos",
						"key": "NOUN",
						"layer": "p",
						"match": "match:eq"
					}
				}
			]
		}`

		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		operands := resultMap["operands"].([]any)
		require.Len(t, operands, 3)

		// Token 1: DET -> ART | PDAT | PIAT | PIDAT | PPOSAT | PRELAT | PWAT
		token1 := operands[0].(map[string]any)
		wrap1 := token1["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap1["@type"], "DET should be mapped to OR group")
		assert.Equal(t, "relation:or", wrap1["relation"])
		ops1 := wrap1["operands"].([]any)
		assert.Len(t, ops1, 7, "DET fallback should have 7 alternatives")

		// Token 2: ADJ -> ADJA | ADJD
		token2 := operands[1].(map[string]any)
		wrap2 := token2["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap2["@type"], "ADJ should be mapped to OR group")
		assert.Equal(t, "relation:or", wrap2["relation"])
		ops2 := wrap2["operands"].([]any)
		assert.Len(t, ops2, 2, "ADJ fallback should have 2 alternatives")

		adjKeys := []string{
			ops2[0].(map[string]any)["key"].(string),
			ops2[1].(map[string]any)["key"].(string),
		}
		assert.Contains(t, adjKeys, "ADJA")
		assert.Contains(t, adjKeys, "ADJD")

		// Token 3: NOUN -> NN (specific rule, not fallback, because
		// [NN] <> [NOUN] has specificity 1 and [NN | NE] <> [NOUN | PROPN]
		// has pattern specificity 0 on B-side (OR group))
		token3 := operands[2].(map[string]any)
		wrap3 := token3["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap3["@type"], "NOUN should map to single NN term")
		assert.Equal(t, "NN", wrap3["key"])
	})

	t.Run("Specific input [ADJ & Variant=Short] maps to ADJD only", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "ADJ",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "Short",
						"layer": "Variant",
						"match": "match:eq"
					}
				],
				"relation": "relation:and"
			}
		}`

		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ADJD", wrap["key"])
	})

	t.Run("Specific input [DET & PronType=Art] maps to ART only", func(t *testing.T) {
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "Art",
						"layer": "PronType",
						"match": "match:eq"
					}
				],
				"relation": "relation:and"
			}
		}`

		var inputData any
		err := json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("stts-upos", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ART", wrap["key"])
	})
}

func TestSpecificityBasedRuleSelection(t *testing.T) {
	t.Run("More specific rule wins over less specific", func(t *testing.T) {
		mappingList := config.MappingList{
			ID:       "spec-test",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []config.MappingRule{
				"[ADJA] <> [ADJ]",
				"[ADJD] <> [ADJ & Variant=Short]",
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		// Input: ADJ & Variant=Short — matches the internal representation
		// where "Variant=Short" is parsed as layer="Variant", key="Short"
		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "ADJ",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "upos",
						"key": "Short",
						"layer": "Variant",
						"match": "match:eq"
					}
				],
				"relation": "relation:and"
			}
		}`

		var inputData any
		err = json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("spec-test", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:term", wrap["@type"])
		assert.Equal(t, "ADJD", wrap["key"])
	})

	t.Run("Same specificity - first rule in file order wins", func(t *testing.T) {
		mappingList := config.MappingList{
			ID:       "tie-test",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []config.MappingRule{
				"[KOUI] <> [SCONJ]",
				"[KOUS] <> [SCONJ]",
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "SCONJ",
				"layer": "p",
				"match": "match:eq"
			}
		}`

		var inputData any
		err = json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("tie-test", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "KOUI", wrap["key"])
	})

	t.Run("Single matching rule - identical to first-match-wins", func(t *testing.T) {
		mappingList := config.MappingList{
			ID:       "single-test",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []config.MappingRule{
				"[NN] <> [NOUN]",
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "NOUN",
				"layer": "p",
				"match": "match:eq"
			}
		}`

		var inputData any
		err = json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("single-test", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "NN", wrap["key"])
	})

	t.Run("No matching rule - node passes through unchanged", func(t *testing.T) {
		mappingList := config.MappingList{
			ID:       "nomatch-test",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []config.MappingRule{
				"[NN] <> [NOUN]",
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "VERB",
				"layer": "p",
				"match": "match:eq"
			}
		}`

		var inputData any
		err = json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("nomatch-test", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "VERB", wrap["key"])
	})

	t.Run("Fallback OR-disjunction rule loses to specific rule", func(t *testing.T) {
		mappingList := config.MappingList{
			ID:       "fallback-test",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []config.MappingRule{
				"[ADJA] <> [ADJ]",
				"[ADJA | ADJD] <> [ADJ]",
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		input := `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "upos",
				"key": "ADJ",
				"layer": "p",
				"match": "match:eq"
			}
		}`

		var inputData any
		err = json.Unmarshal([]byte(input), &inputData)
		require.NoError(t, err)

		result, err := m.ApplyQueryMappings("fallback-test", MappingOptions{Direction: BtoA}, inputData)
		require.NoError(t, err)

		// Both rules match with pattern specificity 1 on B-side.
		// Rule 1 replacement specificity = 1 (Term), Rule 2 replacement specificity = 0 (OR group).
		// Lower replacement specificity wins (broader/fallback output) => rule 2 wins.
		resultMap := result.(map[string]any)
		wrap := resultMap["wrap"].(map[string]any)
		assert.Equal(t, "koral:termGroup", wrap["@type"])
		assert.Equal(t, "relation:or", wrap["relation"])
	})
}
