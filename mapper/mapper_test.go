package mapper

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/config"
	"github.com/KorAP/Koral-Mapper/matcher"
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
			name: "Multi-field change: foundry + layer + key all change",
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
							"scope": "foundry",
							"original": "opennlp"
						},
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"scope": "layer",
							"original": "p"
						},
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"scope": "key",
							"original": "DET"
						}
					]
				}
			}`,
		},
		{
			name: "Reverse direction: foundry + layer + key all change back",
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
							"scope": "foundry",
							"original": "upos"
						},
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"scope": "layer",
							"original": "pos"
						},
						{
							"@type": "koral:rewrite",
							"editor": "Koral-Mapper",
							"scope": "key",
							"original": "PRON"
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
					"scope": "layer",
					"original": "p"
				},
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"scope": "key",
					"original": "DET"
				}
			]
		}
	}`), &expectedData)
	require.NoError(t, err)

	assert.Equal(t, expectedData, result)
}

func TestBuildRewritesFieldInjection(t *testing.T) {
	tests := []struct {
		name           string
		original       *ast.Term
		new_           *ast.Term
		expectedScopes []string
		hasOriginals   []bool
	}{
		{
			name:           "All fields change with originals",
			original:       &ast.Term{Foundry: "a", Layer: "l1", Key: "k1", Value: "v1", Match: ast.MatchEqual},
			new_:           &ast.Term{Foundry: "b", Layer: "l2", Key: "k2", Value: "v2", Match: ast.MatchEqual},
			expectedScopes: []string{"foundry", "layer", "key", "value"},
			hasOriginals:   []bool{true, true, true, true},
		},
		{
			name:           "Injection: empty value becomes non-empty",
			original:       &ast.Term{Foundry: "a", Layer: "l", Key: "k", Match: ast.MatchEqual},
			new_:           &ast.Term{Foundry: "a", Layer: "l", Key: "k", Value: "v", Match: ast.MatchEqual},
			expectedScopes: []string{"value"},
			hasOriginals:   []bool{false},
		},
		{
			name:           "Deletion: non-empty value becomes empty",
			original:       &ast.Term{Foundry: "a", Layer: "l", Key: "k", Value: "v", Match: ast.MatchEqual},
			new_:           &ast.Term{Foundry: "a", Layer: "l", Key: "k", Match: ast.MatchEqual},
			expectedScopes: []string{"value"},
			hasOriginals:   []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewrites := buildRewrites(tt.original, tt.new_)
			require.Len(t, rewrites, len(tt.expectedScopes))
			for i, rw := range rewrites {
				assert.Equal(t, RewriteEditor, rw.Editor)
				assert.Equal(t, tt.expectedScopes[i], rw.Scope)
				if tt.hasOriginals[i] {
					assert.NotNil(t, rw.Original, "expected original for scope %s", tt.expectedScopes[i])
				} else {
					assert.Nil(t, rw.Original, "expected no original for scope %s (injection)", tt.expectedScopes[i])
				}
			}
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
