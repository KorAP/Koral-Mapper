package parser

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			var expected, actual interface{}
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
	var expected, actual interface{}
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
	var expected, actual interface{}
	err = json.Unmarshal([]byte(input), &expected)
	require.NoError(t, err)
	err = json.Unmarshal(output, &actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
