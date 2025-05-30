package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Rewrite
	}{
		{
			name: "Standard rewrite with editor and original",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"original": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Original: map[string]any{
					"@type":   "koral:term",
					"foundry": "opennlp",
					"key":     "PIDAT",
					"layer":   "p",
					"match":   "match:eq",
				},
			},
		},
		{
			name: "Legacy rewrite with source instead of editor",
			input: `{
				"@type": "koral:rewrite",
				"source": "legacy-mapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"src": "legacy-source"
			}`,
			expected: Rewrite{
				Editor:    "legacy-mapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "legacy-source",
			},
		},
		{
			name: "Legacy rewrite with origin instead of original/src",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"origin": "legacy-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "legacy-origin",
			},
		},
		{
			name: "Precedence test: editor over source",
			input: `{
				"@type": "koral:rewrite",
				"editor": "preferred-editor",
				"source": "legacy-source",
				"operation": "operation:mapping"
			}`,
			expected: Rewrite{
				Editor:    "preferred-editor",
				Operation: "operation:mapping",
			},
		},
		{
			name: "Precedence test: original over src over origin",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"original": "preferred-original",
				"src": "middle-src",
				"origin": "lowest-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Original:  "preferred-original",
			},
		},
		{
			name: "Precedence test: src over origin when no original",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"src": "preferred-src",
				"origin": "lowest-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Src:       "preferred-src",
			},
		},
		{
			name: "Only legacy fields",
			input: `{
				"@type": "koral:rewrite",
				"source": "legacy-editor",
				"operation": "operation:mapping",
				"origin": "legacy-origin",
				"_comment": "Legacy rewrite"
			}`,
			expected: Rewrite{
				Editor:    "legacy-editor",
				Operation: "operation:mapping",
				Src:       "legacy-origin",
				Comment:   "Legacy rewrite",
			},
		},
		{
			name: "Mixed with comment",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"src": "original-source",
				"_comment": "This is a comment"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "original-source",
				Comment:   "This is a comment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rewrite Rewrite
			err := json.Unmarshal([]byte(tt.input), &rewrite)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, rewrite)
		})
	}
}

func TestRewriteArrayUnmarshal(t *testing.T) {
	// Test unmarshaling an array of rewrites with mixed legacy and modern fields
	input := `[
		{
			"@type": "koral:rewrite",
			"editor": "termMapper",
			"operation": "operation:mapping",
			"original": "modern-original"
		},
		{
			"@type": "koral:rewrite",
			"source": "legacy-editor",
			"operation": "operation:legacy",
			"origin": "legacy-origin"
		}
	]`

	var rewrites []Rewrite
	err := json.Unmarshal([]byte(input), &rewrites)
	require.NoError(t, err)
	require.Len(t, rewrites, 2)

	// Check first rewrite (modern)
	assert.Equal(t, "termMapper", rewrites[0].Editor)
	assert.Equal(t, "operation:mapping", rewrites[0].Operation)
	assert.Equal(t, "modern-original", rewrites[0].Original)

	// Check second rewrite (legacy)
	assert.Equal(t, "legacy-editor", rewrites[1].Editor)
	assert.Equal(t, "operation:legacy", rewrites[1].Operation)
	assert.Equal(t, "legacy-origin", rewrites[1].Src)
}

func TestRewriteMarshalJSON(t *testing.T) {
	// Test that marshaling works correctly and maintains the modern field names
	rewrite := Rewrite{
		Editor:    "termMapper",
		Operation: "operation:mapping",
		Scope:     "foundry",
		Src:       "source-value",
		Comment:   "Test comment",
		Original:  "original-value",
	}

	data, err := json.Marshal(rewrite)
	require.NoError(t, err)

	// Parse back to verify structure
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "termMapper", result["editor"])
	assert.Equal(t, "operation:mapping", result["operation"])
	assert.Equal(t, "foundry", result["scope"])
	assert.Equal(t, "source-value", result["src"])
	assert.Equal(t, "Test comment", result["_comment"])
	assert.Equal(t, "original-value", result["original"])

	// Ensure legacy fields are not present in output
	assert.NotContains(t, result, "source")
	assert.NotContains(t, result, "origin")
}
