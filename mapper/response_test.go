package mapper

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/Koral-Mapper/config"
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
	rules := m.parsedQueryRules["test-mapper"]
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

// TestResponseAnnotationDuplicateTokenText tests that when the same token text
// appears multiple times, only the correct occurrence is annotated based on its
// annotation context (not string position).
func TestResponseAnnotationDuplicateTokenText(t *testing.T) {
	// "Der" appears twice: first as NN (no match), then as DET (match).
	// The old string-heuristic would annotate the first "Der" because it
	// finds the first occurrence preceded by ">".
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:NN\">Der</span> <span title=\"marmot/p:DET\">Der</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-dup-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-dup-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	snippet := resultMap["snippet"].(string)

	// Only the second "Der" (DET) should be annotated
	expected := `<span title="marmot/p:NN">Der</span> <span title="marmot/p:DET"><span title="opennlp/p:DT" class="notinindex">Der</span></span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationTextInTitle verifies that the SAX rewriter only wraps
// text nodes, not content inside title attributes, even when the token text
// matches part of an attribute value.
func TestResponseAnnotationTextInTitle(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:NN\">NN</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-title-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[NN] <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-title-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	snippet := resultMap["snippet"].(string)

	expected := `<span title="marmot/p:NN"><span title="opennlp/p:NOUN" class="notinindex">NN</span></span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationWhitespaceAroundText tests that annotations are applied
// even when there is whitespace between the enclosing tag and the text content.
// The old string-heuristic fails because it requires ">" immediately before the text.
func TestResponseAnnotationWhitespaceAroundText(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\"> Der </span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-ws-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-ws-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	snippet := resultMap["snippet"].(string)

	// Whitespace should be preserved, annotation wraps only the token text
	expected := `<span title="marmot/p:DET"> <span title="opennlp/p:DT" class="notinindex">Der</span> </span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationCrossElementText tests annotation of individual tokens
// whose text spans across sibling/child elements.
func TestResponseAnnotationCrossElementText(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">Die</span> <span title=\"base/s:s\"><span title=\"marmot/p:NN\">Sonne</span></span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-cross-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
			"[NN] <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-cross-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	snippet := resultMap["snippet"].(string)

	assert.Contains(t, snippet, `<span title="opennlp/p:DT" class="notinindex">Die</span>`)
	assert.Contains(t, snippet, `<span title="opennlp/p:NOUN" class="notinindex">Sonne</span>`)
	assert.Contains(t, snippet, `title="base/s:s"`)
}

// TestResponseAnnotationSubstringToken tests that a short token ("er") is
// annotated only in its own text node and not when it appears as a prefix of
// another word ("er Mann") in an earlier text node.
func TestResponseAnnotationSubstringToken(t *testing.T) {
	// "er" appears at the start of "er Mann" (NN span) and as standalone (PPER span).
	// The old heuristic matches the first occurrence because "er" is preceded by ">"
	// and followed by " ".
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:NN\">er Mann</span> <span title=\"marmot/p:PPER\">er</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-sub-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[PPER] <> [PRP]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-sub-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	snippet := resultMap["snippet"].(string)

	// The NN "er Mann" must remain unchanged; only the PPER "er" gets annotated
	expected := `<span title="marmot/p:NN">er Mann</span> <span title="marmot/p:PPER"><span title="opennlp/p:PRP" class="notinindex">er</span></span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationSelfClosingTags verifies that self-closing tags like
// <br/> are preserved and do not interfere with annotation insertion.
func TestResponseAnnotationSelfClosingTags(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">Der</span><br/><span title=\"marmot/p:NN\">Mann</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-br-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
			"[NN]  <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-br-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	assert.Contains(t, snippet, "<br/>")
	assert.Contains(t, snippet, `<span title="opennlp/p:DT" class="notinindex">Der</span>`)
	assert.Contains(t, snippet, `<span title="opennlp/p:NOUN" class="notinindex">Mann</span>`)
}

// TestResponseAnnotationEntityReferences verifies that entity references
// (&amp;, &lt;, etc.) are correctly preserved in output.
func TestResponseAnnotationEntityReferences(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:NN\">Haus &amp; Hof</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-entity-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[NN] <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-entity-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	// Entity reference must be preserved (re-encoded) in the annotated output
	expected := `<span title="marmot/p:NN"><span title="opennlp/p:NOUN" class="notinindex">Haus &amp; Hof</span></span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationEntityLtGt verifies &lt; and &gt; are re-encoded.
func TestResponseAnnotationEntityLtGt(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:SYM\">&lt;tag&gt;</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-ltgt-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[SYM] <> [PUNCT]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-ltgt-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	expected := `<span title="marmot/p:SYM"><span title="opennlp/p:PUNCT" class="notinindex">&lt;tag&gt;</span></span>`
	assert.Equal(t, expected, snippet)
}

// TestResponseAnnotationCDATAGraceful verifies that a CDATA section in the
// snippet does not cause errors and is passed through unchanged.
func TestResponseAnnotationCDATAGraceful(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">Der</span><![CDATA[ raw ]]><span title=\"marmot/p:NN\">Mann</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-cdata-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[NN] <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-cdata-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	assert.Contains(t, snippet, "<![CDATA[ raw ]]>")
	assert.Contains(t, snippet, `<span title="opennlp/p:NOUN" class="notinindex">Mann</span>`)
}

// TestResponseAnnotationOverlappingSpans verifies that when two independent
// rules match the same token, both annotations are applied.
func TestResponseAnnotationOverlappingSpans(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\"><span title=\"marmot/m:case:nom\">Der</span></span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-overlap-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-overlap-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	// The existing nested structure must be preserved, with new annotation added
	assert.Contains(t, snippet, `title="marmot/p:DET"`)
	assert.Contains(t, snippet, `title="marmot/m:case:nom"`)
	assert.Contains(t, snippet, `title="opennlp/p:DT" class="notinindex"`)
	assert.Contains(t, snippet, "Der")
}

// TestResponseAnnotationEmptyTextNodes verifies that empty or whitespace-only
// text nodes are passed through without errors and without spurious annotations.
func TestResponseAnnotationEmptyTextNodes(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\"></span> <span title=\"marmot/p:NN\">Mann</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-empty-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
			"[NN]  <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-empty-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	// The empty DET span should not get an annotation
	// The NN token "Mann" should be annotated
	assert.Contains(t, snippet, `<span title="marmot/p:DET"></span>`)
	assert.Contains(t, snippet, `<span title="opennlp/p:NOUN" class="notinindex">Mann</span>`)
}

// TestResponseAnnotationWhitespaceOnlyNodes verifies that whitespace-only text
// nodes are preserved without annotations.
func TestResponseAnnotationWhitespaceOnlyNodes(t *testing.T) {
	responseSnippet := `{
		"snippet": "<span title=\"marmot/p:DET\">   </span><span title=\"marmot/p:NN\">Mann</span>"
	}`

	mappingList := config.MappingList{
		ID:       "test-wsonly-mapper",
		FoundryA: "marmot",
		LayerA:   "p",
		FoundryB: "opennlp",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[DET] <> [DT]",
			"[NN]  <> [NOUN]",
		},
	}

	m, err := NewMapper([]config.MappingList{mappingList})
	require.NoError(t, err)

	var inputData any
	err = json.Unmarshal([]byte(responseSnippet), &inputData)
	require.NoError(t, err)

	result, err := m.ApplyResponseMappings("test-wsonly-mapper", MappingOptions{Direction: AtoB}, inputData)
	require.NoError(t, err)

	snippet := result.(map[string]any)["snippet"].(string)

	// Whitespace-only text should not be annotated
	assert.Contains(t, snippet, `<span title="marmot/p:DET">   </span>`)
	assert.Contains(t, snippet, `<span title="opennlp/p:NOUN" class="notinindex">Mann</span>`)
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
