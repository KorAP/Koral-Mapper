package config

import (
	"os"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary YAML file
	content := `
- id: opennlp-mapper
  foundryA: opennlp
  layerA: p
  foundryB: upos
  layerB: p
  mappings:
    - "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]"
    - "[PAV] <> [ADV & PronType:Dem]"

- id: simple-mapper
  mappings:
    - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	// Test loading the configuration
	config, err := LoadConfig(tmpfile.Name())
	require.NoError(t, err)

	// Verify the configuration
	require.Len(t, config.Lists, 2)

	// Check first mapping list
	list1 := config.Lists[0]
	assert.Equal(t, "opennlp-mapper", list1.ID)
	assert.Equal(t, "opennlp", list1.FoundryA)
	assert.Equal(t, "p", list1.LayerA)
	assert.Equal(t, "upos", list1.FoundryB)
	assert.Equal(t, "p", list1.LayerB)
	require.Len(t, list1.Mappings, 2)
	assert.Equal(t, "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]", string(list1.Mappings[0]))
	assert.Equal(t, "[PAV] <> [ADV & PronType:Dem]", string(list1.Mappings[1]))

	// Check second mapping list
	list2 := config.Lists[1]
	assert.Equal(t, "simple-mapper", list2.ID)
	assert.Empty(t, list2.FoundryA)
	assert.Empty(t, list2.LayerA)
	assert.Empty(t, list2.FoundryB)
	assert.Empty(t, list2.LayerB)
	require.Len(t, list2.Mappings, 1)
	assert.Equal(t, "[A] <> [B]", string(list2.Mappings[0]))
}

func TestParseMappings(t *testing.T) {
	list := &MappingList{
		ID:       "test-mapper",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "p",
		Mappings: []MappingRule{
			"[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
		},
	}

	results, err := list.ParseMappings()
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Check the parsed upper pattern
	upper := results[0].Upper
	require.NotNil(t, upper)
	require.IsType(t, &ast.Token{}, upper)
	upperTerm := upper.Wrap.(*ast.Term)
	assert.Equal(t, "opennlp", upperTerm.Foundry)
	assert.Equal(t, "p", upperTerm.Layer)
	assert.Equal(t, "PIDAT", upperTerm.Key)

	// Check the parsed lower pattern
	lower := results[0].Lower
	require.NotNil(t, lower)
	require.IsType(t, &ast.Token{}, lower)
	lowerGroup := lower.Wrap.(*ast.TermGroup)
	require.Len(t, lowerGroup.Operands, 2)
	assert.Equal(t, ast.AndRelation, lowerGroup.Relation)

	// Check first operand
	term1 := lowerGroup.Operands[0].(*ast.Term)
	assert.Equal(t, "opennlp", term1.Foundry)
	assert.Equal(t, "p", term1.Layer)
	assert.Equal(t, "PIDAT", term1.Key)

	// Check second operand
	term2 := lowerGroup.Operands[1].(*ast.Term)
	assert.Equal(t, "opennlp", term2.Foundry)
	assert.Equal(t, "p", term2.Layer)
	assert.Equal(t, "AdjType", term2.Key)
	assert.Equal(t, "Pdt", term2.Value)
}

func TestLoadConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "Missing ID",
			content: `
- foundryA: opennlp
  mappings:
    - "[A] <> [B]"
`,
			wantErr: "mapping list at index 0 is missing an ID",
		},
		{
			name: "Empty mappings",
			content: `
- id: test
  foundryA: opennlp
  mappings: []
`,
			wantErr: "mapping list 'test' has no mapping rules",
		},
		{
			name: "Empty rule",
			content: `
- id: test
  mappings:
    - ""
`,
			wantErr: "mapping list 'test' rule at index 0 is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.WriteString(tt.content)
			require.NoError(t, err)
			err = tmpfile.Close()
			require.NoError(t, err)

			_, err = LoadConfig(tmpfile.Name())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
