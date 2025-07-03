package parser

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// normalizeJSON normalizes JSON by parsing and re-marshaling it
func normalizeJSON(t *testing.T, data json.RawMessage) json.RawMessage {
	var v any
	err := json.Unmarshal(data, &v)
	require.NoError(t, err)

	// Convert to canonical form (sorted keys, no whitespace)
	normalized, err := json.Marshal(v)
	require.NoError(t, err)
	return normalized
}

// compareNodes compares two AST nodes, normalizing JSON content in CatchallNodes
func compareNodes(t *testing.T, expected, actual ast.Node) bool {
	// If both nodes are CatchallNodes, normalize their JSON content before comparison
	if expectedCatchall, ok := expected.(*ast.CatchallNode); ok {
		if actualCatchall, ok := actual.(*ast.CatchallNode); ok {
			// Compare NodeType
			if !assert.Equal(t, expectedCatchall.NodeType, actualCatchall.NodeType) {
				t.Logf("NodeType mismatch: expected '%s', got '%s'", expectedCatchall.NodeType, actualCatchall.NodeType)
				return false
			}

			// Normalize and compare RawContent
			if expectedCatchall.RawContent != nil && actualCatchall.RawContent != nil {
				expectedNorm := normalizeJSON(t, expectedCatchall.RawContent)
				actualNorm := normalizeJSON(t, actualCatchall.RawContent)
				if !assert.Equal(t, string(expectedNorm), string(actualNorm)) {
					t.Logf("RawContent mismatch:\nExpected: %s\nActual: %s", expectedNorm, actualNorm)
					return false
				}
			} else if !assert.Equal(t, expectedCatchall.RawContent == nil, actualCatchall.RawContent == nil) {
				t.Log("One node has RawContent while the other doesn't")
				return false
			}

			// Compare Operands
			if !assert.Equal(t, len(expectedCatchall.Operands), len(actualCatchall.Operands)) {
				t.Logf("Operands length mismatch: expected %d, got %d", len(expectedCatchall.Operands), len(actualCatchall.Operands))
				return false
			}
			for i := range expectedCatchall.Operands {
				if !compareNodes(t, expectedCatchall.Operands[i], actualCatchall.Operands[i]) {
					t.Logf("Operand %d mismatch", i)
					return false
				}
			}

			// Compare Wrap
			if expectedCatchall.Wrap != nil || actualCatchall.Wrap != nil {
				if !assert.Equal(t, expectedCatchall.Wrap != nil, actualCatchall.Wrap != nil) {
					t.Log("One node has Wrap while the other doesn't")
					return false
				}
				if expectedCatchall.Wrap != nil {
					if !compareNodes(t, expectedCatchall.Wrap, actualCatchall.Wrap) {
						t.Log("Wrap node mismatch")
						return false
					}
				}
			}

			return true
		}
	}

	// For Token nodes, compare their Wrap fields using compareNodes
	if expectedToken, ok := expected.(*ast.Token); ok {
		if actualToken, ok := actual.(*ast.Token); ok {
			if expectedToken.Wrap == nil || actualToken.Wrap == nil {
				return assert.Equal(t, expectedToken.Wrap == nil, actualToken.Wrap == nil)
			}
			return compareNodes(t, expectedToken.Wrap, actualToken.Wrap)
		}
	}

	// For TermGroup nodes, compare relation and operands
	if expectedGroup, ok := expected.(*ast.TermGroup); ok {
		if actualGroup, ok := actual.(*ast.TermGroup); ok {
			if !assert.Equal(t, expectedGroup.Relation, actualGroup.Relation) {
				t.Logf("Relation mismatch: expected '%s', got '%s'", expectedGroup.Relation, actualGroup.Relation)
				return false
			}
			if !assert.Equal(t, len(expectedGroup.Operands), len(actualGroup.Operands)) {
				t.Logf("Operands length mismatch: expected %d, got %d", len(expectedGroup.Operands), len(actualGroup.Operands))
				return false
			}
			for i := range expectedGroup.Operands {
				if !compareNodes(t, expectedGroup.Operands[i], actualGroup.Operands[i]) {
					t.Logf("Operand %d mismatch", i)
					return false
				}
			}
			return true
		}
	}

	// For Term nodes, compare all fields
	if expectedTerm, ok := expected.(*ast.Term); ok {
		if actualTerm, ok := actual.(*ast.Term); ok {
			equal := assert.Equal(t, expectedTerm.Foundry, actualTerm.Foundry) &&
				assert.Equal(t, expectedTerm.Key, actualTerm.Key) &&
				assert.Equal(t, expectedTerm.Layer, actualTerm.Layer) &&
				assert.Equal(t, expectedTerm.Match, actualTerm.Match) &&
				assert.Equal(t, expectedTerm.Value, actualTerm.Value)
			if !equal {
				t.Logf("Term mismatch:\nExpected: %+v\nActual: %+v", expectedTerm, actualTerm)
			}
			return equal
		}
	}

	// For other node types or mismatched types, use regular equality comparison
	equal := assert.Equal(t, expected, actual)
	if !equal {
		t.Logf("Node type mismatch:\nExpected type: %T\nActual type: %T", expected, actual)
	}
	return equal
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ast.Node
		wantErr  bool
	}{
		{
			name: "Parse simple term",
			input: `{
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "DET",
				"layer": "p",
				"match": "match:eq"
			}`,
			expected: &ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			wantErr: false,
		},
		{
			name: "Parse term group with AND relation",
			input: `{
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "AdjType",
						"layer": "m",
						"match": "match:eq",
						"value": "Pdt"
					}
				],
				"relation": "relation:and"
			}`,
			expected: &ast.TermGroup{
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
			wantErr: false,
		},
		{
			name: "Parse token with wrapped term",
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
			expected: &ast.Token{
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name: "Parse complex nested structure",
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:termGroup",
					"operands": [
						{
							"@type": "koral:term",
							"foundry": "opennlp",
							"key": "DET",
							"layer": "p",
							"match": "match:eq"
						},
						{
							"@type": "koral:termGroup",
							"operands": [
								{
									"@type": "koral:term",
									"foundry": "opennlp",
									"key": "AdjType",
									"layer": "m",
									"match": "match:eq",
									"value": "Pdt"
								},
								{
									"@type": "koral:term",
									"foundry": "opennlp",
									"key": "PronType",
									"layer": "m",
									"match": "match:ne",
									"value": "Neg"
								}
							],
							"relation": "relation:or"
						}
					],
					"relation": "relation:and"
				}
			}`,
			expected: &ast.Token{
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
									Match:   ast.MatchNotEqual,
									Value:   "Neg",
								},
							},
							Relation: ast.OrRelation,
						},
					},
					Relation: ast.AndRelation,
				},
			},
			wantErr: false,
		},
		{
			name: "Parse token with term and rewrites",
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "Baum",
					"layer": "orth",
					"match": "match:eq",
					"rewrites": [
						{
							"@type": "koral:rewrite",
							"_comment": "Default foundry has been added.",
							"editor": "Kustvakt",
							"operation": "operation:injection",
							"scope": "foundry",
							"src": "Kustvakt"
						}
					]
				}
			}`,
			expected: &ast.Token{
				Wrap: &ast.Term{
					Foundry: "opennlp",
					Key:     "Baum",
					Layer:   "orth",
					Match:   ast.MatchEqual,
					Rewrites: []ast.Rewrite{
						{
							Comment:   "Default foundry has been added.",
							Editor:    "Kustvakt",
							Operation: "operation:injection",
							Scope:     "foundry",
							Src:       "Kustvakt",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parse term group with rewrites",
			input: `{
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					},
					{
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "AdjType",
						"layer": "m",
						"match": "match:eq",
						"value": "Pdt"
					}
				],
				"relation": "relation:and",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Default foundry has been added.",
						"editor": "Kustvakt",
						"operation": "operation:injection",
						"scope": "foundry",
						"src": "Kustvakt"
					}
				]
			}`,
			expected: &ast.TermGroup{
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
				Rewrites: []ast.Rewrite{
					{
						Comment:   "Default foundry has been added.",
						Editor:    "Kustvakt",
						Operation: "operation:injection",
						Scope:     "foundry",
						Src:       "Kustvakt",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Parse term with rewrites",
			input: `{
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "DET",
				"layer": "p",
				"match": "match:eq",
				"rewrites": [
					{
						"@type": "koral:rewrite",
						"_comment": "Default foundry has been added.",
						"editor": "Kustvakt",
						"operation": "operation:injection",
						"scope": "foundry",
						"src": "Kustvakt"
					}
				]
			}`,
			expected: &ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
				Rewrites: []ast.Rewrite{
					{
						Comment:   "Default foundry has been added.",
						Editor:    "Kustvakt",
						Operation: "operation:injection",
						Scope:     "foundry",
						Src:       "Kustvakt",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{"invalid": json`,
			wantErr: true,
		},
		{
			name:    "Empty JSON",
			input:   `{}`,
			wantErr: true,
		},
		{
			name: "Unknown node type",
			input: `{
				"@type": "koral:unknown",
				"key": "value"
			}`,
			expected: &ast.CatchallNode{
				NodeType:   "koral:unknown",
				RawContent: json.RawMessage(`{"@type":"koral:unknown","key":"value"}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSON([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSONErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Empty JSON",
			input:   "{}",
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			input:   "{",
			wantErr: true,
		},
		{
			name: "Token without wrap",
			input: `{
				"@type": "koral:token"
			}`,
			wantErr: true,
		},
		{
			name: "Term without key",
			input: `{
				"@type": "koral:term",
				"foundry": "opennlp",
				"layer": "p",
				"match": "match:eq"
			}`,
			wantErr: true,
		},
		{
			name: "TermGroup without operands",
			input: `{
				"@type": "koral:termGroup",
				"relation": "relation:and"
			}`,
			wantErr: true,
		},
		{
			name: "TermGroup without relation",
			input: `{
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"key": "DET",
						"foundry": "opennlp",
						"layer": "p",
						"match": "match:eq"
					}
				]
			}`,
			wantErr: true,
		},
		{
			name: "Invalid match type",
			input: `{
				"@type": "koral:term",
				"key": "DET",
				"foundry": "opennlp",
				"layer": "p",
				"match": "match:invalid"
			}`,
			wantErr: true,
		},
		{
			name: "Invalid relation type",
			input: `{
				"@type": "koral:termGroup",
				"operands": [
					{
						"@type": "koral:term",
						"key": "DET",
						"foundry": "opennlp",
						"layer": "p",
						"match": "match:eq"
					}
				],
				"relation": "relation:invalid"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSerializeToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    ast.Node
		expected string
		wantErr  bool
	}{
		{
			name: "Serialize simple term",
			input: &ast.Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   ast.MatchEqual,
			},
			expected: `{
  "@type": "koral:term",
  "foundry": "opennlp",
  "key": "DET",
  "layer": "p",
  "match": "match:eq"
}`,
			wantErr: false,
		},
		{
			name: "Serialize term group",
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
			expected: `{
  "@type": "koral:termGroup",
  "operands": [
    {
      "@type": "koral:term",
      "foundry": "opennlp",
      "key": "DET",
      "layer": "p",
      "match": "match:eq"
    },
    {
      "@type": "koral:term",
      "foundry": "opennlp",
      "key": "AdjType",
      "layer": "m",
      "match": "match:eq",
      "value": "Pdt"
    }
  ],
  "relation": "relation:and"
}`,
			wantErr: false,
		},
		{
			name: "Serialize unknown node type",
			input: &ast.CatchallNode{
				NodeType: "koral:unknown",
				RawContent: json.RawMessage(`{
  "@type": "koral:unknown",
  "key": "value"
}`),
			},
			expected: `{
  "@type": "koral:unknown",
  "key": "value"
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SerializeToJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			// Compare JSON objects instead of raw strings to avoid whitespace issues
			var expected, actual any
			err = json.Unmarshal([]byte(tt.expected), &expected)
			require.NoError(t, err)
			err = json.Unmarshal(result, &actual)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that parsing and then serializing produces equivalent JSON
	input := `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:termGroup",
			"operands": [
				{
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "DET",
					"layer": "p",
					"match": "match:eq"
				},
				{
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "AdjType",
					"layer": "m",
					"match": "match:eq",
					"value": "Pdt"
				}
			],
			"relation": "relation:and"
		}
	}`

	// Parse JSON to AST
	node, err := ParseJSON([]byte(input))
	require.NoError(t, err)

	// Serialize AST back to JSON
	output, err := SerializeToJSON(node)
	require.NoError(t, err)

	// Compare JSON objects
	var expected, actual any
	err = json.Unmarshal([]byte(input), &expected)
	require.NoError(t, err)
	err = json.Unmarshal(output, &actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRoundTripUnknownType(t *testing.T) {
	// Test that parsing and then serializing an unknown node type preserves the structure
	input := `{
		"@type": "koral:unknown",
		"key": "value",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "DET",
			"layer": "p",
			"match": "match:eq"
		},
		"operands": [
			{
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "AdjType",
				"layer": "m",
				"match": "match:eq",
				"value": "Pdt"
			}
		]
	}`

	// Parse JSON to AST
	node, err := ParseJSON([]byte(input))
	require.NoError(t, err)

	// Check that it's a CatchallNode
	catchall, ok := node.(*ast.CatchallNode)
	require.True(t, ok)
	assert.Equal(t, "koral:unknown", catchall.NodeType)

	// Check that wrap and operands were parsed
	require.NotNil(t, catchall.Wrap)
	require.Len(t, catchall.Operands, 1)

	// Serialize AST back to JSON
	output, err := SerializeToJSON(node)
	require.NoError(t, err)

	// Compare JSON objects
	var expected, actual any
	err = json.Unmarshal([]byte(input), &expected)
	require.NoError(t, err)
	err = json.Unmarshal(output, &actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ast.Node
		wantErr  bool
	}{
		{
			name: "Unknown node type",
			input: `{
				"@type": "koral:unknown",
				"customField": "value",
				"wrap": {
					"@type": "koral:term",
					"key": "DET"
				}
			}`,
			expected: &ast.CatchallNode{
				NodeType: "koral:unknown",
				RawContent: json.RawMessage(`{
					"@type": "koral:unknown",
					"customField": "value",
					"wrap": {
						"@type": "koral:term",
						"key": "DET"
					}
				}`),
				Wrap: &ast.Term{
					Key:   "DET",
					Match: ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name: "Unknown node with operands",
			input: `{
				"@type": "koral:unknown",
				"operands": [
					{
						"@type": "koral:term",
						"key": "DET"
					},
					{
						"@type": "koral:term",
						"key": "NOUN"
					}
				]
			}`,
			expected: &ast.CatchallNode{
				NodeType: "koral:unknown",
				RawContent: json.RawMessage(`{
					"@type": "koral:unknown",
					"operands": [
						{
							"@type": "koral:term",
							"key": "DET"
						},
						{
							"@type": "koral:term",
							"key": "NOUN"
						}
					]
				}`),
				Operands: []ast.Node{
					&ast.Term{
						Key:   "DET",
						Match: ast.MatchEqual,
					},
					&ast.Term{
						Key:   "NOUN",
						Match: ast.MatchEqual,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Deeply nested unknown nodes",
			input: `{
				"@type": "koral:outer",
				"wrap": {
					"@type": "koral:middle",
					"wrap": {
						"@type": "koral:inner",
						"wrap": {
							"@type": "koral:term",
							"key": "DET"
						}
					}
				}
			}`,
			expected: &ast.CatchallNode{
				NodeType: "koral:outer",
				RawContent: json.RawMessage(`{
					"@type": "koral:outer",
					"wrap": {
						"@type": "koral:middle",
						"wrap": {
							"@type": "koral:inner",
							"wrap": {
								"@type": "koral:term",
								"key": "DET"
							}
						}
					}
				}`),
				Wrap: &ast.CatchallNode{
					NodeType: "koral:middle",
					RawContent: json.RawMessage(`{
						"@type": "koral:middle",
						"wrap": {
							"@type": "koral:inner",
							"wrap": {
								"@type": "koral:term",
								"key": "DET"
							}
						}
					}`),
					Wrap: &ast.CatchallNode{
						NodeType: "koral:inner",
						RawContent: json.RawMessage(`{
							"@type": "koral:inner",
							"wrap": {
								"@type": "koral:term",
								"key": "DET"
							}
						}`),
						Wrap: &ast.Term{
							Key:   "DET",
							Match: ast.MatchEqual,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Mixed known and unknown nodes",
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:custom",
					"customField": "value",
					"operands": [
						{
							"@type": "koral:termGroup",
							"operands": [
								{
									"@type": "koral:term",
									"key": "DET"
								}
							],
							"relation": "relation:and"
						}
					]
				}
			}`,
			expected: &ast.Token{
				Wrap: &ast.CatchallNode{
					NodeType: "koral:custom",
					RawContent: json.RawMessage(`{
						"@type": "koral:custom",
						"customField": "value",
						"operands": [
							{
								"@type": "koral:termGroup",
								"operands": [
									{
										"@type": "koral:term",
										"key": "DET"
									}
								],
								"relation": "relation:and"
							}
						]
					}`),
					Operands: []ast.Node{
						&ast.TermGroup{
							Operands: []ast.Node{
								&ast.Term{
									Key:   "DET",
									Match: ast.MatchEqual,
								},
							},
							Relation: ast.AndRelation,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty operands in term group",
			input: `{
				"@type": "koral:termGroup",
				"operands": [],
				"relation": "relation:and"
			}`,
			wantErr: true,
		},
		{
			name: "Null values in term",
			input: `{
				"@type": "koral:term",
				"foundry": null,
				"key": "DET",
				"layer": null,
				"match": null,
				"value": null
			}`,
			expected: &ast.Term{
				Key:   "DET",
				Match: ast.MatchEqual,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSON([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			compareNodes(t, tt.expected, result)
		})
	}
}
