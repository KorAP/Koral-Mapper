package main

import (
	"testing"

	tmconfig "github.com/KorAP/Koral-Mapper/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cfgTestLists = []tmconfig.MappingList{
	{
		ID:       "stts-upos",
		FoundryA: "opennlp",
		LayerA:   "p",
		FoundryB: "upos",
		LayerB:   "p",
		Mappings: []tmconfig.MappingRule{"[PIDAT] <> [DET]"},
	},
	{
		ID:       "other-mapper",
		FoundryA: "stts",
		LayerA:   "p",
		FoundryB: "ud",
		LayerB:   "pos",
		Mappings: []tmconfig.MappingRule{"[A] <> [B]"},
	},
	{
		ID:       "corpus-map",
		Type:     "corpus",
		Mappings: []tmconfig.MappingRule{"textClass=science <> textClass=akademisch"},
	},
}

func TestParseCfgParam(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []CascadeEntry
		wantErr  string
	}{
		{
			name: "Full 6-field entry",
			raw:  "stts-upos:atob:opennlp:p:upos:p",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
			},
		},
		{
			name: "Short 2-field entry defaults to YAML values",
			raw:  "stts-upos:atob",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
			},
		},
		{
			name: "Short 2-field entry with btoa direction",
			raw:  "other-mapper:btoa",
			expected: []CascadeEntry{
				{ID: "other-mapper", Direction: "btoa", FoundryA: "stts", LayerA: "p", FoundryB: "ud", LayerB: "pos"},
			},
		},
		{
			name: "Mixed 2-field and 6-field entries",
			raw:  "stts-upos:atob;other-mapper:btoa:stts:p:ud:p",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
				{ID: "other-mapper", Direction: "btoa", FoundryA: "stts", LayerA: "p", FoundryB: "ud", LayerB: "p"},
			},
		},
		{
			name:     "Empty cfg string returns empty slice",
			raw:      "",
			expected: nil,
		},
		{
			name:    "Unknown mapping ID returns error",
			raw:     "unknown-id:atob",
			wantErr: "unknown mapping ID",
		},
		{
			name:    "Second entry has unknown mapping ID",
			raw:     "stts-upos:atob;unknown:btoa",
			wantErr: "unknown mapping ID",
		},
		{
			name:    "Malformed entry with 1 field",
			raw:     "stts-upos",
			wantErr: "invalid entry",
		},
		{
			name:    "Malformed entry with 3 fields",
			raw:     "stts-upos:atob:extra",
			wantErr: "invalid entry",
		},
		{
			name:    "Malformed entry with 4 fields",
			raw:     "stts-upos:atob:a:b",
			wantErr: "invalid entry",
		},
		{
			name:    "Malformed entry with 5 fields",
			raw:     "stts-upos:atob:a:b:c",
			wantErr: "invalid entry",
		},
		{
			name: "Empty override fields fall back to YAML defaults",
			raw:  "stts-upos:atob::::",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
			},
		},
		{
			name: "Partial overrides merge with YAML defaults",
			raw:  "stts-upos:atob:custom::custom:",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "custom", LayerA: "p", FoundryB: "custom", LayerB: "p"},
			},
		},
		{
			name: "Corpus mapping 2-field entry has no foundry/layer defaults",
			raw:  "corpus-map:atob",
			expected: []CascadeEntry{
				{ID: "corpus-map", Direction: "atob"},
			},
		},
		{
			name:    "Invalid direction",
			raw:     "stts-upos:invalid",
			wantErr: "invalid direction",
		},
		{
			name: "Three entries with mixed types",
			raw:  "stts-upos:atob;corpus-map:atob;other-mapper:btoa",
			expected: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
				{ID: "corpus-map", Direction: "atob"},
				{ID: "other-mapper", Direction: "btoa", FoundryA: "stts", LayerA: "p", FoundryB: "ud", LayerB: "pos"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCfgParam(tt.raw, cfgTestLists)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCfgParam(t *testing.T) {
	tests := []struct {
		name     string
		entries  []CascadeEntry
		expected string
	}{
		{
			name: "Full 6-field entry",
			entries: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
			},
			expected: "stts-upos:atob:opennlp:p:upos:p",
		},
		{
			name: "Short 2-field entry when all foundry/layer empty",
			entries: []CascadeEntry{
				{ID: "corpus-map", Direction: "atob"},
			},
			expected: "corpus-map:atob",
		},
		{
			name: "Multiple entries",
			entries: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
				{ID: "other-mapper", Direction: "btoa", FoundryA: "stts", LayerA: "p", FoundryB: "ud", LayerB: "p"},
			},
			expected: "stts-upos:atob:opennlp:p:upos:p;other-mapper:btoa:stts:p:ud:p",
		},
		{
			name:     "Nil slice returns empty string",
			entries:  nil,
			expected: "",
		},
		{
			name:     "Empty slice returns empty string",
			entries:  []CascadeEntry{},
			expected: "",
		},
		{
			name: "Mixed full and short entries",
			entries: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp", LayerA: "p", FoundryB: "upos", LayerB: "p"},
				{ID: "corpus-map", Direction: "atob"},
			},
			expected: "stts-upos:atob:opennlp:p:upos:p;corpus-map:atob",
		},
		{
			name: "Entry with some empty foundry/layer fields uses 6-field format",
			entries: []CascadeEntry{
				{ID: "stts-upos", Direction: "atob", FoundryA: "opennlp"},
			},
			expected: "stts-upos:atob:opennlp:::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCfgParam(tt.entries)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAndParseCfgParamRoundTrip(t *testing.T) {
	original := "stts-upos:atob:opennlp:p:upos:p;corpus-map:btoa"
	entries, err := ParseCfgParam(original, cfgTestLists)
	require.NoError(t, err)

	rebuilt := BuildCfgParam(entries)
	assert.Equal(t, original, rebuilt)
}
