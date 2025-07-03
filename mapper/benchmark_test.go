package mapper

import (
	"testing"

	"github.com/KorAP/Koral-Mapper/config"
)

// BenchmarkApplyQueryMappings benchmarks the optimized ApplyQueryMappings method
func BenchmarkApplyQueryMappings(b *testing.B) {
	// Setup test data
	mappingList := config.MappingList{
		ID:       "test-mapper",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "tt",
		LayerB:   "p",
		Mappings: []config.MappingRule{
			"[PIAT] <> [PDAT]",
			"[PPER & opennlp/m=PronType:Prs] <> [PPER]",
			"[PRELS] <> [PRELAT]",
			"[PIDAT] <> [PDAT]",
			"[DET & opennlp/m=AdjType:Pdt] <> [ART]",
			"[DET & opennlp/m=PronType:Ind] <> [PIS]",
		},
	}

	mapper, err := NewMapper([]config.MappingList{mappingList})
	if err != nil {
		b.Fatalf("Failed to create mapper: %v", err)
	}

	// Test cases that represent different scenarios
	testCases := []struct {
		name string
		data map[string]any
		opts MappingOptions
	}{
		{
			name: "Simple_term_match",
			data: map[string]any{
				"@type":   "koral:term",
				"foundry": "opennlp",
				"key":     "PIAT",
				"layer":   "p",
				"match":   "match:eq",
			},
			opts: MappingOptions{Direction: true},
		},
		{
			name: "Complex_term_group_with_match",
			data: map[string]any{
				"@type": "koral:termGroup",
				"operands": []any{
					map[string]any{
						"@type":   "koral:term",
						"foundry": "opennlp",
						"key":     "DET",
						"layer":   "p",
						"match":   "match:eq",
					},
					map[string]any{
						"@type":   "koral:term",
						"foundry": "opennlp",
						"key":     "AdjType",
						"layer":   "m",
						"match":   "match:eq",
						"value":   "Pdt",
					},
				},
				"relation": "relation:and",
			},
			opts: MappingOptions{Direction: true},
		},
		{
			name: "No_match_scenario",
			data: map[string]any{
				"@type":   "koral:term",
				"foundry": "opennlp",
				"key":     "NOMATCH",
				"layer":   "p",
				"match":   "match:eq",
			},
			opts: MappingOptions{Direction: true},
		},
		{
			name: "With_foundry_override",
			data: map[string]any{
				"@type":   "koral:term",
				"foundry": "opennlp",
				"key":     "PIAT",
				"layer":   "p",
				"match":   "match:eq",
			},
			opts: MappingOptions{
				Direction: true,
				FoundryB:  "custom",
				LayerB:    "pos",
			},
		},
		{
			name: "Nested_token_structure",
			data: map[string]any{
				"@type": "koral:token",
				"wrap": map[string]any{
					"@type": "koral:termGroup",
					"operands": []any{
						map[string]any{
							"@type":   "koral:term",
							"foundry": "opennlp",
							"key":     "PPER",
							"layer":   "p",
							"match":   "match:eq",
						},
						map[string]any{
							"@type":   "koral:term",
							"foundry": "opennlp",
							"key":     "PronType",
							"layer":   "m",
							"match":   "match:eq",
							"value":   "Prs",
						},
					},
					"relation": "relation:and",
				},
			},
			opts: MappingOptions{Direction: true, AddRewrites: true},
		},
	}

	// Run benchmarks for each test case
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := mapper.ApplyQueryMappings("test-mapper", tc.opts, tc.data)
				if err != nil {
					b.Fatalf("ApplyQueryMappings failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkApplyQueryMappingsWorstCase benchmarks the worst case scenario
// where we have many rules but no matches (tests optimization effectiveness)
func BenchmarkApplyQueryMappingsWorstCase(b *testing.B) {
	// Create a mapper with many rules
	manyRules := make([]config.MappingRule, 100)
	for i := 0; i < 100; i++ {
		ruleChar := string(rune('A' + i%26))
		manyRules[i] = config.MappingRule("[UNUSED" + ruleChar + "] <> [TARGET" + ruleChar + "]")
	}

	mappingList := config.MappingList{
		ID:       "many-rules",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "tt",
		LayerB:   "p",
		Mappings: manyRules,
	}

	mapper, err := NewMapper([]config.MappingList{mappingList})
	if err != nil {
		b.Fatalf("Failed to create mapper: %v", err)
	}

	// Test data that won't match any rule
	testData := map[string]any{
		"@type":   "koral:term",
		"foundry": "opennlp",
		"key":     "NOMATCH",
		"layer":   "p",
		"match":   "match:eq",
	}

	opts := MappingOptions{Direction: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mapper.ApplyQueryMappings("many-rules", opts, testData)
		if err != nil {
			b.Fatalf("ApplyQueryMappings failed: %v", err)
		}
	}
}
