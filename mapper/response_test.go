package mapper

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResponseMappingAnnotationCreation tests creating new annotations based on RestrictToObligatory
func TestResponseMappingAnnotationCreation(t *testing.T) {
	// Simple snippet with a single annotated token
	responseSnippet := `{
		"snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
	}`

	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-mapper",
		FoundryA: "marmot",
		LayerA:   "m",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[gender:masc] <> [p=M & m=M]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// For step 4, we should at least get back a processed result (even if snippet is unchanged)
	// The main test is that no errors occurred in the processing
	assert.NotNil(t, result)

	// Verify the result is still a map with a snippet field
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")
	assert.Equal(t, "<span title=\"marmot/m:gender:masc\"><span title=\"opennlp/p:M\" class=\"notinindex\"><span title=\"opennlp/m:M\" class=\"notinindex\">Der</span></span></span>", resultMap["snippet"])
}

// TestResponseMappingDebug helps debug the mapping process
func TestResponseMappingDebug(t *testing.T) {
	// Simple snippet with a single annotated token
	responseSnippet := `{
		"snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
	}`

	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-mapper",
		FoundryA: "marmot",
		LayerA:   "m",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[gender=masc] <> [p=M & m=M]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	// Debug: Print what the parsed rules look like
	rules := m.parsedRules["test-mapper"]
	t.Logf("Number of parsed rules: %d", len(rules))
	for i, rule := range rules {
		t.Logf("Rule %d - Upper: %+v", i, rule.Upper)
		t.Logf("Rule %d - Lower: %+v", i, rule.Lower)
	}

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	// Include proper foundry and layer information in the options
	result, err := m.ApplyResponseMappings("test-mapper", MappingOptions{
		Direction: AtoB,
		FoundryA:  "marmot",
		LayerA:    "m",
		FoundryB:  "opennlp",
		LayerB:    "p",
	}, inputData)
	assert.Nil(t, err)
	t.Logf("Result: %+v", result)
}

// TestResponseMappingWithAndRelation tests mapping rules with AND relations
func TestResponseMappingWithAndRelation(t *testing.T) {
	// Snippet with multiple annotations on a single token - both must be on the same span for AND to work
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\"><span title=\"marmot/p:gender:masc\">Der</span></span>"
	}`

	// Create test mapping list with AND relation
	mappingList := config.MappingList{
		ID:       "test-and-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET & gender:masc] <> [p=DT & case=nom]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-and-mapper", MappingOptions{
		Direction: AtoB,
		FoundryA:  "marmot",
		LayerA:    "p",
		FoundryB:  "opennlp",
		LayerB:    "p",
	}, inputData)
	assert.Nil(t, err)

	// Verify the result contains the expected annotations
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	// Should contain both new annotations - checking the actual format produced
	assert.Contains(t, snippet, `title="marmot/p:DET"`)
	assert.Contains(t, snippet, `title="opennlp/p:DT"`)
	assert.Contains(t, snippet, `title="marmot/p:gender:masc"`)
	assert.Contains(t, snippet, `title="opennlp/case:nom"`) // Format is foundry/layer:value for single values
	assert.Contains(t, snippet, `class="notinindex"`)
}

// TestResponseMappingWithOrRelation tests mapping rules with OR relations
func TestResponseMappingWithOrRelation(t *testing.T) {
	// Snippet with one token that matches the OR condition
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">Der</span>"
	}`

	// Create test mapping list with OR relation
	mappingList := config.MappingList{
		ID:       "test-or-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET | ART] <> [determiner=true]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-or-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)

	assert.Contains(t, snippet, `title="marmot/p:DET"`)
	assert.Contains(t, snippet, `title="opennlp/determiner:true" class="notinindex"`)
	assert.NotEmpty(t, snippet)
}

// TestResponseMappingComplexPattern1 tests complex nested patterns
func TestResponseMappingComplexPattern1(t *testing.T) {
	// Snippet with a token that has nested annotations
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:ADJA\"><span title=\"marmot/m:gender:masc\"><span title=\"marmot/m:case:nom\">alter</span></span></span>"
	}`

	// Create test mapping list with complex pattern
	mappingList := config.MappingList{
		ID:       "test-complex-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[ADJA & gender=masc & case=nom] <> [pos=ADJ & gender=M & case=NOM]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-complex-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result contains the expected annotations
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	assert.Contains(t, snippet, `title="marmot/p:ADJA`)
	assert.Contains(t, snippet, `title="marmot/m:gender:masc`)
	assert.NotContains(t, snippet, `title="opennlp`)
	assert.NotEmpty(t, snippet) // At minimum, processing should succeed
}

// TestResponseMappingComplexPattern2 tests complex nested patterns
func TestResponseMappingComplexPattern2(t *testing.T) {
	// Snippet with a token that has nested annotations
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:ADJA\"><span title=\"marmot/p:gender:masc\"><span title=\"marmot/p:case:nom\">alter</span></span></span>"
	}`

	// Create test mapping list with complex pattern
	mappingList := config.MappingList{
		ID:       "test-complex-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[ADJA & gender:masc & case:nom] <> [pos=ADJ & gender=M & case=NOM]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-complex-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result contains the expected annotations
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	assert.Contains(t, snippet, `title="marmot/p:ADJA`)
	assert.Contains(t, snippet, `title="marmot/p:gender:masc`)
	assert.Contains(t, snippet, `title="opennlp/pos:ADJ" class="notinindex"`)
	assert.Contains(t, snippet, `title="opennlp/gender:M" class="notinindex"`)
	assert.Contains(t, snippet, `title="opennlp/case:NOM" class="notinindex"`)
	assert.NotEmpty(t, snippet) // At minimum, processing should succeed
}

// TestResponseMappingMultipleTokens tests mapping across multiple tokens
func TestResponseMappingMultipleTokens(t *testing.T) {
	// Snippet with multiple tokens
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">Der</span> <span title=\"marmot/p:ADJA\"><span title=\"marmot/m:gender:masc\">alte</span></span> <span title=\"marmot/p:NN\">Mann</span>"
	}`

	// Create test mapping list that matches multiple patterns
	mappingList := config.MappingList{
		ID:       "test-multi-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [determiner=true]",
			"[ADJA & gender:masc] <> [adjective=true & gender=M]",
			"[NN] <> [noun=true]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-multi-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	// Should contain annotations for each matching token (checking actual output format)
	assert.Contains(t, snippet, `title="marmot/p:DET"`)
	assert.Contains(t, snippet, `title="opennlp/determiner:true" class="notinindex"`) // Format is foundry/layer:value for single values
	assert.NotContains(t, snippet, `title="opennlp/adjective:true" class="notinindex"`)
	assert.Contains(t, snippet, `title="opennlp/noun:true" class="notinindex"`)
}

// TestResponseMappingNoMatch tests behavior when no patterns match
func TestResponseMappingNoMatch(t *testing.T) {
	// Snippet with tokens that don't match the pattern
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:VERB\">läuft</span>"
	}`

	// Create test mapping list with pattern that won't match
	mappingList := config.MappingList{
		ID:       "test-nomatch-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [determiner=true]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-nomatch-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result is unchanged since no patterns matched
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	// Should be the original snippet without new annotations
	assert.Equal(t, `<span title="marmot/p:VERB">läuft</span>`, snippet)
	assert.NotContains(t, snippet, `class="notinindex"`)
}

// TestResponseMappingBidirectional tests bidirectional mapping (B to A direction)
func TestResponseMappingBidirectional(t *testing.T) {
	// Snippet with opennlp annotations
	responseSnippet := `{
		"snippet": "<span title=\"opennlp/p:DT\"><span title=\"opennlp/p:determiner:true\">Der</span></span>"
	}`

	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-bidirectional-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT & determiner:true]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	// Test B to A direction
	result, err := m.ApplyResponseMappings("test-bidirectional-mapper", MappingOptions{Direction: BtoA}, inputData)
	assert.Nil(t, err)

	// Verify the result
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)

	assert.Contains(t, snippet, `title="opennlp/p:DT"`)
	assert.Contains(t, snippet, `title="marmot/p:DET" class="notinindex"`)
	assert.NotEmpty(t, snippet) // At minimum, processing should succeed
}

// TestResponseMappingWithValuePatterns tests patterns with specific values
func TestResponseMappingWithValuePatterns(t *testing.T) {
	// Snippet with value-specific annotations
	responseSnippet := `{
		"snippet": "<span title=\"marmot/m:case:nom\"><span title=\"marmot/m:gender:fem\">die</span></span>"
	}`

	// Create test mapping list with value-specific patterns
	mappingList := config.MappingList{
		ID:       "test-value-mapper",
		FoundryA: "marmot",
		LayerA:   "m",
		FoundryB: "opennlp",
		LayerB:   "m",
		Mappings: []config.MappingRule{
			"[case:nom & gender:fem] <> [case=NOM & gender=F]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-value-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	assert.Contains(t, snippet, `title="marmot/m:case:nom"`)                   // Format is foundry/layer:value
	assert.Contains(t, snippet, `title="opennlp/case:NOM" class="notinindex"`) // Format is foundry/layer:value
	assert.Contains(t, snippet, `title="opennlp/gender:F" class="notinindex"`)
}

// TestResponseMappingNestedSpans tests handling of deeply nested span structures
func TestResponseMappingNestedSpans(t *testing.T) {
	// Snippet with deeply nested spans
	responseSnippet := `{
		"snippet": "<span title=\"level1/l:outer\"><span title=\"level2/l:middle\"><span title=\"marmot/p:DET\">der</span></span></span>",
		"author": "John Doe"
	}`

	// Create test mapping list
	mappingList := config.MappingList{
		ID:       "test-nested-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [determiner=yes]",
		},
	}

	// Create a new mapper
	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	assert.Nil(t, err)

	result, err := m.ApplyResponseMappings("test-nested-mapper", MappingOptions{Direction: AtoB}, inputData)
	assert.Nil(t, err)

	// Verify the result preserves the nested structure and adds new annotations
	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, resultMap, "snippet")

	snippet := resultMap["snippet"].(string)
	// Should contain the new annotation while preserving existing structure
	assert.Contains(t, snippet, `title="opennlp/determiner:yes"`) // Format is foundry/layer:value
	assert.Contains(t, snippet, `class="notinindex"`)
	assert.Contains(t, snippet, `title="level1/l:outer"`)
	assert.Contains(t, snippet, `title="level2/l:middle"`)
	assert.Contains(t, snippet, `title="marmot/p:DET"`)

	author := resultMap["author"].(string)
	assert.Equal(t, "John Doe", author)
}

// TestResponseMappingWithLayerOverride tests layer precedence rules
func TestResponseMappingWithLayerOverride(t *testing.T) {
	// Test 1: Explicit layer in mapping rule should take precedence over MappingOptions
	t.Run("Explicit layer takes precedence", func(t *testing.T) {
		responseSnippet := `{
			"snippet": "<span title=\"marmot/p:DET\">Der</span>"
		}`

		// Mapping rule with explicit layer [p=DT] - this should NOT be overridden
		mappingList := config.MappingList{
			ID:       "test-layer-precedence",
			FoundryA: "marmot",
			LayerA:   "p",
			FoundryB: "opennlp",
			LayerB:   "p", // default layer
			Mappings: []config.MappingRule{
				"[DET] <> [p=DT]", // Explicit layer "p" should not be overridden
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		var inputData any
		err = json.Unmarshal([]byte(responseSnippet), &inputData)
		require.NoError(t, err)

		// Apply with layer override - should NOT affect explicit layer in mapping rule
		result, err := m.ApplyResponseMappings("test-layer-precedence", MappingOptions{
			Direction: AtoB,
			LayerB:    "pos", // This should NOT override the explicit "p" layer in [p=DT]
		}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		snippet := resultMap["snippet"].(string)

		// Should use explicit layer "p" from mapping rule, NOT "pos" from override
		assert.Contains(t, snippet, `title="opennlp/p:DT" class="notinindex"`)
		assert.NotContains(t, snippet, `title="opennlp/pos:DT" class="notinindex"`)
	})

	// Test 2: Implicit layer in mapping rule should use MappingOptions layer override
	t.Run("Implicit layer uses MappingOptions override", func(t *testing.T) {
		responseSnippet := `{
			"snippet": "<span title=\"marmot/p:DET\">Der</span>"
		}`

		// Mapping rule with implicit layer [DT] - this should use layer override
		mappingList := config.MappingList{
			ID:       "test-layer-override",
			FoundryA: "marmot",
			LayerA:   "p",
			FoundryB: "opennlp",
			LayerB:   "p", // default layer
			Mappings: []config.MappingRule{
				"[DET] <> [DT]", // No explicit layer - should use override
			},
		}

		m, err := NewMapper([]config.MappingList{mappingList})
		require.NoError(t, err)

		var inputData any
		err = json.Unmarshal([]byte(responseSnippet), &inputData)
		require.NoError(t, err)

		// Apply with layer override - should affect implicit layer in mapping rule
		result, err := m.ApplyResponseMappings("test-layer-override", MappingOptions{
			Direction: AtoB,
			LayerB:    "pos", // This should override the default layer for [DT]
		}, inputData)
		require.NoError(t, err)

		resultMap := result.(map[string]any)
		snippet := resultMap["snippet"].(string)

		// Should use layer "pos" from override, NOT default "p" layer
		assert.Contains(t, snippet, `title="opennlp/pos:DT" class="notinindex"`)
		assert.NotContains(t, snippet, `title="opennlp/p:DT" class="notinindex"`)
	})
}
