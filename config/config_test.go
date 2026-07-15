package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/parser"
	"github.com/rs/zerolog/log"
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
	config, err := LoadFromSources(tmpfile.Name(), nil)
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

			_, err = LoadFromSources(tmpfile.Name(), nil)
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
			wantErr: "no mapping lists found",
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

			config, err := LoadFromSources(tmpfile.Name(), nil)
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
	config, err := LoadFromSources(tmpfile.Name(), nil)
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

			config, err := LoadFromSources(tmpfile.Name(), nil)
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
			assert.NotEmpty(t, config.Stylesheet)
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
	assert.Equal(t, defaultStylesheet, config.Stylesheet)
	assert.Equal(t, defaultServer, config.Server)
	require.Len(t, config.Lists, 1)
	assert.Equal(t, "test-mapper", config.Lists[0].ID)
}

func TestLoadFromSourcesDuplicateIDs(t *testing.T) {
	// Set up a buffer to capture log output
	var buf bytes.Buffer
	originalLogger := log.Logger
	defer func() {
		log.Logger = originalLogger
	}()
	log.Logger = log.Logger.Output(&buf)

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

	// The function should now succeed but log the duplicate ID error
	config, err := LoadFromSources(configFile.Name(), []string{mappingFile.Name()})
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check that the duplicate ID error was logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Duplicate mapping list ID found")
	assert.Contains(t, logOutput, "duplicate-id")

	// Only the first mapping list (from config file) should be loaded
	require.Len(t, config.Lists, 1)
	assert.Equal(t, "duplicate-id", config.Lists[0].ID)
	// Check that it's the one from the config file (has mapping "[A] <> [B]")
	assert.Equal(t, "[A] <> [B]", string(config.Lists[0].Mappings[0]))
}

func TestLoadFromSourcesConfigWithOnlyPort(t *testing.T) {
	// Create config file with only port (no lists)
	configContent := `
port: 8080
loglevel: debug
`
	configFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(configContent)
	require.NoError(t, err)
	err = configFile.Close()
	require.NoError(t, err)

	// Create mapping file
	mappingContent := `
id: test-mapper
mappings:
  - "[A] <> [B]"
`
	mappingFile, err := os.CreateTemp("", "mapping-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile.Name())

	_, err = mappingFile.WriteString(mappingContent)
	require.NoError(t, err)
	err = mappingFile.Close()
	require.NoError(t, err)

	// This should work: config file has only port, mapping file provides the mapping list
	config, err := LoadFromSources(configFile.Name(), []string{mappingFile.Name()})
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check that port and log level from config file are preserved
	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, "debug", config.LogLevel)

	// Check that mapping from mapping file is loaded
	require.Len(t, config.Lists, 1)
	assert.Equal(t, "test-mapper", config.Lists[0].ID)

	// Check that defaults are applied for other fields
	assert.Equal(t, defaultSDK, config.SDK)
	assert.Equal(t, defaultStylesheet, config.Stylesheet)
	assert.Equal(t, defaultServer, config.Server)
	assert.Equal(t, defaultServiceURL, config.ServiceURL)
}

func TestCorpusMappingListType(t *testing.T) {
	content := `
lists:
- id: corpus-class-mapping
  type: corpus
  desc: Maps textClass values to genre field
  mappings:
    - "textClass=novel <> genre=fiction"
    - "textClass=science <> genre=nonfiction"
- id: annotation-mapper
  mappings:
    - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-corpus-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	config, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	require.Len(t, config.Lists, 2)

	assert.Equal(t, "corpus", config.Lists[0].Type)
	assert.True(t, config.Lists[0].IsCorpus())

	assert.Equal(t, "", config.Lists[1].Type)
	assert.False(t, config.Lists[1].IsCorpus())
}

func TestParseCorpusMappings(t *testing.T) {
	list := &MappingList{
		ID:   "test-corpus",
		Type: "corpus",
		Mappings: []MappingRule{
			"textClass=novel <> genre=fiction",
			"(textClass=novel & pubDate=2020:geq#date) <> genre=recentfiction",
		},
	}

	results, err := list.ParseCorpusMappings()
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify simple field rule
	require.NotNil(t, results[0].Upper)
	require.NotNil(t, results[0].Lower)

	// Verify group rule
	require.NotNil(t, results[1].Upper)
	require.NotNil(t, results[1].Lower)
}

func TestParseCorpusMappingsErrors(t *testing.T) {
	list := &MappingList{
		ID:       "test-corpus",
		Type:     "corpus",
		Mappings: []MappingRule{""},
	}

	_, err := list.ParseCorpusMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty corpus mapping rule")

	list2 := &MappingList{
		ID:       "test-corpus",
		Type:     "corpus",
		Mappings: []MappingRule{"invalid rule without separator"},
	}

	_, err = list2.ParseCorpusMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse corpus mapping rule")
}

func TestApplyEnvOverrides(t *testing.T) {
	envKeys := []string{
		"KORAL_MAPPER_SERVER",
		"KORAL_MAPPER_SDK",
		"KORAL_MAPPER_STYLESHEET",
		"KORAL_MAPPER_SERVICE_URL",
		"KORAL_MAPPER_COOKIE_NAME",
		"KORAL_MAPPER_PORT",
		"KORAL_MAPPER_LOG_LEVEL",
		"KORAL_MAPPER_ALLOW_ORIGINS",
		"KORAL_MAPPER_REWRITES",
	}

	clearEnv := func() {
		for _, key := range envKeys {
			os.Unsetenv(key)
		}
	}

	t.Run("ENV overrides config values", func(t *testing.T) {
		clearEnv()
		defer clearEnv()

		cfg := &MappingConfig{
			Server:     "from-config",
			SDK:        "from-config",
			Stylesheet: "from-config",
			ServiceURL: "from-config",
			CookieName: "from-config",
			Port:       1234,
			LogLevel:   "warn",
		}

		os.Setenv("KORAL_MAPPER_SERVER", "from-env-server")
		os.Setenv("KORAL_MAPPER_SDK", "from-env-sdk")
		os.Setenv("KORAL_MAPPER_STYLESHEET", "from-env-style")
		os.Setenv("KORAL_MAPPER_SERVICE_URL", "from-env-url")
		os.Setenv("KORAL_MAPPER_COOKIE_NAME", "from-env-cookie")
		os.Setenv("KORAL_MAPPER_PORT", "9999")
		os.Setenv("KORAL_MAPPER_LOG_LEVEL", "debug")

		ApplyEnvOverrides(cfg)

		assert.Equal(t, "from-env-server", cfg.Server)
		assert.Equal(t, "from-env-sdk", cfg.SDK)
		assert.Equal(t, "from-env-style", cfg.Stylesheet)
		assert.Equal(t, "from-env-url", cfg.ServiceURL)
		assert.Equal(t, "from-env-cookie", cfg.CookieName)
		assert.Equal(t, 9999, cfg.Port)
		assert.Equal(t, "debug", cfg.LogLevel)
	})

	t.Run("Empty ENV does not override", func(t *testing.T) {
		clearEnv()
		defer clearEnv()

		cfg := &MappingConfig{
			Server:     "original-server",
			SDK:        "original-sdk",
			Stylesheet: "original-style",
			ServiceURL: "original-url",
			CookieName: "original-cookie",
			Port:       1234,
			LogLevel:   "info",
		}

		ApplyEnvOverrides(cfg)

		assert.Equal(t, "original-server", cfg.Server)
		assert.Equal(t, "original-sdk", cfg.SDK)
		assert.Equal(t, "original-style", cfg.Stylesheet)
		assert.Equal(t, "original-url", cfg.ServiceURL)
		assert.Equal(t, "original-cookie", cfg.CookieName)
		assert.Equal(t, 1234, cfg.Port)
		assert.Equal(t, "info", cfg.LogLevel)
	})

	t.Run("Invalid port ENV is ignored", func(t *testing.T) {
		clearEnv()
		defer clearEnv()

		cfg := &MappingConfig{Port: 5725}
		os.Setenv("KORAL_MAPPER_PORT", "not-a-number")

		ApplyEnvOverrides(cfg)

		assert.Equal(t, 5725, cfg.Port)
	})

	t.Run("Partial ENV overrides", func(t *testing.T) {
		clearEnv()
		defer clearEnv()

		cfg := &MappingConfig{
			Server:   "from-config",
			SDK:      "from-config",
			Port:     1234,
			LogLevel: "warn",
		}

		os.Setenv("KORAL_MAPPER_SERVER", "from-env")
		os.Setenv("KORAL_MAPPER_PORT", "8080")

		ApplyEnvOverrides(cfg)

		assert.Equal(t, "from-env", cfg.Server)
		assert.Equal(t, "from-config", cfg.SDK)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "warn", cfg.LogLevel)
	})
}

func TestBasePathEnvOverride(t *testing.T) {
	t.Setenv("KORAL_MAPPER_BASE_PATH", "/custom/base/path")

	cfg := &MappingConfig{BasePath: "from-config"}
	ApplyEnvOverrides(cfg)

	assert.Equal(t, "/custom/base/path", cfg.BasePath)
}

func TestBasePathFromYAML(t *testing.T) {
	content := `
basePath: "/opt/koralmapper"
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-basepath-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	assert.Equal(t, "/opt/koralmapper", cfg.BasePath)
}

func TestEnvOverridesInLoadFromSources(t *testing.T) {
	envKeys := []string{
		"KORAL_MAPPER_SERVER",
		"KORAL_MAPPER_SDK",
		"KORAL_MAPPER_PORT",
		"KORAL_MAPPER_LOG_LEVEL",
		"KORAL_MAPPER_STYLESHEET",
		"KORAL_MAPPER_SERVICE_URL",
		"KORAL_MAPPER_COOKIE_NAME",
		"KORAL_MAPPER_ALLOW_ORIGINS",
		"KORAL_MAPPER_REWRITES",
	}
	clearEnv := func() {
		for _, key := range envKeys {
			os.Unsetenv(key)
		}
	}
	clearEnv()
	defer clearEnv()

	configContent := `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
port: 3000
lists:
- id: test-mapper
  mappings:
    - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-env-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(configContent)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	os.Setenv("KORAL_MAPPER_SERVER", "https://env-override.example.com/")
	os.Setenv("KORAL_MAPPER_PORT", "7777")

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)

	// ENV overrides YAML values
	assert.Equal(t, "https://env-override.example.com/", cfg.Server)
	assert.Equal(t, 7777, cfg.Port)

	// Non-overridden values preserved from YAML
	assert.Equal(t, "https://custom.example.com/sdk.js", cfg.SDK)

	// Defaults applied for unset fields
	assert.Equal(t, defaultStylesheet, cfg.Stylesheet)
	assert.Equal(t, defaultServiceURL, cfg.ServiceURL)
	assert.Equal(t, defaultCookieName, cfg.CookieName)
	assert.Equal(t, defaultLogLevel, cfg.LogLevel)
}

func TestRewritesYAMLField(t *testing.T) {
	content := `
lists:
  - id: rewrite-on
    rewrites: true
    mappings:
      - "[A] <> [B]"
  - id: rewrite-off
    rewrites: false
    mappings:
      - "[C] <> [D]"
  - id: rewrite-default
    mappings:
      - "[E] <> [F]"
`
	tmpfile, err := os.CreateTemp("", "config-rewrites-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	require.Len(t, cfg.Lists, 3)

	require.NotNil(t, cfg.Lists[0].Rewrites, "rewrites should be set when specified as true")
	assert.True(t, *cfg.Lists[0].Rewrites, "rewrites should be true when set to true")
	require.NotNil(t, cfg.Lists[1].Rewrites, "rewrites should be set when specified as false")
	assert.False(t, *cfg.Lists[1].Rewrites, "rewrites should be false when set to false")
	assert.Nil(t, cfg.Lists[2].Rewrites, "rewrites should be nil when not specified")
}

func TestEffectiveRewrites(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name          string
		listRewrites  *bool
		globalDefault bool
		expected      bool
	}{
		{
			name:          "nil per-list, global false",
			listRewrites:  nil,
			globalDefault: false,
			expected:      false,
		},
		{
			name:          "nil per-list, global true",
			listRewrites:  nil,
			globalDefault: true,
			expected:      true,
		},
		{
			name:          "per-list true, global false",
			listRewrites:  &trueVal,
			globalDefault: false,
			expected:      true,
		},
		{
			name:          "per-list false, global true",
			listRewrites:  &falseVal,
			globalDefault: true,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := &MappingList{
				ID:       "test",
				Rewrites: tt.listRewrites,
				Mappings: []MappingRule{"[A] <> [B]"},
			}
			assert.Equal(t, tt.expected, list.EffectiveRewrites(tt.globalDefault))
		})
	}
}

func TestGlobalRewritesYAMLField(t *testing.T) {
	content := `
rewrites: true
lists:
  - id: inherits-global
    mappings:
      - "[A] <> [B]"
  - id: overrides-global
    rewrites: false
    mappings:
      - "[C] <> [D]"
`
	tmpfile, err := os.CreateTemp("", "config-global-rewrites-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)

	assert.True(t, cfg.Rewrites, "global rewrites should be true")

	assert.Nil(t, cfg.Lists[0].Rewrites, "per-list rewrites should be nil when not specified")
	assert.True(t, cfg.Lists[0].EffectiveRewrites(cfg.Rewrites),
		"list should inherit global rewrites=true")

	require.NotNil(t, cfg.Lists[1].Rewrites)
	assert.False(t, *cfg.Lists[1].Rewrites,
		"per-list rewrites should be false when explicitly set")
	assert.False(t, cfg.Lists[1].EffectiveRewrites(cfg.Rewrites),
		"list should override global rewrites=true with per-list false")
}

func TestGlobalRewritesDefaultFalse(t *testing.T) {
	content := `
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-global-rewrites-default-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)

	assert.False(t, cfg.Rewrites, "global rewrites should default to false")
}

func TestGlobalRewritesEnvOverride(t *testing.T) {
	t.Setenv("KORAL_MAPPER_REWRITES", "true")

	content := `
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-rewrites-env-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)

	assert.True(t, cfg.Rewrites,
		"KORAL_MAPPER_REWRITES=true env var should override default")
}

func TestGlobalRewritesEnvOverridesYAML(t *testing.T) {
	t.Setenv("KORAL_MAPPER_REWRITES", "false")

	content := `
rewrites: true
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-rewrites-env-yaml-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)

	assert.False(t, cfg.Rewrites,
		"KORAL_MAPPER_REWRITES=false env var should override YAML rewrites=true")
}

func TestParseCorpusMappingsWithFieldAFieldB(t *testing.T) {
	list := &MappingList{
		ID:     "test-keyed",
		Type:   "corpus",
		FieldA: "wikiCat",
		FieldB: "textClass",
		Mappings: []MappingRule{
			"Entertainment <> ((kultur & musik) | (kultur & film))",
		},
	}

	results, err := list.ParseCorpusMappings()
	require.NoError(t, err)
	require.Len(t, results, 1)

	upper := results[0].Upper.(*parser.CorpusField)
	assert.Equal(t, "wikiCat", upper.Key)
	assert.Equal(t, "Entertainment", upper.Value)

	group := results[0].Lower.(*parser.CorpusGroup)
	assert.Equal(t, "or", group.Operation)
	require.Len(t, group.Operands, 2)

	and1 := group.Operands[0].(*parser.CorpusGroup)
	assert.Equal(t, "textClass", and1.Operands[0].(*parser.CorpusField).Key)
	assert.Equal(t, "kultur", and1.Operands[0].(*parser.CorpusField).Value)
	assert.Equal(t, "textClass", and1.Operands[1].(*parser.CorpusField).Key)
	assert.Equal(t, "musik", and1.Operands[1].(*parser.CorpusField).Value)
}

func TestRateLimitConfigField(t *testing.T) {
	content := `
rateLimit: 50
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-ratelimit-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	assert.Equal(t, 50, cfg.RateLimit, "rateLimit should be loaded from YAML")
}

func TestRateLimitDefaultApplied(t *testing.T) {
	cfg := &MappingConfig{}
	ApplyDefaults(cfg)
	assert.Equal(t, defaultRateLimit, cfg.RateLimit,
		"default rate limit should be applied when not specified")
}

func TestRateLimitEnvOverride(t *testing.T) {
	t.Setenv("KORAL_MAPPER_RATE_LIMIT", "200")

	content := `
rateLimit: 50
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-ratelimit-env-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, cfg.RateLimit,
		"KORAL_MAPPER_RATE_LIMIT env var should override YAML value")
}

func TestAllowOriginsDefault(t *testing.T) {
	cfg := &MappingConfig{}
	ApplyDefaults(cfg)
	// AllowOrigins should derive from the Server default (trailing slash stripped)
	assert.Equal(t, []string{"https://korap.ids-mannheim.de"}, cfg.AllowOrigins,
		"default AllowOrigins should derive from defaultServer")
}

func TestAllowOriginsDerivedFromCustomServer(t *testing.T) {
	cfg := &MappingConfig{
		Server: "https://custom.example.com/",
	}
	ApplyDefaults(cfg)
	assert.Equal(t, []string{"https://custom.example.com"}, cfg.AllowOrigins,
		"AllowOrigins should derive from the configured Server (trailing slash stripped)")
}

func TestAllowOriginsExplicitNotOverriddenByServer(t *testing.T) {
	cfg := &MappingConfig{
		Server:       "https://custom.example.com/",
		AllowOrigins: []string{"https://explicit-origin.example.com"},
	}
	ApplyDefaults(cfg)
	assert.Equal(t, []string{"https://explicit-origin.example.com"}, cfg.AllowOrigins,
		"explicit AllowOrigins should not be overridden by Server default")
}

func TestAllowOriginsFromYAML(t *testing.T) {
	content := `
allowOrigins:
  - "https://custom.example.com"
  - "https://other.example.com"
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-cors-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://custom.example.com", "https://other.example.com"},
		cfg.AllowOrigins)
}

func TestAllowOriginsStringFormatRejected(t *testing.T) {
	content := `
allowOrigins: "https://custom.example.com,https://other.example.com"
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-cors-reject-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	_, err = LoadFromSources(tmpfile.Name(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowOrigins must be a YAML list")
}

func TestAllowOriginsEnvOverride(t *testing.T) {
	t.Setenv("KORAL_MAPPER_ALLOW_ORIGINS", "https://env-origin.example.com")

	content := `
allowOrigins:
  - "https://yaml-origin.example.com"
lists:
  - id: test-mapper
    mappings:
      - "[A] <> [B]"
`
	tmpfile, err := os.CreateTemp("", "config-cors-env-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources(tmpfile.Name(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://env-origin.example.com"}, cfg.AllowOrigins,
		"KORAL_MAPPER_ALLOW_ORIGINS env var should override YAML value")
}

func TestAllowOriginsDerivedFromServerWithPath(t *testing.T) {
	cfg := &MappingConfig{
		Server: "https://korap.ids-mannheim.de/instance/test",
	}
	ApplyDefaults(cfg)
	assert.Equal(t, []string{"https://korap.ids-mannheim.de"}, cfg.AllowOrigins,
		"AllowOrigins should be pruned to host-level origin when Server contains a path")
}

func TestAllowOriginsExplicitWithPathsPruned(t *testing.T) {
	cfg := &MappingConfig{
		AllowOrigins: []string{"https://korap.ids-mannheim.de/instance/test", "https://other.example.com/app"},
	}
	ApplyDefaults(cfg)
	assert.Equal(t, []string{"https://korap.ids-mannheim.de", "https://other.example.com"}, cfg.AllowOrigins,
		"explicit AllowOrigins entries should be pruned to host-level origins")
}

func TestAllowOriginsWithPort(t *testing.T) {
	cfg := &MappingConfig{
		Server: "https://korap.ids-mannheim.de:8080/instance/test",
	}
	ApplyDefaults(cfg)
	assert.Equal(t, []string{"https://korap.ids-mannheim.de:8080"}, cfg.AllowOrigins,
		"AllowOrigins should preserve port but strip path")
}

func TestSanitizeFilePathRejectsOutsideBase(t *testing.T) {
	// Set base to a specific directory and verify paths outside are rejected
	tmpDir, err := os.MkdirTemp("", "koral-base-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = tmpDir

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Path within base is accepted",
			input:   filepath.Join(tmpDir, "config.yaml"),
			wantErr: false,
		},
		{
			name:    "Path outside base is rejected",
			input:   "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "Traversal escaping base and tmp is rejected",
			input:   "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "Empty path is rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Subdirectory within base is accepted",
			input:   filepath.Join(tmpDir, "sub", "dir", "file.yaml"),
			wantErr: false,
		},
		{
			name:    "Relative path within base is rejected when CWD differs",
			input:   "config.yaml",
			wantErr: true, // resolves against CWD, not base
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeFilePath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, filepath.IsAbs(result),
				"sanitized path should be absolute, got: %s", result)
			assert.NotContains(t, result, "..")
		})
	}
}

func TestSanitizeFilePathTraversalToPasswd(t *testing.T) {
	// Verify /etc/passwd cannot be accessed via traversal
	cwd, err := os.Getwd()
	require.NoError(t, err)

	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = cwd

	_, err = sanitizeFilePath("../../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")
}

func TestSanitizeFilePathWithDockerRoot(t *testing.T) {
	// In Docker the WORKDIR is "/" -- all absolute paths should be valid
	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = "/"

	result, err := sanitizeFilePath("/mappings/stts-upos.yaml")
	require.NoError(t, err)
	assert.Equal(t, "/mappings/stts-upos.yaml", result)

	// Even deeply nested paths work when base is /
	result, err = sanitizeFilePath("/etc/ssl/certs/ca-certificates.crt")
	require.NoError(t, err)
	assert.Equal(t, "/etc/ssl/certs/ca-certificates.crt", result)
}

func TestSanitizeFilePathPrefixFalsePositive(t *testing.T) {
	// Ensure /home/user does not match /home/username
	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = "/home/user"

	_, err := sanitizeFilePath("/home/username/secret.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")
}

func TestLoadFromSourcesRejectsTraversal(t *testing.T) {
	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	AllowedBasePath = cwd

	// Config file traversal should be rejected
	_, err = LoadFromSources("../../../etc/passwd", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")

	// Mapping file traversal should be rejected
	_, err = LoadFromSources("", []string{"../../../etc/passwd"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")
}

func TestValidPathsStillWork(t *testing.T) {
	content := `
id: test-mapper
mappings:
  - "[A] <> [B]"
`
	tmpDir, err := os.MkdirTemp("", "koral-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = tmpDir

	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	tmpfile, err := os.CreateTemp(subDir, "mapping-*.yaml")
	require.NoError(t, err)

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	cfg, err := LoadFromSources("", []string{tmpfile.Name()})
	require.NoError(t, err)
	require.Len(t, cfg.Lists, 1)
	assert.Equal(t, "test-mapper", cfg.Lists[0].ID)
}

func TestRelativePathWithTraversalWithinBase(t *testing.T) {
	// Paths with ".." that still resolve within the base should work
	content := `
id: traversal-test-mapper
mappings:
  - "[A] <> [B]"
`
	tmpDir, err := os.MkdirTemp("", "koral-traversal-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origBase := AllowedBasePath
	defer func() { AllowedBasePath = origBase }()
	AllowedBasePath = tmpDir

	// Create file at tmpDir/config.yaml
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	// Reference via a traversal path: tmpDir/subdir/../config.yaml
	// This resolves to tmpDir/config.yaml which is within the base
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	traversalPath := filepath.Join(subDir, "..", "config.yaml")

	cfg, err := LoadFromSources("", []string{traversalPath})
	require.NoError(t, err)
	require.Len(t, cfg.Lists, 1)
	assert.Equal(t, "traversal-test-mapper", cfg.Lists[0].ID)
}
