package config

import (
	"os"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
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

func TestLoadConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "Duplicate mapping list IDs",
			content: `
- id: test
  mappings:
    - "[A] <> [B]"
- id: test
  mappings:
    - "[C] <> [D]"`,
			wantErr: "duplicate mapping list ID found: test",
		},
		{
			name: "Invalid YAML syntax",
			content: `
- id: test
  mappings:
    - [A] <> [B]  # Unquoted special characters
`,
			wantErr: "yaml",
		},
		{
			name:    "Empty file",
			content: "",
			wantErr: "EOF",
		},
		{
			name: "Non-list YAML",
			content: `
id: test
mappings:
  - "[A] <> [B]"`,
			wantErr: "cannot unmarshal",
		},
		{
			name: "Missing required fields",
			content: `
- mappings:
    - "[A] <> [B]"
- id: test2
  foundryA: opennlp`,
			wantErr: "missing an ID",
		},
		{
			name: "Empty mappings list",
			content: `
- id: test
  foundryA: opennlp
  mappings: []`,
			wantErr: "has no mapping rules",
		},
		{
			name: "Null values in optional fields",
			content: `
- id: test
  foundryA: null
  layerA: null
  foundryB: null
  layerB: null
  mappings:
    - "[A] <> [B]"`,
			wantErr: "",
		},
		{
			name: "Special characters in IDs",
			content: `
- id: "test/special@chars#1"
  mappings:
    - "[A] <> [B]"`,
			wantErr: "",
		},
		{
			name: "Unicode characters in mappings",
			content: `
- id: test
  mappings:
    - "[ß] <> [ss]"
    - "[é] <> [e]"`,
			wantErr: "",
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

			config, err := LoadConfig(tmpfile.Name())
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	}
}

func TestParseMappingsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		list     *MappingList
		wantErr  bool
		errCheck func(t *testing.T, err error)
	}{
		{
			name: "Empty mapping rule",
			list: &MappingList{
				ID:       "test",
				Mappings: []MappingRule{""},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "empty")
			},
		},
		{
			name: "Invalid mapping syntax",
			list: &MappingList{
				ID:       "test",
				Mappings: []MappingRule{"[A] -> [B]"},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to parse")
			},
		},
		{
			name: "Missing brackets",
			list: &MappingList{
				ID:       "test",
				Mappings: []MappingRule{"A <> B"},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to parse")
			},
		},
		{
			name: "Complex nested expressions",
			list: &MappingList{
				ID: "test",
				Mappings: []MappingRule{
					"[A & (B | C) & (D | (E & F))] <> [X & (Y | Z)]",
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple foundry/layer combinations",
			list: &MappingList{
				ID: "test",
				Mappings: []MappingRule{
					"[foundry1/layer1=A & foundry2/layer2=B] <> [foundry3/layer3=C]",
				},
			},
			wantErr: false,
		},
		{
			name: "Default foundry/layer override",
			list: &MappingList{
				ID:       "test",
				FoundryA: "defaultFoundry",
				LayerA:   "defaultLayer",
				Mappings: []MappingRule{
					"[A] <> [B]", // Should use defaults
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := tt.list.ParseMappings()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCheck != nil {
					tt.errCheck(t, err)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, results)
		})
	}
}

func TestUserProvidedMappingRules(t *testing.T) {
	// Test the exact YAML mapping rules provided by the user
	content := `
- id: stts-ud
  foundryA: opennlp
  layerA: p
  foundryB: upos
  layerB: p
  mappings:
    - "[$\\(] <> [PUNCT & PunctType=Brck]"
    - "[$,] <> [PUNCT & PunctType=Comm]"
    - "[$.] <> [PUNCT & PunctType=Peri]"
    - "[ADJA] <> [ADJ]"
    - "[ADJD] <> [ADJ & Variant=Short]"
    - "[ADV] <> [ADV]"
`
	tmpfile, err := os.CreateTemp("", "user-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	// Test loading the configuration
	config, err := LoadConfig(tmpfile.Name())
	require.NoError(t, err)

	// Verify the configuration loaded correctly
	require.Len(t, config.Lists, 1)
	list := config.Lists[0]
	assert.Equal(t, "stts-ud", list.ID)
	assert.Equal(t, "opennlp", list.FoundryA)
	assert.Equal(t, "p", list.LayerA)
	assert.Equal(t, "upos", list.FoundryB)
	assert.Equal(t, "p", list.LayerB)
	require.Len(t, list.Mappings, 6)

	// First, test individual mappings to isolate the issue
	t.Run("parenthesis mapping", func(t *testing.T) {
		singleRule := &MappingList{
			ID:       "test-paren",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []MappingRule{"[$\\(] <> [PUNCT & PunctType=Brck]"},
		}
		results, err := singleRule.ParseMappings()
		require.NoError(t, err)
		require.Len(t, results, 1)

		upperTerm := results[0].Upper.Wrap.(*ast.Term)
		assert.Equal(t, "$(", upperTerm.Key)
	})

	t.Run("comma mapping", func(t *testing.T) {
		singleRule := &MappingList{
			ID:       "test-comma",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []MappingRule{"[$,] <> [PUNCT & PunctType=Comm]"},
		}
		results, err := singleRule.ParseMappings()
		require.NoError(t, err)
		require.Len(t, results, 1)

		upperTerm := results[0].Upper.Wrap.(*ast.Term)
		assert.Equal(t, "$,", upperTerm.Key)
	})

	t.Run("period mapping", func(t *testing.T) {
		singleRule := &MappingList{
			ID:       "test-period",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "p",
			Mappings: []MappingRule{"[$.] <> [PUNCT & PunctType=Peri]"},
		}
		results, err := singleRule.ParseMappings()
		require.NoError(t, err)
		require.Len(t, results, 1)

		upperTerm := results[0].Upper.Wrap.(*ast.Term)
		assert.Equal(t, "$.", upperTerm.Key)
	})

	// Test that all mapping rules can be parsed successfully
	results, err := list.ParseMappings()
	require.NoError(t, err)
	require.Len(t, results, 6)

	// Verify specific parsing of the special character mapping
	// The first mapping "[$\\(] <> [PUNCT & PunctType=Brck]" should parse correctly
	firstMapping := results[0]
	require.NotNil(t, firstMapping.Upper)
	upperTerm := firstMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "$(", upperTerm.Key) // The actual parsed key should be "$("
	assert.Equal(t, "opennlp", upperTerm.Foundry)
	assert.Equal(t, "p", upperTerm.Layer)

	require.NotNil(t, firstMapping.Lower)
	lowerGroup := firstMapping.Lower.Wrap.(*ast.TermGroup)
	require.Len(t, lowerGroup.Operands, 2)
	assert.Equal(t, ast.AndRelation, lowerGroup.Relation)

	// Check the PUNCT term
	punctTerm := lowerGroup.Operands[0].(*ast.Term)
	assert.Equal(t, "PUNCT", punctTerm.Key)
	assert.Equal(t, "upos", punctTerm.Foundry)
	assert.Equal(t, "p", punctTerm.Layer)

	// Check the PunctType term
	punctTypeTerm := lowerGroup.Operands[1].(*ast.Term)
	assert.Equal(t, "PunctType", punctTypeTerm.Layer)
	assert.Equal(t, "Brck", punctTypeTerm.Key)
	assert.Equal(t, "upos", punctTypeTerm.Foundry)

	// Verify the comma mapping as well
	secondMapping := results[1]
	upperTerm2 := secondMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "$,", upperTerm2.Key)

	// Verify the period mapping
	thirdMapping := results[2]
	upperTerm3 := thirdMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "$.", upperTerm3.Key)

	// Verify basic mappings without special characters
	fourthMapping := results[3]
	upperTerm4 := fourthMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "ADJA", upperTerm4.Key)
	lowerTerm4 := fourthMapping.Lower.Wrap.(*ast.Term)
	assert.Equal(t, "ADJ", lowerTerm4.Key)
}

func TestExistingUposYaml(t *testing.T) {
	// Test that the existing upos.yaml file can be parsed correctly
	config, err := LoadConfig("../upos.yaml")
	require.NoError(t, err)

	// Verify the configuration loaded correctly
	require.Len(t, config.Lists, 1)
	list := config.Lists[0]
	assert.Equal(t, "stts-ud", list.ID)
	assert.Equal(t, "opennlp", list.FoundryA)
	assert.Equal(t, "p", list.LayerA)
	assert.Equal(t, "upos", list.FoundryB)
	assert.Equal(t, "p", list.LayerB)
	require.Len(t, list.Mappings, 54) // Should have 54 mapping rules

	// Test that all mapping rules can be parsed successfully
	results, err := list.ParseMappings()
	require.NoError(t, err)
	require.Len(t, results, 54)

	// Test a few specific mappings to ensure they parse correctly

	// Test the special character mappings
	firstMapping := results[0] // "[$\\(] <> [PUNCT & PunctType=Brck]"
	upperTerm := firstMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "$(", upperTerm.Key)
	assert.Equal(t, "opennlp", upperTerm.Foundry)
	assert.Equal(t, "p", upperTerm.Layer)

	lowerGroup := firstMapping.Lower.Wrap.(*ast.TermGroup)
	require.Len(t, lowerGroup.Operands, 2)
	assert.Equal(t, ast.AndRelation, lowerGroup.Relation)

	punctTerm := lowerGroup.Operands[0].(*ast.Term)
	assert.Equal(t, "PUNCT", punctTerm.Key)
	assert.Equal(t, "upos", punctTerm.Foundry)
	assert.Equal(t, "p", punctTerm.Layer)

	punctTypeTerm := lowerGroup.Operands[1].(*ast.Term)
	assert.Equal(t, "PunctType", punctTypeTerm.Layer)
	assert.Equal(t, "Brck", punctTypeTerm.Key)
	assert.Equal(t, "upos", punctTypeTerm.Foundry)

	// Test a complex mapping with multiple attributes
	// "[PIDAT] <> [DET & AdjType=Pdt & (PronType=Ind | PronType=Neg | PronType=Tot)]"
	pidatMapping := results[24] // This should be the PIDAT mapping
	pidatUpper := pidatMapping.Upper.Wrap.(*ast.Term)
	assert.Equal(t, "PIDAT", pidatUpper.Key)

	pidatLower := pidatMapping.Lower.Wrap.(*ast.TermGroup)
	assert.Equal(t, ast.AndRelation, pidatLower.Relation)
	require.Len(t, pidatLower.Operands, 3) // DET, AdjType=Pdt, and the parenthesized group

	detTerm := pidatLower.Operands[0].(*ast.Term)
	assert.Equal(t, "DET", detTerm.Key)

	adjTypeTerm := pidatLower.Operands[1].(*ast.Term)
	assert.Equal(t, "AdjType", adjTypeTerm.Layer)
	assert.Equal(t, "Pdt", adjTypeTerm.Key)

	// The third operand should be a nested TermGroup with OR relation
	nestedGroup := pidatLower.Operands[2].(*ast.TermGroup)
	assert.Equal(t, ast.OrRelation, nestedGroup.Relation)
	require.Len(t, nestedGroup.Operands, 3) // PronType=Ind, PronType=Neg, PronType=Tot

	for i, expectedValue := range []string{"Ind", "Neg", "Tot"} {
		pronTypeTerm := nestedGroup.Operands[i].(*ast.Term)
		assert.Equal(t, "PronType", pronTypeTerm.Layer)
		assert.Equal(t, expectedValue, pronTypeTerm.Key)
	}
}

func TestConfigWithSdkAndServer(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedSDK    string
		expectedServer string
		wantErr        bool
	}{
		{
			name: "Configuration with SDK and Server values",
			content: `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
lists:
- id: test-mapper
  foundryA: opennlp
  layerA: p
  foundryB: upos
  layerB: p
  mappings:
    - "[A] <> [B]"
`,
			expectedSDK:    "https://custom.example.com/sdk.js",
			expectedServer: "https://custom.example.com/",
			wantErr:        false,
		},
		{
			name: "Configuration with only SDK value",
			content: `
sdk: "https://custom.example.com/sdk.js"
lists:
- id: test-mapper
  mappings:
    - "[A] <> [B]"
`,
			expectedSDK:    "https://custom.example.com/sdk.js",
			expectedServer: "https://korap.ids-mannheim.de/", // default applied
			wantErr:        false,
		},
		{
			name: "Configuration with only Server value",
			content: `
server: "https://custom.example.com/"
lists:
- id: test-mapper
  mappings:
    - "[A] <> [B]"
`,
			expectedSDK:    "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", // default applied
			expectedServer: "https://custom.example.com/",
			wantErr:        false,
		},
		{
			name: "Configuration without SDK and Server (old format with defaults applied)",
			content: `
- id: test-mapper
  mappings:
    - "[A] <> [B]"
`,
			expectedSDK:    "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", // default applied
			expectedServer: "https://korap.ids-mannheim.de/",                          // default applied
			wantErr:        false,
		},
		{
			name: "Configuration with lists field explicitly",
			content: `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
lists:
- id: test-mapper-1
  mappings:
    - "[A] <> [B]"
- id: test-mapper-2
  mappings:
    - "[C] <> [D]"
`,
			expectedSDK:    "https://custom.example.com/sdk.js",
			expectedServer: "https://custom.example.com/",
			wantErr:        false,
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

			config, err := LoadConfig(tmpfile.Name())
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			// Check SDK and Server values
			assert.Equal(t, tt.expectedSDK, config.SDK)
			assert.Equal(t, tt.expectedServer, config.Server)

			// Ensure lists are still loaded correctly
			require.Greater(t, len(config.Lists), 0)

			// Verify first mapping list
			firstList := config.Lists[0]
			assert.NotEmpty(t, firstList.ID)
			assert.Greater(t, len(firstList.Mappings), 0)
		})
	}
}

func TestLoadFromSources(t *testing.T) {
	// Create main config file
	mainConfigContent := `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
lists:
- id: main-mapper
  mappings:
    - "[A] <> [B]"
`
	mainConfigFile, err := os.CreateTemp("", "main-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mainConfigFile.Name())

	_, err = mainConfigFile.WriteString(mainConfigContent)
	require.NoError(t, err)
	err = mainConfigFile.Close()
	require.NoError(t, err)

	// Create individual mapping files
	mappingFile1Content := `
id: mapper-1
foundryA: opennlp
layerA: p
mappings:
  - "[C] <> [D]"
`
	mappingFile1, err := os.CreateTemp("", "mapping1-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile1.Name())

	_, err = mappingFile1.WriteString(mappingFile1Content)
	require.NoError(t, err)
	err = mappingFile1.Close()
	require.NoError(t, err)

	mappingFile2Content := `
id: mapper-2
foundryB: upos
layerB: p
mappings:
  - "[E] <> [F]"
`
	mappingFile2, err := os.CreateTemp("", "mapping2-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile2.Name())

	_, err = mappingFile2.WriteString(mappingFile2Content)
	require.NoError(t, err)
	err = mappingFile2.Close()
	require.NoError(t, err)

	tests := []struct {
		name         string
		configFile   string
		mappingFiles []string
		wantErr      bool
		expectedIDs  []string
	}{
		{
			name:         "Main config only",
			configFile:   mainConfigFile.Name(),
			mappingFiles: []string{},
			wantErr:      false,
			expectedIDs:  []string{"main-mapper"},
		},
		{
			name:         "Mapping files only",
			configFile:   "",
			mappingFiles: []string{mappingFile1.Name(), mappingFile2.Name()},
			wantErr:      false,
			expectedIDs:  []string{"mapper-1", "mapper-2"},
		},
		{
			name:         "Main config and mapping files",
			configFile:   mainConfigFile.Name(),
			mappingFiles: []string{mappingFile1.Name(), mappingFile2.Name()},
			wantErr:      false,
			expectedIDs:  []string{"main-mapper", "mapper-1", "mapper-2"},
		},
		{
			name:         "No configuration sources",
			configFile:   "",
			mappingFiles: []string{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromSources(tt.configFile, tt.mappingFiles)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			// Check that all expected mapping IDs are present
			require.Len(t, config.Lists, len(tt.expectedIDs))
			actualIDs := make([]string, len(config.Lists))
			for i, list := range config.Lists {
				actualIDs[i] = list.ID
			}
			for _, expectedID := range tt.expectedIDs {
				assert.Contains(t, actualIDs, expectedID)
			}

			// Check that SDK and Server are set (either from config or defaults)
			assert.NotEmpty(t, config.SDK)
			assert.NotEmpty(t, config.Server)
		})
	}
}

func TestLoadFromSourcesWithDefaults(t *testing.T) {
	// Test that defaults are applied when loading only mapping files
	mappingFileContent := `
id: test-mapper
mappings:
  - "[A] <> [B]"
`
	mappingFile, err := os.CreateTemp("", "mapping-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile.Name())

	_, err = mappingFile.WriteString(mappingFileContent)
	require.NoError(t, err)
	err = mappingFile.Close()
	require.NoError(t, err)

	config, err := LoadFromSources("", []string{mappingFile.Name()})
	require.NoError(t, err)

	// Check that defaults are applied
	assert.Equal(t, defaultSDK, config.SDK)
	assert.Equal(t, defaultServer, config.Server)
	require.Len(t, config.Lists, 1)
	assert.Equal(t, "test-mapper", config.Lists[0].ID)
}

func TestLoadFromSourcesDuplicateIDs(t *testing.T) {
	// Create config with duplicate IDs across sources
	configContent := `
lists:
- id: duplicate-id
  mappings:
    - "[A] <> [B]"
`
	configFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(configContent)
	require.NoError(t, err)
	err = configFile.Close()
	require.NoError(t, err)

	mappingContent := `
id: duplicate-id
mappings:
  - "[C] <> [D]"
`
	mappingFile, err := os.CreateTemp("", "mapping-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile.Name())

	_, err = mappingFile.WriteString(mappingContent)
	require.NoError(t, err)
	err = mappingFile.Close()
	require.NoError(t, err)

	_, err = LoadFromSources(configFile.Name(), []string{mappingFile.Name()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate mapping list ID found: duplicate-id")
}
