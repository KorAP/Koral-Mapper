package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tmconfig "github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/KorAP/KoralPipe-TermMapper/mapper"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzInput represents the input data for the fuzzer
type FuzzInput struct {
	MapID     string
	Direction string
	FoundryA  string
	FoundryB  string
	LayerA    string
	LayerB    string
	Body      []byte
}

func FuzzTransformEndpoint(f *testing.F) {
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
	if err != nil {
		f.Fatal(err)
	}

	// Create mock config for testing
	mockConfig := &tmconfig.MappingLists{
		Lists: []tmconfig.MappingList{mappingList},
	}

	// Create fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// For body limit errors, return 413 status code
			if err.Error() == "body size exceeds the given limit" || errors.Is(err, fiber.ErrRequestEntityTooLarge) {
				return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
					"error": fmt.Sprintf("request body too large (max %d bytes)", maxInputLength),
				})
			}
			// For other errors, return 500 status code
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
		BodyLimit: maxInputLength,
	})
	setupRoutes(app, m, mockConfig)

	// Add seed corpus
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token"}`))                                  // Valid minimal input
	f.Add("test-mapper", "btoa", "custom", "", "", "", []byte(`{"@type": "koral:token"}`))                            // Valid with foundry override
	f.Add("", "", "", "", "", "", []byte(`{}`))                                                                       // Empty parameters
	f.Add("nonexistent", "invalid", "!@#$", "%^&*", "()", "[]", []byte(`invalid json`))                               // Invalid everything
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": null}`))                    // Valid JSON, invalid structure
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": {"@type": "unknown"}}`))    // Unknown type
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": {"@type": "koral:term"}}`)) // Missing required fields
	f.Add("0", "0", strings.Repeat("\x83", 1000), "0", "Q", "", []byte("0"))                                          // Failing fuzz test case

	f.Fuzz(func(t *testing.T, mapID, dir, foundryA, foundryB, layerA, layerB string, body []byte) {

		// Validate input first
		if err := validateInput(mapID, dir, foundryA, foundryB, layerA, layerB, body); err != nil {
			// Skip this test case as it's invalid
			t.Skip(err)
		}

		// Build URL with query parameters
		params := url.Values{}
		if dir != "" {
			params.Set("dir", dir)
		}
		if foundryA != "" {
			params.Set("foundryA", foundryA)
		}
		if foundryB != "" {
			params.Set("foundryB", foundryB)
		}
		if layerA != "" {
			params.Set("layerA", layerA)
		}
		if layerB != "" {
			params.Set("layerB", layerB)
		}

		url := fmt.Sprintf("/%s/query", url.PathEscape(mapID))
		if len(params) > 0 {
			url += "?" + params.Encode()
		}

		// Make request
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Verify that we always get a valid response
		if resp.StatusCode != http.StatusOK &&
			resp.StatusCode != http.StatusBadRequest &&
			resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Verify that the response is valid JSON
		var result any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("invalid JSON response: %v", err)
		}

		// For error responses, verify that we have an error message
		if resp.StatusCode != http.StatusOK {
			// For error responses, we expect a JSON object with an error field
			if resultMap, ok := result.(map[string]any); ok {
				if errMsg, ok := resultMap["error"].(string); !ok || errMsg == "" {
					t.Error("error response missing error message")
				}
			} else {
				t.Error("error response should be a JSON object")
			}
		}
	})
}

func TestLargeInput(t *testing.T) {
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
	mockConfig := &tmconfig.MappingLists{
		Lists: []tmconfig.MappingList{mappingList},
	}

	// Create fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// For body limit errors, return 413 status code
			if err.Error() == "body size exceeds the given limit" || errors.Is(err, fiber.ErrRequestEntityTooLarge) {
				return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
					"error": fmt.Sprintf("request body too large (max %d bytes)", maxInputLength),
				})
			}
			// For other errors, return 500 status code
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
		BodyLimit: maxInputLength,
	})
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
		expectedError string
	}{
		{
			name:          "Large map ID",
			mapID:         strings.Repeat("a", maxParamLength+1),
			direction:     "atob",
			input:         "{}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "mapID too long (max 1024 bytes)",
		},
		{
			name:          "Large direction",
			mapID:         "test-mapper",
			direction:     strings.Repeat("a", maxParamLength+1),
			input:         "{}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "dir too long (max 1024 bytes)",
		},
		{
			name:          "Large foundryA",
			mapID:         "test-mapper",
			direction:     "atob",
			foundryA:      strings.Repeat("a", maxParamLength+1),
			input:         "{}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "foundryA too long (max 1024 bytes)",
		},
		{
			name:          "Invalid characters in mapID",
			mapID:         "test<>mapper",
			direction:     "atob",
			input:         "{}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "mapID contains invalid characters",
		},
		{
			name:          "Large request body",
			mapID:         "test-mapper",
			direction:     "atob",
			input:         strings.Repeat("a", maxInputLength+1),
			expectedCode:  http.StatusRequestEntityTooLarge,
			expectedError: "body size exceeds the given limit",
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
			req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			if resp == nil {
				assert.Equal(t, tt.expectedError, err.Error())
				return
			}

			require.NoError(t, err)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			// Check error message
			var result map[string]any
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			errMsg, ok := result["error"].(string)
			require.True(t, ok)
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}

// # Run fuzzing for 1 minute
// go test -fuzz=FuzzTransformEndpoint -fuzztime=1m ./cmd/termmapper
//
// # Run fuzzing until a crash is found or Ctrl+C is pressed
// go test -fuzz=FuzzTransformEndpoint ./cmd/termmapper
//
// # Run fuzzing with verbose output
// go test -fuzz=FuzzTransformEndpoint -v ./cmd/termmapper
//
// go test -run=FuzzTransformEndpoint/testdata/fuzz/FuzzTransformEndpoint/$SEED
