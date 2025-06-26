package parser

import (
	"testing"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTitleAttributeParser_ParseTitleAttribute(t *testing.T) {
	parser := NewTitleAttributeParser()

	tests := []struct {
		name     string
		input    string
		expected *TitleAttribute
		wantErr  bool
	}{
		{
			name:  "Parse simple title with key only",
			input: "corenlp/p:ART",
			expected: &TitleAttribute{
				Foundry: "corenlp",
				Layer:   "p",
				Key:     "ART",
				Value:   "",
			},
			wantErr: false,
		},
		{
			name:  "Parse title with key and value",
			input: "marmot/m:case:nom",
			expected: &TitleAttribute{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "case",
				Value:   "nom",
			},
			wantErr: false,
		},
		{
			name:  "Parse title with colon separator for value",
			input: "marmot/m:gender:masc",
			expected: &TitleAttribute{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "gender",
				Value:   "masc",
			},
			wantErr: false,
		},
		{
			name:  "Parse title with equals separator for value",
			input: "marmot/m:degree:pos",
			expected: &TitleAttribute{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "degree",
				Value:   "pos",
			},
			wantErr: false,
		},
		{
			name:  "Parse title with lemma layer",
			input: "tt/l:die",
			expected: &TitleAttribute{
				Foundry: "tt",
				Layer:   "l",
				Key:     "die",
				Value:   "",
			},
			wantErr: false,
		},
		{
			name:  "Parse title with special characters in value",
			input: "tt/l:@card@",
			expected: &TitleAttribute{
				Foundry: "tt",
				Layer:   "l",
				Key:     "@card@",
				Value:   "",
			},
			wantErr: false,
		},
		{
			name:    "Empty title should fail",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Missing foundry separator should fail",
			input:   "corenlp_p:ART",
			wantErr: true,
		},
		{
			name:    "Missing layer separator should fail",
			input:   "corenlp/p_ART",
			wantErr: true,
		},
		{
			name:    "Only foundry should fail",
			input:   "corenlp",
			wantErr: true,
		},
		{
			name:    "Only foundry and layer should fail",
			input:   "corenlp/p",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseTitleAttribute(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Foundry, result.Foundry)
				assert.Equal(t, tt.expected.Layer, result.Layer)
				assert.Equal(t, tt.expected.Key, result.Key)
				assert.Equal(t, tt.expected.Value, result.Value)
			}
		})
	}
}

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

func TestTitleAttribute_ToAST(t *testing.T) {
	tests := []struct {
		name     string
		attr     *TitleAttribute
		expected *ast.Term
	}{
		{
			name: "Convert title attribute to AST term",
			attr: &TitleAttribute{
				Foundry: "corenlp",
				Layer:   "p",
				Key:     "ART",
				Value:   "",
			},
			expected: &ast.Term{
				Foundry: "corenlp",
				Layer:   "p",
				Key:     "ART",
				Value:   "",
				Match:   ast.MatchEqual,
			},
		},
		{
			name: "Convert title attribute with value to AST term",
			attr: &TitleAttribute{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "case",
				Value:   "nom",
			},
			expected: &ast.Term{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "case",
				Value:   "nom",
				Match:   ast.MatchEqual,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.attr.ToAST()

			termResult := result.(*ast.Term)
			assert.Equal(t, tt.expected.Foundry, termResult.Foundry)
			assert.Equal(t, tt.expected.Layer, termResult.Layer)
			assert.Equal(t, tt.expected.Key, termResult.Key)
			assert.Equal(t, tt.expected.Value, termResult.Value)
			assert.Equal(t, tt.expected.Match, termResult.Match)
		})
	}
}

func TestTitleAttribute_String(t *testing.T) {
	tests := []struct {
		name     string
		attr     *TitleAttribute
		expected string
	}{
		{
			name: "String representation without value",
			attr: &TitleAttribute{
				Foundry: "corenlp",
				Layer:   "p",
				Key:     "ART",
				Value:   "",
			},
			expected: "corenlp/p:ART",
		},
		{
			name: "String representation with value",
			attr: &TitleAttribute{
				Foundry: "marmot",
				Layer:   "m",
				Key:     "case",
				Value:   "nom",
			},
			expected: "marmot/m:case=nom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.attr.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTitleAttributeParser_RealWorldExample(t *testing.T) {
	parser := NewTitleAttributeParser()

	// Example titles from the response test file
	titles := []string{
		"corenlp/p:ART",
		"marmot/m:case=nom",
		"marmot/m:gender=masc",
		"marmot/m:number=sg",
		"marmot/p:ART",
		"opennlp/p:ART",
		"tt/l:die",
		"tt/p:ART",
	}

	// Parse each title attribute
	for _, title := range titles {
		attr, err := parser.ParseTitleAttribute(title)
		require.NoError(t, err)
		require.NotNil(t, attr)

		// Verify the string representation matches
		assert.Equal(t, title, attr.String())

		// Verify conversion to AST works
		astNode := attr.ToAST()
		require.NotNil(t, astNode)

		term := astNode.(*ast.Term)
		assert.NotEmpty(t, term.Foundry)
		assert.NotEmpty(t, term.Layer)
		assert.NotEmpty(t, term.Key)
		assert.Equal(t, ast.MatchEqual, term.Match)
	}
}
