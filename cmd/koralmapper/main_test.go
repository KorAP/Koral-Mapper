package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tmconfig "github.com/KorAP/Koral-Mapper/config"
	"github.com/KorAP/Koral-Mapper/mapper"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadConfigFromYAML(t *testing.T, configYAML string, mappingYAMLs ...string) *tmconfig.MappingConfig {
	t.Helper()

	configPath := ""
	if configYAML != "" {
		cfgFile, err := os.CreateTemp("", "koralmapper-config-*.yaml")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Remove(cfgFile.Name()) })
		_, err = cfgFile.WriteString(configYAML)
		require.NoError(t, err)
		require.NoError(t, cfgFile.Close())
		configPath = cfgFile.Name()
	}

	mappingPaths := make([]string, 0, len(mappingYAMLs))
	for _, content := range mappingYAMLs {
		mapFile, err := os.CreateTemp("", "koralmapper-mapping-*.yaml")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Remove(mapFile.Name()) })
		_, err = mapFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, mapFile.Close())
		mappingPaths = append(mappingPaths, mapFile.Name())
	}

	cfg, err := tmconfig.LoadFromSources(configPath, mappingPaths)
	require.NoError(t, err)
	return cfg
}

func TestTransformEndpoint(t *testing.T) {
	cfg := loadConfigFromYAML(t, `
lists:
  - id: test-mapper
    foundryA: opennlp
    layerA: p
    foundryB: upos
    layerB: p
    mappings:
      - "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]"
      - "[DET] <> [opennlp/p=DET]"
`)

	// Create mapper
	m, err := mapper.NewMapper(cfg.Lists)
	require.NoError(t, err)

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, cfg)

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

func TestResponseTransformEndpoint(t *testing.T) {
	cfg := loadConfigFromYAML(t, `
lists:
  - id: test-response-mapper
    foundryA: marmot
    layerA: m
    foundryB: opennlp
    layerB: p
    mappings:
      - "[gender:masc] <> [p=M & m=M]"
`)

	// Create mapper
	m, err := mapper.NewMapper(cfg.Lists)
	require.NoError(t, err)

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, cfg)

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
			name:      "Simple response mapping with snippet transformation",
			mapID:     "test-response-mapper",
			direction: "atob",
			input: `{
				"snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
			}`,
			expectedCode: http.StatusOK,
			expectedBody: `{
				"snippet": "<span title=\"marmot/m:gender:masc\"><span title=\"opennlp/p:M\" class=\"notinindex\"><span title=\"opennlp/m:M\" class=\"notinindex\">Der</span></span></span>"
			}`,
		},
		{
			name:      "Response with no snippet field",
			mapID:     "test-response-mapper",
			direction: "atob",
			input: `{
				"@type": "koral:response",
				"meta": {
					"version": "Krill-0.64.1"
				}
			}`,
			expectedCode: http.StatusOK,
			expectedBody: `{
				"@type": "koral:response",
				"meta": {
					"version": "Krill-0.64.1"
				}
			}`,
		},
		{
			name:      "Response with null snippet",
			mapID:     "test-response-mapper",
			direction: "atob",
			input: `{
				"snippet": null
			}`,
			expectedCode: http.StatusOK,
			expectedBody: `{
				"snippet": null
			}`,
		},
		{
			name:      "Response with non-string snippet",
			mapID:     "test-response-mapper",
			direction: "atob",
			input: `{
				"snippet": 123
			}`,
			expectedCode: http.StatusOK,
			expectedBody: `{
				"snippet": 123
			}`,
		},
		{
			name:      "Response mapping with foundry override",
			mapID:     "test-response-mapper",
			direction: "atob",
			foundryB:  "custom",
			input: `{
				"snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
			}`,
			expectedCode: http.StatusOK,
			expectedBody: `{
				"snippet": "<span title=\"marmot/m:gender:masc\"><span title=\"custom/p:M\" class=\"notinindex\"><span title=\"custom/m:M\" class=\"notinindex\">Der</span></span></span>"
			}`,
		},
		{
			name:          "Invalid mapping ID for response",
			mapID:         "nonexistent",
			direction:     "atob",
			input:         `{"snippet": "<span>test</span>"}`,
			expectedCode:  http.StatusInternalServerError,
			expectedError: "mapping list with ID nonexistent not found",
		},
		{
			name:          "Invalid direction for response",
			mapID:         "test-response-mapper",
			direction:     "invalid",
			input:         `{"snippet": "<span>test</span>"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid direction, must be 'atob' or 'btoa'",
		},
		{
			name:          "Invalid JSON for response",
			mapID:         "test-response-mapper",
			direction:     "atob",
			input:         `{invalid json}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid JSON in request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build URL with query parameters
			url := "/" + tt.mapID + "/response"
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
	assert.Contains(t, string(body), "Koral-Mapper")

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
			assert.Contains(t, htmlContent, "Koral-Mapper")
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
			customServiceURL:   "https://custom.example.com/plugin/koralmapper",
			expectedServiceURL: "https://custom.example.com/plugin/koralmapper",
		},
		{
			name:               "Default service URL when not specified",
			customServiceURL:   "", // Will use default
			expectedServiceURL: "https://korap.ids-mannheim.de/plugin/koralmapper",
		},
		{
			name:               "Custom service URL with different path",
			customServiceURL:   "https://my-server.org/api/v1/koralmapper",
			expectedServiceURL: "https://my-server.org/api/v1/koralmapper",
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
			assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL)

			// Ensure it's still a valid HTML page
			assert.Contains(t, htmlContent, "Koral-Mapper")
			assert.Contains(t, htmlContent, "<!DOCTYPE html>")
		})
	}
}

func TestServiceURLConfigFileLoading(t *testing.T) {
	// Create a temporary config file with custom service URL
	configContent := `
sdk: "https://custom.example.com/sdk.js"
server: "https://custom.example.com/"
serviceURL: "https://custom.example.com/api/koralmapper"
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
	assert.Equal(t, "https://custom.example.com/api/koralmapper", config.ServiceURL)

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
	expectedJSURL := "https://custom.example.com/api/koralmapper/config-mapper/query"
	assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL)
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
	assert.Equal(t, "https://korap.ids-mannheim.de/plugin/koralmapper", config.ServiceURL)

	// Check that other defaults were also applied
	assert.Equal(t, "https://korap.ids-mannheim.de/", config.Server)
	assert.Equal(t, "https://korap.ids-mannheim.de/js/korap-plugin-latest.js", config.SDK)
	assert.Equal(t, 5725, config.Port)
	assert.Equal(t, "warn", config.LogLevel)
}

func TestServiceURLWithExampleConfig(t *testing.T) {
	// Test that the actual example config file works with the new serviceURL functionality
	// and that defaults are properly applied when serviceURL is not specified

	config, err := tmconfig.LoadFromSources("../../testdata/example-config.yaml", nil)
	require.NoError(t, err)

	// Verify that the default service URL was applied since it's not in the example config
	assert.Equal(t, "https://korap.ids-mannheim.de/plugin/koralmapper", config.ServiceURL)

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
	expectedJSURL := "https://korap.ids-mannheim.de/plugin/koralmapper/main-config-mapper/query"
	assert.Contains(t, htmlContent, "'service' : '"+expectedJSURL)
}

func TestBuildMapServiceURLWithURLJoining(t *testing.T) {
	tests := []struct {
		name       string
		serviceURL string
		mapID      string
		endpoint   string
		expected   string
	}{
		{
			name:       "Service URL without trailing slash",
			serviceURL: "https://example.com/plugin/koralmapper",
			mapID:      "test-mapper",
			endpoint:   "query",
			expected:   "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
		},
		{
			name:       "Service URL with trailing slash",
			serviceURL: "https://example.com/plugin/koralmapper/",
			mapID:      "test-mapper",
			endpoint:   "query",
			expected:   "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
		},
		{
			name:       "Map ID with leading slash",
			serviceURL: "https://example.com/plugin/koralmapper",
			mapID:      "/test-mapper",
			endpoint:   "query",
			expected:   "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
		},
		{
			name:       "Both with slashes",
			serviceURL: "https://example.com/plugin/koralmapper/",
			mapID:      "/test-mapper",
			endpoint:   "query",
			expected:   "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
		},
		{
			name:       "Complex map ID",
			serviceURL: "https://example.com/api/v1/",
			mapID:      "complex-mapper-name_123",
			endpoint:   "query",
			expected:   "https://example.com/api/v1/complex-mapper-name_123/query?dir=atob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildMapServiceURL(tt.serviceURL, tt.mapID, tt.endpoint, QueryParams{
				Dir:      "atob",
				FoundryA: "",
				FoundryB: "",
				LayerA:   "",
				LayerB:   "",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestKalamarPluginWithQueryParameters(t *testing.T) {
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

	// Create mock config
	mockConfig := &tmconfig.MappingConfig{
		ServiceURL: "https://example.com/plugin/koralmapper",
		Lists:      []tmconfig.MappingList{mappingList},
	}

	// Apply defaults
	tmconfig.ApplyDefaults(mockConfig)

	// Create fiber app
	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	tests := []struct {
		name             string
		url              string
		expectedQueryURL string
		expectedRespURL  string
		expectedStatus   int
		expectedError    string
	}{
		{
			name:             "Default parameters (no query params)",
			url:              "/test-mapper",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=btoa",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Explicit dir=atob",
			url:              "/test-mapper?dir=atob",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=btoa",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Explicit dir=btoa",
			url:              "/test-mapper?dir=btoa",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=btoa",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=atob",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "With foundry parameters",
			url:              "/test-mapper?dir=atob&foundryA=opennlp&foundryB=upos",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob&foundryA=opennlp&foundryB=upos",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=btoa&foundryA=opennlp&foundryB=upos",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "With layer parameters",
			url:              "/test-mapper?dir=btoa&layerA=pos&layerB=upos",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=btoa&layerA=pos&layerB=upos",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=atob&layerA=pos&layerB=upos",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "All parameters",
			url:              "/test-mapper?dir=atob&foundryA=opennlp&foundryB=upos&layerA=pos&layerB=upos",
			expectedQueryURL: "https://example.com/plugin/koralmapper/test-mapper/query?dir=atob&foundryA=opennlp&foundryB=upos&layerA=pos&layerB=upos",
			expectedRespURL:  "https://example.com/plugin/koralmapper/test-mapper/response?dir=btoa&foundryA=opennlp&foundryB=upos&layerA=pos&layerB=upos",
			expectedStatus:   http.StatusOK,
		},
		{
			name:           "Invalid direction",
			url:            "/test-mapper?dir=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid direction, must be 'atob' or 'btoa'",
		},
		{
			name:           "Parameter too long",
			url:            "/test-mapper?foundryA=" + strings.Repeat("a", 1025),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "foundryA too long (max 1024 bytes)",
		},
		{
			name:           "Invalid characters in parameter",
			url:            "/test-mapper?foundryA=invalid<>chars",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "foundryA contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectedError != "" {
				// Check error message
				var errResp fiber.Map
				err = json.Unmarshal(body, &errResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errResp["error"])
			} else {
				htmlContent := string(body)

				// Check that both query and response URLs are present with correct parameters
				assert.Contains(t, htmlContent, "'service' : '"+tt.expectedQueryURL+"'")
				assert.Contains(t, htmlContent, "'service' : '"+tt.expectedRespURL+"'")

				// Ensure it's still a valid HTML page
				assert.Contains(t, htmlContent, "Koral-Mapper")
				assert.Contains(t, htmlContent, "<!DOCTYPE html>")
			}
		})
	}
}

func TestBuildQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		foundryA string
		foundryB string
		layerA   string
		layerB   string
		expected string
	}{
		{
			name:     "Only direction parameter",
			dir:      "atob",
			expected: "dir=atob",
		},
		{
			name:     "All parameters",
			dir:      "btoa",
			foundryA: "opennlp",
			foundryB: "upos",
			layerA:   "pos",
			layerB:   "upos",
			expected: "dir=btoa&foundryA=opennlp&foundryB=upos&layerA=pos&layerB=upos",
		},
		{
			name:     "Some parameters empty",
			dir:      "atob",
			foundryA: "opennlp",
			foundryB: "",
			layerA:   "pos",
			layerB:   "",
			expected: "dir=atob&foundryA=opennlp&layerA=pos",
		},
		{
			name:     "All parameters empty",
			dir:      "",
			foundryA: "",
			foundryB: "",
			layerA:   "",
			layerB:   "",
			expected: "",
		},
		{
			name:     "URL encoding needed",
			dir:      "atob",
			foundryA: "test space",
			foundryB: "test&special",
			expected: "dir=atob&foundryA=test+space&foundryB=test%26special",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildQueryParams(tt.dir, tt.foundryA, tt.foundryB, tt.layerA, tt.layerB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompositeQueryEndpoint(t *testing.T) {
	cfg := loadConfigFromYAML(t, `
lists:
  - id: step1
    foundryA: opennlp
    layerA: p
    foundryB: opennlp
    layerB: p
    mappings:
      - "[PIDAT] <> [DET]"
  - id: step2
    foundryA: opennlp
    layerA: p
    foundryB: upos
    layerB: p
    mappings:
      - "[DET] <> [PRON]"
`)
	m, err := mapper.NewMapper(cfg.Lists)
	require.NoError(t, err)

	app := fiber.New()
	setupRoutes(app, m, cfg)

	tests := []struct {
		name         string
		url          string
		input        string
		expectedCode int
		expected     any
	}{
		{
			name:         "cascades two query mappings",
			url:          "/query?cfg=step1:atob;step2:atob",
			expectedCode: http.StatusOK,
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
			expected: map[string]any{
				"@type": "koral:token",
				"wrap": map[string]any{
					"@type":   "koral:term",
					"foundry": "upos",
					"key":     "PRON",
					"layer":   "p",
					"match":   "match:eq",
				},
			},
		},
		{
			name:         "empty cfg returns input unchanged",
			url:          "/query?cfg=",
			expectedCode: http.StatusOK,
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
			expected: map[string]any{
				"@type": "koral:token",
				"wrap": map[string]any{
					"@type":   "koral:term",
					"foundry": "opennlp",
					"key":     "PIDAT",
					"layer":   "p",
					"match":   "match:eq",
				},
			},
		},
		{
			name:         "invalid cfg returns bad request",
			url:          "/query?cfg=missing:atob",
			expectedCode: http.StatusBadRequest,
			input:        `{"@type": "koral:token"}`,
			expected: map[string]any{
				"error": `unknown mapping ID "missing"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, bytes.NewBufferString(tt.input))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var actual any
			err = json.Unmarshal(body, &actual)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestCompositeResponseEndpoint(t *testing.T) {
	cfg := loadConfigFromYAML(t, `
lists:
  - id: resp-step1
    type: corpus
    mappings:
      - "textClass=novel <> genre=fiction"
  - id: resp-step2
    type: corpus
    mappings:
      - "genre=fiction <> category=lit"
`)
	m, err := mapper.NewMapper(cfg.Lists)
	require.NoError(t, err)

	app := fiber.New()
	setupRoutes(app, m, cfg)

	tests := []struct {
		name         string
		url          string
		input        string
		expectedCode int
		assertBody   func(t *testing.T, actual map[string]any)
	}{
		{
			name:         "cascades two response mappings",
			url:          "/response?cfg=resp-step1:atob;resp-step2:atob",
			expectedCode: http.StatusOK,
			input: `{
				"fields": [{
					"@type": "koral:field",
					"key": "textClass",
					"value": "novel",
					"type": "type:string"
				}]
			}`,
			assertBody: func(t *testing.T, actual map[string]any) {
				fields := actual["fields"].([]any)
				require.Len(t, fields, 3)
				assert.Equal(t, "textClass", fields[0].(map[string]any)["key"])
				assert.Equal(t, "genre", fields[1].(map[string]any)["key"])
				assert.Equal(t, "fiction", fields[1].(map[string]any)["value"])
				assert.Equal(t, "category", fields[2].(map[string]any)["key"])
				assert.Equal(t, "lit", fields[2].(map[string]any)["value"])
			},
		},
		{
			name:         "empty cfg returns input unchanged",
			url:          "/response?cfg=",
			expectedCode: http.StatusOK,
			input: `{
				"fields": [{
					"@type": "koral:field",
					"key": "textClass",
					"value": "novel",
					"type": "type:string"
				}]
			}`,
			assertBody: func(t *testing.T, actual map[string]any) {
				fields := actual["fields"].([]any)
				require.Len(t, fields, 1)
				assert.Equal(t, "textClass", fields[0].(map[string]any)["key"])
				assert.Equal(t, "novel", fields[0].(map[string]any)["value"])
			},
		},
		{
			name:         "invalid cfg returns bad request",
			url:          "/response?cfg=resp-step1",
			expectedCode: http.StatusBadRequest,
			input:        `{"fields": []}`,
			assertBody: func(t *testing.T, actual map[string]any) {
				assert.Contains(t, actual["error"], "expected 2 or 6 colon-separated fields")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, bytes.NewBufferString(tt.input))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var actual map[string]any
			err = json.Unmarshal(body, &actual)
			require.NoError(t, err)
			tt.assertBody(t, actual)
		})
	}
}

func TestEmbeddedFilesExist(t *testing.T) {
	files := []string{"static/config.html", "static/plugin.html", "static/config.js", "static/style.css"}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			data, err := fs.ReadFile(staticFS, f)
			require.NoError(t, err, "embedded file %s should exist", f)
			assert.NotEmpty(t, data, "embedded file %s should not be empty", f)
		})
	}
}

func TestConfigTemplateParsesSuccessfully(t *testing.T) {
	tmpl, err := template.ParseFS(staticFS, "static/config.html")
	require.NoError(t, err, "config template should parse without error")
	require.NotNil(t, tmpl)
}

func TestStaticFileServing(t *testing.T) {
	mappingList := tmconfig.MappingList{
		ID:       "test-mapper",
		Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
	}
	m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
	require.NoError(t, err)
	mockConfig := &tmconfig.MappingConfig{Lists: []tmconfig.MappingList{mappingList}}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	tests := []struct {
		name          string
		url           string
		expectedCode  int
		expectedCType string
	}{
		{
			name:          "config.js is served with correct content type",
			url:           "/static/config.js",
			expectedCode:  http.StatusOK,
			expectedCType: "text/javascript",
		},
		{
			name:          "style.css is served with correct content type",
			url:           "/static/style.css",
			expectedCode:  http.StatusOK,
			expectedCType: "text/css",
		},
		{
			name:         "non-existent static file returns 404",
			url:          "/static/nonexistent.txt",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if tt.expectedCType != "" {
				assert.Contains(t, resp.Header.Get("Content-Type"), tt.expectedCType)
			}
		})
	}
}

func TestStaticFileContent(t *testing.T) {
	mappingList := tmconfig.MappingList{
		ID:       "test-mapper",
		Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
	}
	m, err := mapper.NewMapper([]tmconfig.MappingList{mappingList})
	require.NoError(t, err)
	mockConfig := &tmconfig.MappingConfig{Lists: []tmconfig.MappingList{mappingList}}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	// Verify served content matches embedded content
	files := []string{"config.js", "style.css"}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			embedded, err := fs.ReadFile(staticFS, "static/"+f)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/static/"+f, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, embedded, body, "served content should match embedded content for %s", f)
		})
	}
}

func TestConfigPageRendering(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:          "anno-mapper",
			Description: "Annotation mapping",
			FoundryA:    "opennlp",
			LayerA:      "p",
			FoundryB:    "upos",
			LayerB:      "p",
			Mappings:    []tmconfig.MappingRule{"[A] <> [B]"},
		},
		{
			ID:          "corpus-mapper",
			Type:        "corpus",
			Description: "Corpus mapping",
			Mappings:    []tmconfig.MappingRule{"textClass=science <> textClass=akademisch"},
		},
	}
	m, err := mapper.NewMapper(lists)
	require.NoError(t, err)

	mockConfig := &tmconfig.MappingConfig{
		SDK:        "https://example.com/sdk.js",
		Server:     "https://example.com/",
		ServiceURL: "https://example.com/plugin/koralmapper",
		Lists:      lists,
	}

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlContent := string(body)

	// HTML structure
	assert.Contains(t, htmlContent, "<!DOCTYPE html>")
	assert.Contains(t, htmlContent, `<meta charset="UTF-8">`)
	assert.Contains(t, htmlContent, "Koral-Mapper")

	// SDK and server
	assert.Contains(t, htmlContent, `src="https://example.com/sdk.js"`)
	assert.Contains(t, htmlContent, `data-server="https://example.com/"`)

	// ServiceURL as data attribute
	assert.Contains(t, htmlContent, `data-service-url="https://example.com/plugin/koralmapper"`)

	// Static file references
	assert.Contains(t, htmlContent, `/static/style.css`)
	assert.Contains(t, htmlContent, `/static/config.js`)

	// Annotation mapping section
	assert.Contains(t, htmlContent, "Query")
	assert.Contains(t, htmlContent, `data-id="anno-mapper"`)
	assert.Contains(t, htmlContent, `data-type="annotation"`)
	assert.Contains(t, htmlContent, `value="opennlp"`)
	assert.Contains(t, htmlContent, `value="upos"`)
	assert.Contains(t, htmlContent, "Annotation mapping")

	// Corpus mapping section
	assert.Contains(t, htmlContent, "Corpus")
	assert.Contains(t, htmlContent, `data-id="corpus-mapper"`)
	assert.Contains(t, htmlContent, `data-type="corpus"`)
	assert.Contains(t, htmlContent, "Corpus mapping")
}

func TestConfigPageAnnotationMappingHasFoundryInputs(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:       "anno-mapper",
			FoundryA: "opennlp",
			LayerA:   "p",
			FoundryB: "upos",
			LayerB:   "pos",
			Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
		},
	}
	m, err := mapper.NewMapper(lists)
	require.NoError(t, err)

	mockConfig := &tmconfig.MappingConfig{Lists: lists}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlContent := string(body)

	// Data attributes for default values
	assert.Contains(t, htmlContent, `data-default-foundry-a="opennlp"`)
	assert.Contains(t, htmlContent, `data-default-layer-a="p"`)
	assert.Contains(t, htmlContent, `data-default-foundry-b="upos"`)
	assert.Contains(t, htmlContent, `data-default-layer-b="pos"`)

	// Input fields with correct CSS classes
	assert.Contains(t, htmlContent, `class="foundryA"`)
	assert.Contains(t, htmlContent, `class="layerA"`)
	assert.Contains(t, htmlContent, `class="foundryB"`)
	assert.Contains(t, htmlContent, `class="layerB"`)

	// Direction arrow
	assert.Contains(t, htmlContent, `class="dir-arrow"`)
	assert.Contains(t, htmlContent, `data-dir="atob"`)

	// Request and response checkboxes
	assert.Contains(t, htmlContent, `class="request-cb"`)
	assert.Contains(t, htmlContent, `class="response-cb"`)
}

func TestConfigPageCorpusMappingHasNoFoundryInputs(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:       "corpus-mapper",
			Type:     "corpus",
			Mappings: []tmconfig.MappingRule{"textClass=science <> textClass=akademisch"},
		},
	}

	m, err := mapper.NewMapper(lists)
	require.NoError(t, err)

	mockConfig := &tmconfig.MappingConfig{Lists: lists}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlContent := string(body)

	// Corpus section exists
	assert.Contains(t, htmlContent, `data-id="corpus-mapper"`)
	assert.Contains(t, htmlContent, `data-type="corpus"`)

	// Checkboxes present
	assert.Contains(t, htmlContent, `class="request-cb"`)
	assert.Contains(t, htmlContent, `class="response-cb"`)

	// No foundry/layer inputs (only corpus mappings, no annotation section)
	assert.NotContains(t, htmlContent, `class="foundryA"`)
	assert.NotContains(t, htmlContent, `class="dir-arrow"`)
}

func TestConfigPageBackwardCompatibility(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:       "test-mapper",
			Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
		},
	}

	m, err := mapper.NewMapper(lists)
	require.NoError(t, err)

	mockConfig := &tmconfig.MappingConfig{
		ServiceURL: "https://example.com/plugin/koralmapper",
		Lists:      lists,
	}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	req := httptest.NewRequest(http.MethodGet, "/test-mapper", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlContent := string(body)

	// Old-style single-mapping page behavior
	assert.Contains(t, htmlContent, "<!DOCTYPE html>")
	assert.Contains(t, htmlContent, "Koral-Mapper")
	assert.Contains(t, htmlContent, "Map ID: test-mapper")
	assert.Contains(t, htmlContent, "KorAPlugin.sendMsg")
	assert.Contains(t, htmlContent, "test-mapper/query")
}

func TestBuildConfigPageData(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:          "anno1",
			Description: "First annotation",
			FoundryA:    "f1a",
			LayerA:      "l1a",
			FoundryB:    "f1b",
			LayerB:      "l1b",
			Mappings:    []tmconfig.MappingRule{"[A] <> [B]"},
		},
		{
			ID:          "corpus1",
			Type:        "corpus",
			Description: "First corpus",
			Mappings:    []tmconfig.MappingRule{"textClass=a <> textClass=b"},
		},
		{
			ID:          "anno2",
			Type:        "annotation",
			Description: "Second annotation",
			FoundryA:    "f2a",
			LayerA:      "l2a",
			FoundryB:    "f2b",
			LayerB:      "l2b",
			Mappings:    []tmconfig.MappingRule{"[C] <> [D]"},
		},
	}

	mockConfig := &tmconfig.MappingConfig{
		SDK:        "https://example.com/sdk.js",
		Server:     "https://example.com/",
		ServiceURL: "https://example.com/service",
		Lists:      lists,
	}

	data := buildConfigPageData(mockConfig)

	assert.Equal(t, "https://example.com/sdk.js", data.SDK)
	assert.Equal(t, "https://example.com/", data.Server)
	assert.Equal(t, "https://example.com/service", data.ServiceURL)

	require.Len(t, data.AnnotationMappings, 2)
	assert.Equal(t, "anno1", data.AnnotationMappings[0].ID)
	assert.Equal(t, "annotation", data.AnnotationMappings[0].Type)
	assert.Equal(t, "f1a", data.AnnotationMappings[0].FoundryA)
	assert.Equal(t, "First annotation", data.AnnotationMappings[0].Description)
	assert.Equal(t, "anno2", data.AnnotationMappings[1].ID)
	assert.Equal(t, "annotation", data.AnnotationMappings[1].Type)

	require.Len(t, data.CorpusMappings, 1)
	assert.Equal(t, "corpus1", data.CorpusMappings[0].ID)
	assert.Equal(t, "corpus", data.CorpusMappings[0].Type)
	assert.Equal(t, "First corpus", data.CorpusMappings[0].Description)
}

func TestConfigPagePreservesOrderOfMappings(t *testing.T) {
	lists := []tmconfig.MappingList{
		{
			ID:       "mapper-z",
			FoundryA: "fa",
			FoundryB: "fb",
			Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
		},
		{
			ID:       "mapper-a",
			FoundryA: "fa",
			FoundryB: "fb",
			Mappings: []tmconfig.MappingRule{"[C] <> [D]"},
		},
		{
			ID:       "mapper-m",
			FoundryA: "fa",
			FoundryB: "fb",
			Mappings: []tmconfig.MappingRule{"[E] <> [F]"},
		},
	}
	m, err := mapper.NewMapper(lists)
	require.NoError(t, err)

	mockConfig := &tmconfig.MappingConfig{Lists: lists}
	tmconfig.ApplyDefaults(mockConfig)

	app := fiber.New()
	setupRoutes(app, m, mockConfig)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlContent := string(body)

	// Verify the order is preserved (z before a before m)
	idxZ := strings.Index(htmlContent, `data-id="mapper-z"`)
	idxA := strings.Index(htmlContent, `data-id="mapper-a"`)
	idxM := strings.Index(htmlContent, `data-id="mapper-m"`)
	assert.Greater(t, idxA, idxZ, "mapper-a should appear after mapper-z")
	assert.Greater(t, idxM, idxA, "mapper-m should appear after mapper-a")
}
