package mapper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapper(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `- id: test-mapper
  foundryA: opennlp
  layerA: p
  foundryB: upos
  layerB: p
  mappings:
    - "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]"
    - "[DET] <> [opennlp/p=DET]"`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create a new mapper
	m, err := NewMapper(configFile)
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
			name:      "B to A mapping",
			mappingID: "test-mapper",
			opts: MappingOptions{
				Direction: BtoA,
			},
			input: `{
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
				Direction: "invalid",
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
