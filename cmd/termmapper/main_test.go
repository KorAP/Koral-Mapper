package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
