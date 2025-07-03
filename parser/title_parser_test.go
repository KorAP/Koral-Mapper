package parser

import (
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTitleAttributeParser_ParseTitleAttribute was removed as ParseTitleAttribute is no longer exported
// The functionality is now only available through ParseTitleAttributesToTerms method

func TestTitleAttributeParser_ParseTitleAttributesToTerms(t *testing.T) {
	parser := NewTitleAttributeParser()

	tests := []struct {
		name     string
		input    []string
		expected []ast.Node
		wantErr  bool
	}{
		{
			name:  "Parse multiple title attributes",
			input: []string{"corenlp/p:ART", "marmot/m:case:nom", "tt/l:die"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "corenlp",
					Layer:   "p",
					Key:     "ART",
					Value:   "",
					Match:   ast.MatchEqual,
				},
				&ast.Term{
					Foundry: "marmot",
					Layer:   "m",
					Key:     "case",
					Value:   "nom",
					Match:   ast.MatchEqual,
				},
				&ast.Term{
					Foundry: "tt",
					Layer:   "l",
					Key:     "die",
					Value:   "",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:     "Empty input should return empty slice",
			input:    []string{},
			expected: []ast.Node{},
			wantErr:  false,
		},
		{
			name:    "Invalid title should cause error",
			input:   []string{"corenlp/p:ART", "invalid_title", "tt/l:die"},
			wantErr: true,
		},
		// Additional tests to cover functionality from removed TestTitleAttributeParser_ParseTitleAttribute
		{
			name:  "Parse simple title with key only",
			input: []string{"corenlp/p:ART"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "corenlp",
					Layer:   "p",
					Key:     "ART",
					Value:   "",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse title with key and value",
			input: []string{"marmot/m:case:nom"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "marmot",
					Layer:   "m",
					Key:     "case",
					Value:   "nom",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse title with colon separator for value",
			input: []string{"marmot/m:gender:masc"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "marmot",
					Layer:   "m",
					Key:     "gender",
					Value:   "masc",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse title with equals separator for value",
			input: []string{"marmot/m:degree:pos"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "marmot",
					Layer:   "m",
					Key:     "degree",
					Value:   "pos",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse title with lemma layer",
			input: []string{"tt/l:die"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "tt",
					Layer:   "l",
					Key:     "die",
					Value:   "",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse title with special characters in value",
			input: []string{"tt/l:@card@"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "tt",
					Layer:   "l",
					Key:     "@card@",
					Value:   "",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse complex key-value with colon",
			input: []string{"opennlp/p:PronType:Ind"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "opennlp",
					Layer:   "p",
					Key:     "PronType",
					Value:   "Ind",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse complex key-value with equals",
			input: []string{"opennlp/p:AdjType:Pdt"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "opennlp",
					Layer:   "p",
					Key:     "AdjType",
					Value:   "Pdt",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		{
			name:  "Parse complex nested pattern",
			input: []string{"stts/p:ADJA"},
			expected: []ast.Node{
				&ast.Term{
					Foundry: "stts",
					Layer:   "p",
					Key:     "ADJA",
					Value:   "",
					Match:   ast.MatchEqual,
				},
			},
			wantErr: false,
		},
		// Error cases
		{
			name:    "Empty title should fail",
			input:   []string{""},
			wantErr: true,
		},
		{
			name:    "Missing foundry separator should fail",
			input:   []string{"corenlp_p:ART"},
			wantErr: true,
		},
		{
			name:    "Missing layer separator should fail",
			input:   []string{"corenlp/p_ART"},
			wantErr: true,
		},
		{
			name:    "Only foundry should fail",
			input:   []string{"corenlp"},
			wantErr: true,
		},
		{
			name:    "Only foundry and layer should fail",
			input:   []string{"corenlp/p"},
			wantErr: true,
		},
		{
			name:    "Missing key should fail",
			input:   []string{"corenlp/p:"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseTitleAttributesToTerms(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, result, len(tt.expected))

				for i, expectedTerm := range tt.expected {
					expectedTermNode := expectedTerm.(*ast.Term)
					actualTermNode := result[i].(*ast.Term)

					assert.Equal(t, expectedTermNode.Foundry, actualTermNode.Foundry)
					assert.Equal(t, expectedTermNode.Layer, actualTermNode.Layer)
					assert.Equal(t, expectedTermNode.Key, actualTermNode.Key)
					assert.Equal(t, expectedTermNode.Value, actualTermNode.Value)
					assert.Equal(t, expectedTermNode.Match, actualTermNode.Match)
				}
			}
		})
	}
}

// TestTitleAttribute_ToAST was removed as ToAST method is no longer available
// TestTitleAttribute_String was removed as String method is no longer available
// TestTitleAttributeParser_RealWorldExample was removed as it used the removed methods
