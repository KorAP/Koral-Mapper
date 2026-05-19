package mapper

import (
	"encoding/json"
	"testing"

	"github.com/KorAP/Koral-Mapper/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseJSON is a test helper that unmarshals a JSON string.
func parseJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal([]byte(s), &v))
	return v
}

func TestCascadeQueryTwoAnnotationMappings(t *testing.T) {
	m, err := NewMapper([]config.MappingList{
		{
			ID: "ann-step1", FoundryA: "opennlp", LayerA: "p",
			FoundryB: "stts", LayerB: "p",
			Mappings: []config.MappingRule{`[PIDAT] <> [DET]`},
		},
		{
			ID: "ann-step2", FoundryA: "stts", LayerA: "p",
			FoundryB: "upos", LayerB: "p",
			Mappings: []config.MappingRule{`[DET] <> [PRON]`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PIDAT",
			"layer": "p",
			"match": "match:eq"
		}
	}`)

	result, err := m.CascadeQueryMappings(
		[]string{"ann-step1", "ann-step2"},
		[]MappingOptions{{Direction: AtoB}, {Direction: AtoB}},
		input,
	)
	require.NoError(t, err)

	expected := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "upos",
			"key": "PRON",
			"layer": "p",
			"match": "match:eq"
		}
	}`)
	assert.Equal(t, expected, result)
}

func TestCascadeQueryMixAnnotationAndCorpus(t *testing.T) {
	m, err := NewMapper([]config.MappingList{
		{
			ID: "ann-mapper", FoundryA: "opennlp", LayerA: "p",
			FoundryB: "upos", LayerB: "p",
			Mappings: []config.MappingRule{`[PIDAT] <> [DET]`},
		},
		{
			ID:       "corpus-mapper",
			Type:     "corpus",
			Mappings: []config.MappingRule{`textClass=novel <> genre=fiction`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"query": {
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "PIDAT",
				"layer": "p",
				"match": "match:eq"
			}
		},
		"collection": {
			"@type": "koral:doc",
			"key": "textClass",
			"value": "novel",
			"match": "match:eq"
		}
	}`)

	result, err := m.CascadeQueryMappings(
		[]string{"ann-mapper", "corpus-mapper"},
		[]MappingOptions{{Direction: AtoB}, {Direction: AtoB}},
		input,
	)
	require.NoError(t, err)

	resultMap := result.(map[string]any)

	query := resultMap["query"].(map[string]any)
	wrap := query["wrap"].(map[string]any)
	assert.Equal(t, "DET", wrap["key"])
	assert.Equal(t, "upos", wrap["foundry"])

	collection := resultMap["collection"].(map[string]any)
	assert.Equal(t, "genre", collection["key"])
	assert.Equal(t, "fiction", collection["value"])
}

func TestCascadeQuerySingleElement(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "single", FoundryA: "opennlp", LayerA: "p",
		FoundryB: "upos", LayerB: "p",
		Mappings: []config.MappingRule{`[PIDAT] <> [DET]`},
	}})
	require.NoError(t, err)

	makeInput := func() any {
		return parseJSON(t, `{
			"@type": "koral:token",
			"wrap": {
				"@type": "koral:term",
				"foundry": "opennlp",
				"key": "PIDAT",
				"layer": "p",
				"match": "match:eq"
			}
		}`)
	}

	opts := MappingOptions{Direction: AtoB}

	cascadeResult, err := m.CascadeQueryMappings(
		[]string{"single"}, []MappingOptions{opts}, makeInput(),
	)
	require.NoError(t, err)

	directResult, err := m.ApplyQueryMappings("single", opts, makeInput())
	require.NoError(t, err)

	assert.Equal(t, directResult, cascadeResult)
}

func TestCascadeQueryEmptyList(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "dummy", FoundryA: "x", LayerA: "y",
		FoundryB: "a", LayerB: "b",
		Mappings: []config.MappingRule{`[X] <> [Y]`},
	}})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {"@type": "koral:term", "key": "Z"}
	}`)

	result, err := m.CascadeQueryMappings(nil, nil, input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestCascadeQueryUnknownID(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "known", FoundryA: "x", LayerA: "y",
		FoundryB: "a", LayerB: "b",
		Mappings: []config.MappingRule{`[X] <> [Y]`},
	}})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {"@type": "koral:term", "key": "X"}
	}`)

	_, err = m.CascadeQueryMappings(
		[]string{"known", "nonexistent"},
		[]MappingOptions{{Direction: AtoB}, {Direction: AtoB}},
		input,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

// --- Response cascade tests ---

func TestCascadeQueryRewritesPreservedAcrossSteps(t *testing.T) {
	// Step 1 changes foundry (opennlp->stts) and key (PIDAT->DET).
	// Step 2 changes foundry (stts->upos) and key (DET->PRON).
	// Rewrites from step 1 must survive step 2's replacement.
	m, err := NewMapper([]config.MappingList{
		{
			ID: "step1", FoundryA: "opennlp", LayerA: "p",
			FoundryB: "stts", LayerB: "p",
			Mappings: []config.MappingRule{`[PIDAT] <> [DET]`},
		},
		{
			ID: "step2", FoundryA: "stts", LayerA: "p",
			FoundryB: "upos", LayerB: "p",
			Mappings: []config.MappingRule{`[DET] <> [PRON]`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PIDAT",
			"layer": "p",
			"match": "match:eq"
		}
	}`)

	result, err := m.CascadeQueryMappings(
		[]string{"step1", "step2"},
		[]MappingOptions{
			{Direction: AtoB, AddRewrites: true},
			{Direction: AtoB, AddRewrites: true},
		},
		input,
	)
	require.NoError(t, err)

	// After both steps, the term should have one rewrite per step:
	// step 1 recorded the full original term (opennlp/p/PIDAT),
	// step 2 recorded the full original term (stts/p/DET).
	expected := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "upos",
			"key": "PRON",
			"layer": "p",
			"match": "match:eq",
			"rewrites": [
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"original": {
						"@type": "koral:term",
						"foundry": "opennlp",
						"key": "PIDAT",
						"layer": "p",
						"match": "match:eq"
					}
				},
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"original": {
						"@type": "koral:term",
						"foundry": "stts",
						"key": "DET",
						"layer": "p",
						"match": "match:eq"
					}
				}
			]
		}
	}`)
	assert.Equal(t, expected, result)
}

func TestCascadeQueryRewritesPreservedStructuralChange(t *testing.T) {
	// Step 1 changes foundry (opennlp->stts) and key (PIDAT->DET).
	// Step 2 replaces Term with TermGroup (structural change).
	// Rewrites from step 1 must be carried into the new TermGroup.
	m, err := NewMapper([]config.MappingList{
		{
			ID: "sc-step1", FoundryA: "opennlp", LayerA: "p",
			FoundryB: "stts", LayerB: "p",
			Mappings: []config.MappingRule{`[PIDAT] <> [DET]`},
		},
		{
			ID: "sc-step2", FoundryA: "stts", LayerA: "p",
			FoundryB: "tt", LayerB: "pos",
			Mappings: []config.MappingRule{`[DET] <> [opennlp/p=DET & opennlp/p=PronType:Art]`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"@type": "koral:token",
		"wrap": {
			"@type": "koral:term",
			"foundry": "opennlp",
			"key": "PIDAT",
			"layer": "p",
			"match": "match:eq"
		}
	}`)

	result, err := m.CascadeQueryMappings(
		[]string{"sc-step1", "sc-step2"},
		[]MappingOptions{
			{Direction: AtoB, AddRewrites: true},
			{Direction: AtoB, AddRewrites: true},
		},
		input,
	)
	require.NoError(t, err)

	// Step 1 produced a single rewrite with the full original term (opennlp/p/PIDAT).
	// Step 2 replaced the term with a TermGroup (structural change) and produced
	// another single rewrite with the full original term (stts/p/DET).
	// Both rewrites must appear on the TermGroup.
	resultMap := result.(map[string]any)
	wrap := resultMap["wrap"].(map[string]any)
	require.Equal(t, "koral:termGroup", wrap["@type"])

	rewrites := wrap["rewrites"].([]any)
	require.Len(t, rewrites, 2)

	// First rewrite is from step 1 (carried forward): full original term
	rw0 := rewrites[0].(map[string]any)
	assert.Equal(t, "Koral-Mapper", rw0["editor"])
	original0 := rw0["original"].(map[string]any)
	assert.Equal(t, "koral:term", original0["@type"])
	assert.Equal(t, "PIDAT", original0["key"])
	assert.Equal(t, "opennlp", original0["foundry"])

	// Second rewrite is from step 2: full original term
	rw1 := rewrites[1].(map[string]any)
	assert.Equal(t, "Koral-Mapper", rw1["editor"])
	original1 := rw1["original"].(map[string]any)
	assert.Equal(t, "koral:term", original1["@type"])
	assert.Equal(t, "DET", original1["key"])
}

func TestCascadeResponseTwoCorpusMappings(t *testing.T) {
	m, err := NewMapper([]config.MappingList{
		{
			ID: "corpus-step1", Type: "corpus",
			Mappings: []config.MappingRule{`textClass=novel <> genre=fiction`},
		},
		{
			ID: "corpus-step2", Type: "corpus",
			Mappings: []config.MappingRule{`genre=fiction <> category=lit`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"fields": [{
			"@type": "koral:field",
			"key": "textClass",
			"value": "novel",
			"type": "type:string"
		}]
	}`)

	result, err := m.CascadeResponseMappings(
		[]string{"corpus-step1", "corpus-step2"},
		[]MappingOptions{{Direction: AtoB}, {Direction: AtoB}},
		input,
	)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.GreaterOrEqual(t, len(fields), 3)

	assert.Equal(t, "textClass", fields[0].(map[string]any)["key"])

	assert.Equal(t, "genre", fields[1].(map[string]any)["key"])
	assert.Equal(t, "fiction", fields[1].(map[string]any)["value"])

	assert.Equal(t, "category", fields[2].(map[string]any)["key"])
	assert.Equal(t, "lit", fields[2].(map[string]any)["value"])
}

func TestCascadeResponseMixAnnotationAndCorpus(t *testing.T) {
	m, err := NewMapper([]config.MappingList{
		{
			ID: "ann-resp", FoundryA: "opennlp", LayerA: "p",
			FoundryB: "upos", LayerB: "p",
			Mappings: []config.MappingRule{`[DET] <> [PRON]`},
		},
		{
			ID:       "corpus-resp",
			Type:     "corpus",
			Mappings: []config.MappingRule{`textClass=novel <> genre=fiction`},
		},
	})
	require.NoError(t, err)

	input := parseJSON(t, `{
		"snippet": "<span title=\"opennlp/p:DET\">Der</span>",
		"fields": [{
			"@type": "koral:field",
			"key": "textClass",
			"value": "novel",
			"type": "type:string"
		}]
	}`)

	result, err := m.CascadeResponseMappings(
		[]string{"ann-resp", "corpus-resp"},
		[]MappingOptions{{Direction: AtoB}, {Direction: AtoB}},
		input,
	)
	require.NoError(t, err)

	resultMap := result.(map[string]any)

	snippet := resultMap["snippet"].(string)
	assert.Contains(t, snippet, "opennlp/p:DET")
	assert.Contains(t, snippet, "upos/p:PRON")

	fields := resultMap["fields"].([]any)
	require.GreaterOrEqual(t, len(fields), 2)
	assert.Equal(t, "genre", fields[1].(map[string]any)["key"])
}

func TestCascadeResponseSingleElement(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "corpus-single", Type: "corpus",
		Mappings: []config.MappingRule{`textClass=novel <> genre=fiction`},
	}})
	require.NoError(t, err)

	makeInput := func() any {
		return parseJSON(t, `{
			"fields": [{
				"@type": "koral:field",
				"key": "textClass",
				"value": "novel",
				"type": "type:string"
			}]
		}`)
	}

	opts := MappingOptions{Direction: AtoB}

	cascadeResult, err := m.CascadeResponseMappings(
		[]string{"corpus-single"}, []MappingOptions{opts}, makeInput(),
	)
	require.NoError(t, err)

	directResult, err := m.ApplyResponseMappings("corpus-single", opts, makeInput())
	require.NoError(t, err)

	assert.Equal(t, directResult, cascadeResult)
}

func TestCascadeResponseEmptyList(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "dummy", Type: "corpus",
		Mappings: []config.MappingRule{`x=y <> a=b`},
	}})
	require.NoError(t, err)

	input := parseJSON(t, `{"fields": []}`)

	result, err := m.CascadeResponseMappings(nil, nil, input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestCascadeResponseUnknownID(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID: "known", Type: "corpus",
		Mappings: []config.MappingRule{`x=y <> a=b`},
	}})
	require.NoError(t, err)

	_, err = m.CascadeResponseMappings(
		[]string{"nonexistent"},
		[]MappingOptions{{Direction: AtoB}},
		parseJSON(t, `{"fields": []}`),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}
