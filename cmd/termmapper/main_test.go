package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	tmconfig "github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/KorAP/KoralPipe-TermMapper/mapper"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformEndpoint(t *testing.T) {
	// Create test mapping list
	mappingList := tmconfig.MappingList{
		ID:       "test-mapper",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "p",
		Mappings: []tmconfig.MappingRule{
			"[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]",
			"[DET] <> [opennlp/p=DET]",
		},
	}

	// Create mapper
	m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
	require.NoError(t, err)

	// Create mock config for testing
	mockConfig := &tmconfig.MappingConfig{
		Lists: []tmconfig.MappingList{mappingList},
	}

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	tests := []struct {
		name          string
		mapID         string
		direction     string
		foundryA      string
		foundryB      string
		layerA        string
		layerB        string
		input         string
		expectedCode  int
		expectedBody  string
		expectedError string
	}{
		{
			name:      "Simple A to B mapping",
			mapID:     "test-mapper",
			direction: "atob",
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
			expectedCode: http.StatusOK,
			expectedBody: `{
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
			mapID:     "test-mapper",
			direction: "btoa",
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
			expectedCode: http.StatusOK,
			expectedBody: `{
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
			mapID:     "test-mapper",
			direction: "atob",
			foundryB:  "custom",
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
			expectedCode: http.StatusOK,
			expectedBody: `{
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
			name:          "Invalid mapping ID",
			mapID:         "nonexistent",
			direction:     "atob",
			input:         `{"@type": "koral:token"}`,
			expectedCode:  http.StatusInternalServerError,
			expectedError: "mapping list with ID nonexistent not found",
		},
		{
			name:          "Invalid direction",
			mapID:         "test-mapper",
			direction:     "invalid",
			input:         `{"@type": "koral:token"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid direction, must be 'atob' or 'btoa'",
		},
		{
			name:          "Invalid JSON",
			mapID:         "test-mapper",
			direction:     "atob",
			input:         `invalid json`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid JSON in request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build URL with query parameters
			url := "/" + tt.mapID + "/query"
			if tt.direction != "" {
				url += "?dir=" + tt.direction
			}
			if tt.foundryA != "" {
				url += "&foundryA=" + tt.foundryA
			}
			if tt.foundryB != "" {
				url += "&foundryB=" + tt.foundryB
			}
			if tt.layerA != "" {
				url += "&layerA=" + tt.layerA
			}
			if tt.layerB != "" {
				url += "&layerB=" + tt.layerB
			}

			// Make request
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(tt.input))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			// Read response body
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectedError != "" {
				// Check error message
				var errResp fiber.Map
				err = json.Unmarshal(body, &errResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errResp["error"])
			} else {
				// Compare JSON responses
				var expected, actual any
				err = json.Unmarshal([]byte(tt.expectedBody), &expected)
				require.NoError(t, err)
				err = json.Unmarshal(body, &actual)
				require.NoError(t, err)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Create test mapping list
	mappingList := tmconfig.MappingList{
		ID: "test-mapper",
		Mappings: []tmconfig.MappingRule{
			"[A] <> [B]",
		},
	}

	// Create mapper
	m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
	require.NoError(t, err)

	// Create mock config for testing
	mockConfig := &tmconfig.MappingConfig{
		Lists: []tmconfig.MappingList{mappingList},
	}

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "OK", string(body))

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "KoralPipe-TermMapper")

}

func TestKalamarPluginWithCustomSdkAndServer(t *testing.T) {
	// Create test mapping list
	mappingList := tmconfig.MappingList{
		ID: "test-mapper",
		Mappings: []tmconfig.MappingRule{
			"[A] <> [B]",
		},
	}

	// Create mapper
	m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
	require.NoError(t, err)

	tests := []struct {
		name           string
		customSDK      string
		customServer   string
		expectedSDK    string
		expectedServer string
	}{
		{
			name:           "Custom SDK and Server values",
			customSDK:      "https://custom.example.com/custom-sdk.js",
			customServer:   "https://custom.example.com/",
			expectedSDK:    "https://custom.example.com/custom-sdk.js",
			expectedServer: "https://custom.example.com/",
		},
		{
			name:           "Only custom SDK value",
			customSDK:      "https://custom.example.com/custom-sdk.js",
			customServer:   "https://korap.ids-mannheim.de/", // defaults applied during parsing
			expectedSDK:    "https://custom.example.com/custom-sdk.js",
			expectedServer: "https://korap.ids-mannheim.de/",
		},
		{
			name:           "Only custom Server value",
			customSDK:      "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", // defaults applied during parsing
			customServer:   "https://custom.example.com/",
			expectedSDK:    "https://korap.ids-mannheim.de/js/korap-plugin-latest.js",
			expectedServer: "https://custom.example.com/",
		},
		{
			name:           "Defaults applied during parsing",
			customSDK:      "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", // defaults applied during parsing
			customServer:   "https://korap.ids-mannheim.de/",                          // defaults applied during parsing
			expectedSDK:    "https://korap.ids-mannheim.de/js/korap-plugin-latest.js",
			expectedServer: "https://korap.ids-mannheim.de/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock config with custom values
			mockConfig := &tmconfig.MappingConfig{
				SDK:    tt.customSDK,
				Server: tt.customServer,
				Lists:  []tmconfig.MappingList{mappingList},
			}

			// Create fiber app
			app := fiber.New()
			setupRoutes(app, m, mockConfig)

			// Test Kalamar plugin endpoint
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			htmlContent := string(body)

			// Check that the HTML contains the expected SDK and Server values
			assert.Contains(t, htmlContent, `src="`+tt.expectedSDK+`"`)
			assert.Contains(t, htmlContent, `data-server="`+tt.expectedServer+`"`)

			// Ensure it's still a valid HTML page
			assert.Contains(t, htmlContent, "KoralPipe-TermMapper")
			assert.Contains(t, htmlContent, "<!DOCTYPE html>")
		})
	}
}

func TestMultipleMappingFiles(t *testing.T) {
	// Create test mapping files
	mappingFile1Content := `
id: test-mapper-1
foundryA: opennlp
layerA: p
foundryB: upos
layerB: p
mappings:
  - "[PIDAT] <> [DET & AdjType=Pdt]"
  - "[PAV] <> [ADV & PronType=Dem]"
`
	mappingFile1, err := os.CreateTemp("", "mapping1-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile1.Name())

	_, err = mappingFile1.WriteString(mappingFile1Content)
	require.NoError(t, err)
	err = mappingFile1.Close()
	require.NoError(t, err)

	mappingFile2Content := `
id: test-mapper-2
foundryA: stts
layerA: p
foundryB: upos
layerB: p
mappings:
  - "[DET] <> [PRON]"
  - "[ADJ] <> [NOUN]"
`
	mappingFile2, err := os.CreateTemp("", "mapping2-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile2.Name())

	_, err = mappingFile2.WriteString(mappingFile2Content)
	require.NoError(t, err)
	err = mappingFile2.Close()
	require.NoError(t, err)

	// Load configuration using multiple mapping files
	config, err := tmconfig.LoadFromSources("", []string{mappingFile1.Name(), mappingFile2.Name()})
	require.NoError(t, err)

	// Create mapper
	m, err := mapper.NewMapper(config.Lists)
	require.NoError(t, err)

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, config)

	// Test that both mappers work
	testCases := []struct {
		name        string
		mapID       string
		input       string
		expectGroup bool
		expectedKey string
	}{
		{
			name:  "test-mapper-1 with complex mapping",
			mapID: "test-mapper-1",
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
			expectGroup: true,  // This mapping creates a termGroup because of "&"
			expectedKey: "DET", // The first operand should be DET
		},
		{
			name:  "test-mapper-2 with simple mapping",
			mapID: "test-mapper-2",
			input: `{
				"@type": "koral:token",
				"wrap": {
					"@type": "koral:term",
					"foundry": "stts",
					"key": "DET",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expectGroup: false, // This mapping creates a simple term
			expectedKey: "PRON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/"+tc.mapID+"/query?dir=atob", bytes.NewBufferString(tc.input))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			// Check that the mapping was applied
			wrap := result["wrap"].(map[string]interface{})
			if tc.expectGroup {
				// For complex mappings, check the first operand
				assert.Equal(t, "koral:termGroup", wrap["@type"])
				operands := wrap["operands"].([]interface{})
				require.Greater(t, len(operands), 0)
				firstOperand := operands[0].(map[string]interface{})
				assert.Equal(t, tc.expectedKey, firstOperand["key"])
			} else {
				// For simple mappings, check the key directly
				assert.Equal(t, "koral:term", wrap["@type"])
				assert.Equal(t, tc.expectedKey, wrap["key"])
			}
		})
	}
}

func TestCombinedConfigAndMappingFiles(t *testing.T) {
	// Create main config file
	mainConfigContent := `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
lists:
- id: main-mapper
  foundryA: opennlp
  layerA: p
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

	// Create individual mapping file
	mappingFileContent := `
id: additional-mapper
foundryA: stts
layerA: p
mappings:
  - "[C] <> [D]"
`
	mappingFile, err := os.CreateTemp("", "mapping-*.yaml")
	require.NoError(t, err)
	defer os.Remove(mappingFile.Name())

	_, err = mappingFile.WriteString(mappingFileContent)
	require.NoError(t, err)
	err = mappingFile.Close()
	require.NoError(t, err)

	// Load configuration from both sources
	config, err := tmconfig.LoadFromSources(mainConfigFile.Name(), []string{mappingFile.Name()})
	require.NoError(t, err)

	// Verify that both mappers are loaded
	require.Len(t, config.Lists, 2)

	ids := make([]string, len(config.Lists))
	for i, list := range config.Lists {
		ids[i] = list.ID
	}
	assert.Contains(t, ids, "main-mapper")
	assert.Contains(t, ids, "additional-mapper")

	// Verify custom SDK and server are preserved from main config
	assert.Equal(t, "https://custom.example.com/sdk.js", config.SDK)
	assert.Equal(t, "https://custom.example.com/", config.Server)

	// Create mapper and test it works
	m, err := mapper.NewMapper(config.Lists)
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestExpandGlobs(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "glob_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files with .yaml and .yml extensions
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name: "mapper1.yaml",
			content: `
id: test-mapper-1
mappings:
  - "[A] <> [B]"
`,
		},
		{
			name: "mapper2.yml",
			content: `
id: test-mapper-2
mappings:
  - "[C] <> [D]"
`,
		},
		{
			name: "mapper3.yaml",
			content: `
id: test-mapper-3
mappings:
  - "[E] <> [F]"
`,
		},
		{
			name:    "other.txt",
			content: "not a yaml file",
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		err := os.WriteFile(filePath, []byte(file.content), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		patterns  []string
		expected  []string
		expectErr bool
	}{
		{
			name:     "Single literal file",
			patterns: []string{filepath.Join(tempDir, "mapper1.yaml")},
			expected: []string{filepath.Join(tempDir, "mapper1.yaml")},
		},
		{
			name:     "Multiple literal files",
			patterns: []string{filepath.Join(tempDir, "mapper1.yaml"), filepath.Join(tempDir, "mapper2.yml")},
			expected: []string{filepath.Join(tempDir, "mapper1.yaml"), filepath.Join(tempDir, "mapper2.yml")},
		},
		{
			name:     "Glob pattern for yaml files",
			patterns: []string{filepath.Join(tempDir, "*.yaml")},
			expected: []string{filepath.Join(tempDir, "mapper1.yaml"), filepath.Join(tempDir, "mapper3.yaml")},
		},
		{
			name:     "Glob pattern for yml files",
			patterns: []string{filepath.Join(tempDir, "*.yml")},
			expected: []string{filepath.Join(tempDir, "mapper2.yml")},
		},
		{
			name:     "Glob pattern for all yaml/yml files",
			patterns: []string{filepath.Join(tempDir, "*.y*ml")},
			expected: []string{
				filepath.Join(tempDir, "mapper1.yaml"),
				filepath.Join(tempDir, "mapper2.yml"),
				filepath.Join(tempDir, "mapper3.yaml"),
			},
		},
		{
			name:     "Mixed literal and glob",
			patterns: []string{filepath.Join(tempDir, "mapper1.yaml"), filepath.Join(tempDir, "*.yml")},
			expected: []string{filepath.Join(tempDir, "mapper1.yaml"), filepath.Join(tempDir, "mapper2.yml")},
		},
		{
			name:     "No matches - treats as literal",
			patterns: []string{filepath.Join(tempDir, "nonexistent*.yaml")},
			expected: []string{filepath.Join(tempDir, "nonexistent*.yaml")},
		},
		{
			name:      "Invalid glob pattern",
			patterns:  []string{filepath.Join(tempDir, "[")},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandGlobs(tt.patterns)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Sort both slices for comparison since glob results may not be in consistent order
			sort.Strings(result)
			sort.Strings(tt.expected)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobMappingFileLoading(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "glob_mapping_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test mapping files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name: "pos-mapper.yaml",
			content: `
id: pos-mapper
foundryA: opennlp
layerA: p
foundryB: upos
layerB: p
mappings:
  - "[PIDAT] <> [DET]"
  - "[ADJA] <> [ADJ]"
`,
		},
		{
			name: "ner-mapper.yml",
			content: `
id: ner-mapper
foundryA: opennlp
layerA: ner
foundryB: upos
layerB: ner
mappings:
  - "[PER] <> [PERSON]"
  - "[LOC] <> [LOCATION]"
`,
		},
		{
			name: "special-mapper.yaml",
			content: `
id: special-mapper
mappings:
  - "[X] <> [Y]"
`,
		},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		err := os.WriteFile(filePath, []byte(file.content), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		configFile     string
		mappingPattern string
		expectedIDs    []string
	}{
		{
			name:           "Load all yaml files",
			mappingPattern: filepath.Join(tempDir, "*.yaml"),
			expectedIDs:    []string{"pos-mapper", "special-mapper"},
		},
		{
			name:           "Load all yml files",
			mappingPattern: filepath.Join(tempDir, "*.yml"),
			expectedIDs:    []string{"ner-mapper"},
		},
		{
			name:           "Load all yaml/yml files",
			mappingPattern: filepath.Join(tempDir, "*-mapper.y*ml"),
			expectedIDs:    []string{"pos-mapper", "ner-mapper", "special-mapper"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expand the glob pattern
			expanded, err := expandGlobs([]string{tt.mappingPattern})
			require.NoError(t, err)

			// Load configuration using the expanded file list
			config, err := tmconfig.LoadFromSources(tt.configFile, expanded)
			require.NoError(t, err)

			// Verify that the expected mappers are loaded
			require.Len(t, config.Lists, len(tt.expectedIDs))

			actualIDs := make([]string, len(config.Lists))
			for i, list := range config.Lists {
				actualIDs[i] = list.ID
			}

			// Sort both slices for comparison
			sort.Strings(actualIDs)
			sort.Strings(tt.expectedIDs)
			assert.Equal(t, tt.expectedIDs, actualIDs)

			// Create mapper to ensure all loaded configs are valid
			m, err := mapper.NewMapper(config.Lists)
			require.NoError(t, err)
			require.NotNil(t, m)
		})
	}
}

func TestGlobErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		expectErr bool
	}{
		{
			name:      "Empty patterns",
			patterns:  []string{},
			expectErr: false, // Should return empty slice, no error
		},
		{
			name:      "Invalid glob pattern",
			patterns:  []string{"["},
			expectErr: true,
		},
		{
			name:      "Valid and invalid mixed",
			patterns:  []string{"valid.yaml", "["},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandGlobs(tt.patterns)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if len(tt.patterns) == 0 {
					assert.Empty(t, result)
				}
			}
		})
	}
}

func TestGlobIntegrationWithTestData(t *testing.T) {
	// Test that our glob functionality works with the actual testdata files
	// This ensures the feature works end-to-end in a realistic scenario

	// Expand glob pattern for the example mapper files
	expanded, err := expandGlobs([]string{"../../testdata/example-mapper*.yaml"})
	require.NoError(t, err)

	// Should match exactly the two mapper files
	sort.Strings(expanded)
	assert.Len(t, expanded, 2)
	assert.Contains(t, expanded[0], "example-mapper1.yaml")
	assert.Contains(t, expanded[1], "example-mapper2.yaml")

	// Load configuration using the expanded files
	config, err := tmconfig.LoadFromSources("", expanded)
	require.NoError(t, err)

	// Verify that both mappers are loaded correctly
	require.Len(t, config.Lists, 2)

	// Get the IDs to verify they match the expected ones
	actualIDs := make([]string, len(config.Lists))
	for i, list := range config.Lists {
		actualIDs[i] = list.ID
	}
	sort.Strings(actualIDs)

	expectedIDs := []string{"example-mapper-1", "example-mapper-2"}
	assert.Equal(t, expectedIDs, actualIDs)

	// Create mapper to ensure everything works
	m, err := mapper.NewMapper(config.Lists)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Test that the mapper actually works with a real transformation
	app := fiber.New()
	setupRoutes(app, m, config)

	// Test a transformation from example-mapper-1
	testInput := `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PIDAT",
			"layer": "p",
			"match": "match:eq"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/example-mapper-1/query?dir=atob", bytes.NewBufferString(testInput))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify the transformation was applied
	wrap := result["wrap"].(map[string]interface{})
	assert.Equal(t, "koral:termGroup", wrap["@type"])
	operands := wrap["operands"].([]interface{})
	require.Greater(t, len(operands), 0)
	firstOperand := operands[0].(map[string]interface{})
	assert.Equal(t, "DET", firstOperand["key"])
}

func TestConfigurableServiceURL(t *testing.T) {
	// Create test mapping list
	mappingList := tmconfig.MappingList{
		ID: "test-mapper",
		Mappings: []tmconfig.MappingRule{
			"[A] <> [B]",
		},
	}

	tests := []struct {
		name               string
		customServiceURL   string
		expectedServiceURL string
	}{
		{
			name:               "Custom service URL",
			customServiceURL:   "https://custom.example.com/plugin/termmapper",
			expectedServiceURL: "https://custom.example.com/plugin/termmapper",
		},
		{
			name:               "Default service URL when not specified",
			customServiceURL:   "", // Will use default
			expectedServiceURL: "https://korap.ids-mannheim.de/plugin/termmapper",
		},
		{
			name:               "Custom service URL with different path",
			customServiceURL:   "https://my-server.org/api/v1/termmapper",
			expectedServiceURL: "https://my-server.org/api/v1/termmapper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mapper
			m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
			require.NoError(t, err)

			// Create mock config with custom service URL
			mockConfig := &tmconfig.MappingConfig{
				ServiceURL: tt.customServiceURL,
				Lists:      []tmconfig.MappingList{mappingList},
			}

			// Apply defaults to simulate the real loading process
			tmconfig.ApplyDefaults(mockConfig)

			// Create fiber app
			app := fiber.New()
			setupRoutes(app, m, mockConfig)

			// Test Kalamar plugin endpoint with a specific mapID
			req := httptest.NewRequest(http.MethodGet, "/test-mapper", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			htmlContent := string(body)

			// Check that the HTML contains the expected service URL in the JavaScript
			expectedJSURL := tt.expectedServiceURL + "/test-mapper/query"
			assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL+"'")

			// Ensure it's still a valid HTML page
			assert.Contains(t, htmlContent, "KoralPipe-TermMapper")
			assert.Contains(t, htmlContent, "<!DOCTYPE html>")
		})
	}
}

func TestServiceURLConfigFileLoading(t *testing.T) {
	// Create a temporary config file with custom service URL
	configContent := `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
serviceURL: "https://custom.example.com/api/termmapper"
lists:
- id: config-mapper
  mappings:
    - "[X] <> [Y]"
`
	configFile, err := os.CreateTemp("", "service-url-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(configContent)
	require.NoError(t, err)
	err = configFile.Close()
	require.NoError(t, err)

	// Load configuration from file
	config, err := tmconfig.LoadFromSources(configFile.Name(), nil)
	require.NoError(t, err)

	// Verify that the service URL was loaded correctly
	assert.Equal(t, "https://custom.example.com/api/termmapper", config.ServiceURL)

	// Verify other fields are also preserved
	assert.Equal(t, "https://custom.example.com/sdk.js", config.SDK)
	assert.Equal(t, "https://custom.example.com/", config.Server)

	// Create mapper and test the service URL is used in the HTML
	m, err := mapper.NewMapper(config.Lists)
	require.NoError(t, err)

	app := fiber.New()
	setupRoutes(app, m, config)

	req := httptest.NewRequest(http.MethodGet, "/config-mapper", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	htmlContent := string(body)
	expectedJSURL := "https://custom.example.com/api/termmapper/config-mapper/query"
	assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL+"'")
}

func TestServiceURLDefaults(t *testing.T) {
	// Test that defaults are applied correctly when creating a config
	config := &tmconfig.MappingConfig{
		Lists: []tmconfig.MappingList{
			{
				ID:       "test",
				Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
			},
		},
	}

	// Apply defaults (simulating what happens during loading)
	tmconfig.ApplyDefaults(config)

	// Check that the default service URL was applied
	assert.Equal(t, "https://korap.ids-mannheim.de/plugin/termmapper", config.ServiceURL)

	// Check that other defaults were also applied
	assert.Equal(t, "https://korap.ids-mannheim.de/", config.Server)
	assert.Equal(t, "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", config.SDK)
	assert.Equal(t, 3000, config.Port)
	assert.Equal(t, "warn", config.LogLevel)
}

func TestServiceURLWithExampleConfig(t *testing.T) {
	// Test that the actual example config file works with the new serviceURL functionality
	// and that defaults are properly applied when serviceURL is not specified

	config, err := tmconfig.LoadFromSources("../../testdata/example-config.yaml", nil)
	require.NoError(t, err)

	// Verify that the default service URL was applied since it's not in the example config
	assert.Equal(t, "https://korap.ids-mannheim.de/plugin/termmapper", config.ServiceURL)

	// Verify other values from the example config are preserved
	assert.Equal(t, "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", config.SDK)
	assert.Equal(t, "https://korap.ids-mannheim.de/", config.Server)

	// Verify the mapper was loaded correctly
	require.Len(t, config.Lists, 1)
	assert.Equal(t, "main-config-mapper", config.Lists[0].ID)

	// Create mapper and test that the service URL is used correctly in the HTML
	m, err := mapper.NewMapper(config.Lists)
	require.NoError(t, err)

	app := fiber.New()
	setupRoutes(app, m, config)

	req := httptest.NewRequest(http.MethodGet, "/main-config-mapper", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	htmlContent := string(body)
	expectedJSURL := "https://korap.ids-mannheim.de/plugin/termmapper/main-config-mapper/query"
	assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL+"'")
}
