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
