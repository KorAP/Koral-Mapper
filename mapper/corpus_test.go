package mapper

import (
	"testing"

	"github.com/KorAP/Koral-Mapper/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCorpusMapper(t *testing.T, rules ...string) *Mapper {
	t.Helper()
	mappingRules := make([]config.MappingRule, len(rules))
	for i, r := range rules {
		mappingRules[i] = config.MappingRule(r)
	}
	m, err := NewMapper([]config.MappingList{{
		ID:       "corpus-test",
		Type:     "corpus",
		Mappings: mappingRules,
	}})
	require.NoError(t, err)
	return m
}

// --- Corpus query mapping tests ---

func TestCorpusQuerySimpleFieldRewrite(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:doc", corpus["@type"])
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
	assert.Equal(t, "match:eq", corpus["match"])
}

func TestCorpusQueryNoMatch(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "science",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "textClass", corpus["key"])
	assert.Equal(t, "science", corpus["value"])
}

func TestCorpusQueryBtoA(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "genre",
			"value": "fiction",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "textClass", corpus["key"])
	assert.Equal(t, "novel", corpus["value"])
}

func TestCorpusQueryDocGroupRecursive(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "novel",
					"match": "match:eq",
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "author",
					"value": "Fontane",
					"match": "match:eq",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:and", corpus["operation"])

	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)

	first := operands[0].(map[string]any)
	assert.Equal(t, "genre", first["key"])
	assert.Equal(t, "fiction", first["value"])

	second := operands[1].(map[string]any)
	assert.Equal(t, "author", second["key"])
	assert.Equal(t, "Fontane", second["value"])
}

func TestCorpusQueryDocGroupRefPassthrough(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:docGroupRef",
			"ref":   "https://korap.ids-mannheim.de/@ndiewald/MyCorpus",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroupRef", corpus["@type"])
	assert.Equal(t, "https://korap.ids-mannheim.de/@ndiewald/MyCorpus", corpus["ref"])
}

func TestCorpusQueryFieldAlias(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:field",
			"key":   "textClass",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryFieldOverridesAtoB(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID:       "corpus-test",
		Type:     "corpus",
		FieldA:   "textClass",
		FieldB:   "genre",
		Mappings: []config.MappingRule{"novel <> fiction"},
	}})
	require.NoError(t, err)

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "domain",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{
		Direction: AtoB,
		FieldA:    "domain",
		FieldB:    "subject",
	}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "subject", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryFieldOverridesBtoA(t *testing.T) {
	m, err := NewMapper([]config.MappingList{{
		ID:       "corpus-test",
		Type:     "corpus",
		FieldA:   "textClass",
		FieldB:   "genre",
		Mappings: []config.MappingRule{"novel <> fiction"},
	}})
	require.NoError(t, err)

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "subject",
			"value": "fiction",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{
		Direction: BtoA,
		FieldA:    "domain",
		FieldB:    "subject",
	}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "domain", corpus["key"])
	assert.Equal(t, "novel", corpus["value"])
}

func TestCorpusQueryFieldGroupAlias(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:fieldGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{
					"@type": "koral:field",
					"key":   "textClass",
					"value": "novel",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	operands := corpus["operands"].([]any)
	first := operands[0].(map[string]any)
	assert.Equal(t, "genre", first["key"])
}

func TestCorpusQueryCollectionAttribute(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"collection": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["collection"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQuerySingleToGroupReplacement(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> (genre=fiction & type=book)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:and", corpus["operation"])

	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)
	assert.Equal(t, "genre", operands[0].(map[string]any)["key"])
	assert.Equal(t, "type", operands[1].(map[string]any)["key"])
}

func TestCorpusQueryRegexMatch(t *testing.T) {
	m := newCorpusMapper(t, "textClass=wissenschaft.*#regex <> genre=science")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "wissenschaft-populaer",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "science", corpus["value"])
}

func TestCorpusQueryRegexNoMatch(t *testing.T) {
	m := newCorpusMapper(t, "textClass=wissenschaft.*#regex <> genre=science")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "belletristik",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "textClass", corpus["key"])
	assert.Equal(t, "belletristik", corpus["value"])
}

func TestCorpusQueryMatchTypeFilter(t *testing.T) {
	m := newCorpusMapper(t, "pubDate=2020:geq <> yearFrom=2020:geq")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "pubDate",
			"value": "2020",
			"match": "match:geq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "yearFrom", corpus["key"])
	assert.Equal(t, "match:geq", corpus["match"])
}

func TestCorpusQueryMatchTypeFilterNoMatchTest(t *testing.T) {
	m := newCorpusMapper(t, "pubDate=2020 <> yearFrom=2020")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "pubDate",
			"value": "2020",
			"match": "match:geq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "yearFrom", corpus["key"])
	assert.Equal(t, "match:geq", corpus["match"])
}

func TestCorpusQueryMatchTypeFilterNoMatch(t *testing.T) {
	m := newCorpusMapper(t, "pubDate=2020:geq <> yearFrom=2020:geq")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "pubDate",
			"value": "2020",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "pubDate", corpus["key"])
}

func TestCorpusQueryRewriteAnnotation(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
			"match": "match:eq",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB, AddRewrites: true}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])

	rewrites, ok := corpus["rewrites"].([]any)
	require.True(t, ok)
	require.Len(t, rewrites, 1)

	rewrite := rewrites[0].(map[string]any)
	assert.Equal(t, "koral:rewrite", rewrite["@type"])
	assert.Equal(t, "Koral-Mapper", rewrite["editor"])
}

func TestCorpusQueryPreservesMatchTypeFromOriginal(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
			"match": "match:contains",
			"type":  "type:string",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "match:contains", corpus["match"])
	assert.Equal(t, "type:string", corpus["type"])
}

func TestCorpusQueryNoCorpusSection(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"query": map[string]any{"@type": "koral:token"},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestCorpusQueryMultipleRules(t *testing.T) {
	m := newCorpusMapper(t,
		"textClass=novel <> genre=fiction",
		"textClass=science <> genre=nonfiction",
	)

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "novel",
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "science",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)
	assert.Equal(t, "genre", operands[0].(map[string]any)["key"])
	assert.Equal(t, "fiction", operands[0].(map[string]any)["value"])
	assert.Equal(t, "genre", operands[1].(map[string]any)["key"])
	assert.Equal(t, "nonfiction", operands[1].(map[string]any)["value"])
}

func TestCorpusQueryNestedDocGroups(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{
					"@type":     "koral:docGroup",
					"operation": "operation:or",
					"operands": []any{
						map[string]any{
							"@type": "koral:doc",
							"key":   "textClass",
							"value": "novel",
						},
					},
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "author",
					"value": "Fontane",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	outerOperands := corpus["operands"].([]any)
	innerGroup := outerOperands[0].(map[string]any)
	innerOperands := innerGroup["operands"].([]any)
	assert.Equal(t, "genre", innerOperands[0].(map[string]any)["key"])
}

// --- Corpus response mapping tests ---

func TestCorpusResponseSimpleFieldEnrichment(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "genre",
				"value": "fiction",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 2)

	original := fields[0].(map[string]any)
	assert.Equal(t, "genre", original["key"])

	mapped := fields[1].(map[string]any)
	assert.Equal(t, "textClass", mapped["key"])
	assert.Equal(t, "novel", mapped["value"])
	assert.Equal(t, true, mapped["mapped"])
}

func TestCorpusResponseNoMatch(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "author",
				"value": "Fontane",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 1)
}

func TestCorpusResponseMultiValuedField(t *testing.T) {
	m := newCorpusMapper(t,
		"textClass=wissenschaft <> genre=science",
		"textClass=populaerwissenschaft <> genre=popsci",
	)

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": []any{"wissenschaft", "populaerwissenschaft"},
				"type":  "type:keywords",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 3)

	mapped1 := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped1["key"])
	assert.Equal(t, "science", mapped1["value"])
	assert.Equal(t, true, mapped1["mapped"])

	mapped2 := fields[2].(map[string]any)
	assert.Equal(t, "genre", mapped2["key"])
	assert.Equal(t, "popsci", mapped2["value"])
	assert.Equal(t, true, mapped2["mapped"])
}

func TestCorpusResponseRegexMatch(t *testing.T) {
	m := newCorpusMapper(t, "textClass=wissenschaft.*#regex <> genre=science")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "wissenschaft-populaer",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 2)

	mapped := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped["key"])
	assert.Equal(t, "science", mapped["value"])
}

func TestCorpusResponseDocTypeAlias(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:doc",
				"key":   "genre",
				"value": "fiction",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 2)

	mapped := fields[1].(map[string]any)
	assert.Equal(t, "textClass", mapped["key"])
}

func TestCorpusResponseGroupReplacement(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> (genre=fiction & type=book)")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "novel",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 3)

	mapped1 := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped1["key"])
	assert.Equal(t, "fiction", mapped1["value"])
	assert.Equal(t, true, mapped1["mapped"])

	mapped2 := fields[2].(map[string]any)
	assert.Equal(t, "type", mapped2["key"])
	assert.Equal(t, "book", mapped2["value"])
	assert.Equal(t, true, mapped2["mapped"])
}

func TestCorpusResponseNoFieldsSection(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"snippet": "<span>test</span>",
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestCorpusResponseDirectionAtoB(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "novel",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 2)

	mapped := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped["key"])
	assert.Equal(t, "fiction", mapped["value"])
}

func TestCorpusQueryValueTypeInReplacement(t *testing.T) {
	m := newCorpusMapper(t, "pubDate=2020-01#date <> publicationYear=2020#string")

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "pubDate",
			"value": "2020-01",
			"match": "match:eq",
			"type":  "type:date",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "publicationYear", corpus["key"])
	assert.Equal(t, "2020", corpus["value"])
	assert.Equal(t, "type:string", corpus["type"])
}

func TestCorpusQueryMappingListNotFound(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")
	_, err := m.ApplyQueryMappings("nonexistent", MappingOptions{Direction: AtoB}, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCorpusResponseMappingListNotFound(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> genre=fiction")
	_, err := m.ApplyResponseMappings("nonexistent", MappingOptions{Direction: AtoB}, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Group pattern matching tests ---

func TestCorpusQueryANDGroupPatternMatchBtoA(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "kultur",
					"match": "match:eq",
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "musik",
					"match": "match:eq",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:doc", corpus["@type"])
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryANDGroupPatternCommutative(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	// Operands in reversed order — should still match
	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "musik",
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "kultur",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryANDGroupPatternNoMatchWrongOp(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	// OR operation doesn't match AND pattern
	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "kultur",
				},
				map[string]any{
					"@type": "koral:doc",
					"key":   "textClass",
					"value": "musik",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
}

func TestCorpusQueryANDGroupPatternSubsetMatch(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	// Three operands: pattern matches the two matching operands (subset),
	// the unmatched operand is preserved alongside the replacement.
	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
				map[string]any{"@type": "koral:doc", "key": "pubDate", "value": "2020"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:and", corpus["operation"])

	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)

	// First operand is the replacement
	first := operands[0].(map[string]any)
	assert.Equal(t, "genre", first["key"])
	assert.Equal(t, "fiction", first["value"])

	// Second operand is the preserved unmatched operand
	second := operands[1].(map[string]any)
	assert.Equal(t, "pubDate", second["key"])
	assert.Equal(t, "2020", second["value"])
}

func TestCorpusQueryORGroupPatternExactMatch(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction | genre=novel) <> textClass=belletristik")

	// Exact OR structure matches
	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "genre", "value": "fiction"},
				map[string]any{"@type": "koral:doc", "key": "genre", "value": "novel"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:doc", corpus["@type"])
	assert.Equal(t, "textClass", corpus["key"])
	assert.Equal(t, "belletristik", corpus["value"])
}

func TestCorpusQueryORGroupPatternSingleOperandMatch(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction | genre=novel) <> textClass=belletristik")

	// Single doc matches OR group pattern when any operand matches.
	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "genre",
			"value": "fiction",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "textClass", corpus["key"])
	assert.Equal(t, "belletristik", corpus["value"])
}

func TestCorpusQueryORGroupPatternNoMatchWrongValue(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction | genre=novel) <> textClass=belletristik")

	// Single doc with value not in OR pattern should not match.
	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "genre",
			"value": "science",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "science", corpus["value"])
}

func TestCorpusQueryComplexityOrdering(t *testing.T) {
	// Rules ordered by complexity (most specific first).
	// Group patterns use structural matching: AND matches AND groups,
	// OR matches OR groups. The forward rule's OR B-side does NOT match
	// individual AND groups in BtoA, so reverse rules handle those.
	m := newCorpusMapper(t,
		// Forward: Entertainment → OR-of-ANDs (complex B-side, for AtoB)
		"genre=Entertainment <> ((textClass=kultur & textClass=musik) | (textClass=kultur & textClass=film))",
		// Reverse aggregated: (Entertainment | Culture) → AND (for BtoA with (k&f))
		"(genre=Entertainment | genre=Culture) <> (textClass=kultur & textClass=film)",
		// Reverse individual: Entertainment → AND (for BtoA with (k&m))
		"genre=Entertainment <> (textClass=kultur & textClass=musik)",
	)

	// AtoB: first rule matches (simple field pattern on A-side)
	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "genre",
			"value": "Entertainment",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:or", corpus["operation"])
	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)

	// BtoA with AND group (kultur & film): forward rule's OR B-side doesn't
	// match AND structurally, so the reverse aggregated rule matches
	input2 := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "film"},
			},
		},
	}
	result2, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input2)
	require.NoError(t, err)

	corpus2 := result2.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus2["@type"])
	assert.Equal(t, "operation:or", corpus2["operation"])
	ops := corpus2["operands"].([]any)
	require.Len(t, ops, 2)
	assert.Equal(t, "Entertainment", ops[0].(map[string]any)["value"])
	assert.Equal(t, "Culture", ops[1].(map[string]any)["value"])

	// BtoA with AND group (kultur & musik): reverse individual rule matches
	input3 := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
			},
		},
	}
	result3, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input3)
	require.NoError(t, err)

	corpus3 := result3.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:doc", corpus3["@type"])
	assert.Equal(t, "genre", corpus3["key"])
	assert.Equal(t, "Entertainment", corpus3["value"])
}

func TestCorpusQueryGroupToFieldReplacementRewrite(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA, AddRewrites: true}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])

	rewrites, ok := corpus["rewrites"].([]any)
	require.True(t, ok)
	require.Len(t, rewrites, 1)

	rewrite := rewrites[0].(map[string]any)
	assert.Equal(t, "koral:rewrite", rewrite["@type"])
	// Original was a group, so the whole structure is stored
	original, ok := rewrite["original"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "koral:docGroup", original["@type"])
}

func TestCorpusQueryNestedGroupPatternMatch(t *testing.T) {
	// Nested: OR of AND groups
	m := newCorpusMapper(t, "genre=fiction <> ((textClass=kultur & textClass=musik) | (textClass=kultur & textClass=film))")

	// BtoA: the OR-of-AND pattern matches an exact OR-of-AND docGroup
	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				map[string]any{
					"@type":     "koral:docGroup",
					"operation": "operation:and",
					"operands": []any{
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
					},
				},
				map[string]any{
					"@type":     "koral:docGroup",
					"operation": "operation:and",
					"operands": []any{
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "film"},
					},
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:doc", corpus["@type"])
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryGroupPatternRecursionFallthrough(t *testing.T) {
	// Group pattern doesn't match the outer group, so we recurse into operands
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				// This inner AND group matches the rule's B-side pattern
				map[string]any{
					"@type":     "koral:docGroup",
					"operation": "operation:and",
					"operands": []any{
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
					},
				},
				// This stays unchanged
				map[string]any{
					"@type": "koral:doc",
					"key":   "author",
					"value": "Fontane",
				},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:or", corpus["operation"])

	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)

	// First operand was replaced
	first := operands[0].(map[string]any)
	assert.Equal(t, "genre", first["key"])
	assert.Equal(t, "fiction", first["value"])

	// Second operand unchanged
	second := operands[1].(map[string]any)
	assert.Equal(t, "author", second["key"])
	assert.Equal(t, "Fontane", second["value"])
}

func TestCorpusQueryFieldGroupAliasWithGroupPattern(t *testing.T) {
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:fieldGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:field", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:field", "key": "textClass", "value": "musik"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "genre", corpus["key"])
	assert.Equal(t, "fiction", corpus["value"])
}

func TestCorpusQueryComplexPatternAndComplexReplacementAtoB(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction & region=de) <> ((textClass=kultur & textClass=film) | textClass=kultur.film)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "genre", "value": "fiction"},
				map[string]any{"@type": "koral:doc", "key": "region", "value": "de"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:or", corpus["operation"])

	orOps := corpus["operands"].([]any)
	require.Len(t, orOps, 2)

	andGroup := orOps[0].(map[string]any)
	assert.Equal(t, "koral:docGroup", andGroup["@type"])
	assert.Equal(t, "operation:and", andGroup["operation"])
	andOps := andGroup["operands"].([]any)
	require.Len(t, andOps, 2)
	assert.Equal(t, "textClass", andOps[0].(map[string]any)["key"])
	assert.Equal(t, "kultur", andOps[0].(map[string]any)["value"])
	assert.Equal(t, "textClass", andOps[1].(map[string]any)["key"])
	assert.Equal(t, "film", andOps[1].(map[string]any)["value"])

	dotValue := orOps[1].(map[string]any)
	assert.Equal(t, "koral:doc", dotValue["@type"])
	assert.Equal(t, "textClass", dotValue["key"])
	assert.Equal(t, "kultur.film", dotValue["value"])
}

func TestCorpusQueryComplexPatternAndComplexReplacementBtoA(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction & region=de) <> ((textClass=kultur & textClass=film) | textClass=kultur.film)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:or",
			"operands": []any{
				map[string]any{
					"@type":     "koral:docGroup",
					"operation": "operation:and",
					"operands": []any{
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
						map[string]any{"@type": "koral:doc", "key": "textClass", "value": "film"},
					},
				},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur.film"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:and", corpus["operation"])
	andOps := corpus["operands"].([]any)
	require.Len(t, andOps, 2)

	left := andOps[0].(map[string]any)
	right := andOps[1].(map[string]any)
	keys := []string{left["key"].(string), right["key"].(string)}
	values := []string{left["value"].(string), right["value"].(string)}
	assert.ElementsMatch(t, []string{"genre", "region"}, keys)
	assert.ElementsMatch(t, []string{"fiction", "de"}, values)
}

// --- Iterative rule application tests ---

func TestCorpusQueryIterativeRuleApplication(t *testing.T) {
	// Two rules applied to the same tree — both should fire on different operands.
	m := newCorpusMapper(t,
		"textClass=novel <> genre=fiction",
		"textClass=science <> genre=nonfiction",
	)

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "novel"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "science"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)
	assert.Equal(t, "fiction", operands[0].(map[string]any)["value"])
	assert.Equal(t, "nonfiction", operands[1].(map[string]any)["value"])
}

func TestCorpusQueryIterativeSuccessiveTransform(t *testing.T) {
	// Rule 1 transforms a field, rule 2 transforms the result of rule 1.
	m := newCorpusMapper(t,
		"textClass=novel <> genre=fiction",
		"genre=fiction <> category=lit",
	)

	input := map[string]any{
		"corpus": map[string]any{
			"@type": "koral:doc",
			"key":   "textClass",
			"value": "novel",
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "category", corpus["key"])
	assert.Equal(t, "lit", corpus["value"])
}

// --- AND subset matching tests ---

func TestCorpusQueryANDSubsetMatchGroupReplacement(t *testing.T) {
	// AND pattern with group replacement on subset match
	m := newCorpusMapper(t, "genre=fiction <> (textClass=kultur & textClass=musik)")

	input := map[string]any{
		"corpus": map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
			"operands": []any{
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "kultur"},
				map[string]any{"@type": "koral:doc", "key": "textClass", "value": "musik"},
				map[string]any{"@type": "koral:doc", "key": "author", "value": "Goethe"},
			},
		},
	}
	result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: BtoA}, input)
	require.NoError(t, err)

	corpus := result.(map[string]any)["corpus"].(map[string]any)
	assert.Equal(t, "koral:docGroup", corpus["@type"])
	assert.Equal(t, "operation:and", corpus["operation"])

	operands := corpus["operands"].([]any)
	require.Len(t, operands, 2)

	assert.Equal(t, "genre", operands[0].(map[string]any)["key"])
	assert.Equal(t, "fiction", operands[0].(map[string]any)["value"])
	assert.Equal(t, "author", operands[1].(map[string]any)["key"])
	assert.Equal(t, "Goethe", operands[1].(map[string]any)["value"])
}

// --- OR any-operand matching tests ---

func TestCorpusQueryORPatternMatchesBothOperands(t *testing.T) {
	m := newCorpusMapper(t, "(genre=fiction | genre=novel) <> textClass=belletristik")

	// Both "fiction" and "novel" should match the OR pattern
	for _, val := range []string{"fiction", "novel"} {
		input := map[string]any{
			"corpus": map[string]any{
				"@type": "koral:doc",
				"key":   "genre",
				"value": val,
			},
		}
		result, err := m.ApplyQueryMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
		require.NoError(t, err)

		corpus := result.(map[string]any)["corpus"].(map[string]any)
		assert.Equal(t, "textClass", corpus["key"], "value %s should match", val)
		assert.Equal(t, "belletristik", corpus["value"])
	}
}

// --- Response-side OR pattern and replacement tests ---

func TestCorpusResponseORPatternMatchesSingleField(t *testing.T) {
	m := newCorpusMapper(t, "(textClass=novel | textClass=fiction) <> (genre=lit & type=book)")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "novel",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 3)

	mapped1 := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped1["key"])
	assert.Equal(t, "lit", mapped1["value"])

	mapped2 := fields[2].(map[string]any)
	assert.Equal(t, "type", mapped2["key"])
	assert.Equal(t, "book", mapped2["value"])
}

func TestCorpusResponseORReplacementSkipped(t *testing.T) {
	m := newCorpusMapper(t, "textClass=novel <> (genre=fiction | genre=novel)")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "novel",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 1, "OR replacement should be skipped in response")
}

func TestCorpusResponseORPatternANDReplacementBothFields(t *testing.T) {
	// Rule: (a | b) <> (c & d)
	// When "a" is in response, both "c" and "d" should be added.
	m := newCorpusMapper(t, "(textClass=a | textClass=b) <> (genre=c & genre=d)")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "a",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 3)

	mapped1 := fields[1].(map[string]any)
	assert.Equal(t, "genre", mapped1["key"])
	assert.Equal(t, "c", mapped1["value"])
	assert.Equal(t, true, mapped1["mapped"])

	mapped2 := fields[2].(map[string]any)
	assert.Equal(t, "genre", mapped2["key"])
	assert.Equal(t, "d", mapped2["value"])
	assert.Equal(t, true, mapped2["mapped"])
}

func TestCorpusResponseORPatternORReplacementSkipped(t *testing.T) {
	// Rule: (a | b) <> (c | d)
	// When "a" is in response, nothing should be added (OR replacement skipped).
	m := newCorpusMapper(t, "(textClass=a | textClass=b) <> (genre=c | genre=d)")

	input := map[string]any{
		"fields": []any{
			map[string]any{
				"@type": "koral:field",
				"key":   "textClass",
				"value": "a",
				"type":  "type:string",
			},
		},
	}
	result, err := m.ApplyResponseMappings("corpus-test", MappingOptions{Direction: AtoB}, input)
	require.NoError(t, err)

	fields := result.(map[string]any)["fields"].([]any)
	require.Len(t, fields, 1, "OR replacement should be skipped")
}
