package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/mapper"
	"github.com/gofiber/fiber/v2"
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
	// Create a temporary config file with valid mappings
	tmpDir := f.TempDir()
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
	if err != nil {
		f.Fatal(err)
	}

	// Create mapper
	m, err := mapper.NewMapper(configFile)
	if err != nil {
		f.Fatal(err)
	}

	// Create fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Ensure we always return a valid JSON response even for panic cases
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})
	setupRoutes(app, m)

	// Add seed corpus
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token"}`))                                  // Valid minimal input
	f.Add("test-mapper", "btoa", "custom", "", "", "", []byte(`{"@type": "koral:token"}`))                            // Valid with foundry override
	f.Add("", "", "", "", "", "", []byte(`{}`))                                                                       // Empty parameters
	f.Add("nonexistent", "invalid", "!@#$", "%^&*", "()", "[]", []byte(`invalid json`))                               // Invalid everything
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": null}`))                    // Valid JSON, invalid structure
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": {"@type": "unknown"}}`))    // Unknown type
	f.Add("test-mapper", "atob", "", "", "", "", []byte(`{"@type": "koral:token", "wrap": {"@type": "koral:term"}}`)) // Missing required fields

	f.Fuzz(func(t *testing.T, mapID, dir, foundryA, foundryB, layerA, layerB string, body []byte) {
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
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("invalid JSON response: %v", err)
		}

		// For error responses, verify that we have an error message
		if resp.StatusCode != http.StatusOK {
			if errMsg, ok := result["error"].(string); !ok || errMsg == "" {
				t.Error("error response missing error message")
			}
		}
	})
}
