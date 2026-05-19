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

	// After both steps, the term should have rewrites from both steps:
	// step 1 recorded scope=foundry original=opennlp and scope=key original=PIDAT,
	// step 2 recorded scope=foundry original=stts and scope=key original=DET.
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
					"scope": "foundry",
					"original": "opennlp"
				},
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"scope": "key",
					"original": "PIDAT"
				},
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"scope": "foundry",
					"original": "stts"
				},
				{
					"@type": "koral:rewrite",
					"editor": "Koral-Mapper",
					"scope": "key",
					"original": "DET"
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

	// Step 1 rewrites (scope=foundry original=opennlp, scope=key original=PIDAT)
	// must appear on the TermGroup created by step 2, along with step 2's
	// own structural rewrite.
	resultMap := result.(map[string]any)
	wrap := resultMap["wrap"].(map[string]any)
	require.Equal(t, "koral:termGroup", wrap["@type"])

	rewrites := wrap["rewrites"].([]any)
	// First rewrite is from step 1 (carried forward): foundry change
	rw0 := rewrites[0].(map[string]any)
	assert.Equal(t, "foundry", rw0["scope"])
	assert.Equal(t, "opennlp", rw0["original"])

	// Second rewrite is from step 1 (carried forward): key change
	rw1 := rewrites[1].(map[string]any)
	assert.Equal(t, "key", rw1["scope"])
	assert.Equal(t, "PIDAT", rw1["original"])

	// Last rewrite is from step 2 (structural: original is the full term)
	rwLast := rewrites[len(rewrites)-1].(map[string]any)
	assert.Equal(t, "Koral-Mapper", rwLast["editor"])
	// Structural rewrite stores the full original node (no scope)
	original := rwLast["original"].(map[string]any)
	assert.Equal(t, "koral:term", original["@type"])
	assert.Equal(t, "DET", original["key"])
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
