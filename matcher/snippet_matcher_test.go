package matcher

import (
	"testing"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnippetMatcher_ParseSnippet(t *testing.T) {
	// Create a pattern for testing
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "marmot",
			Layer:   "m",
			Key:     "gender",
			Value:   "masc",
			Match:   ast.MatchEqual,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Layer:   "m",
			Key:     "M",
			Value:   "",
			Match:   ast.MatchEqual,
		},
	}

	sm, err := NewSnippetMatcher(pattern, replacement)
	require.NoError(t, err)

	tests := []struct {
		name             string
		snippet          string
		expectedTokens   int
		expectedContains []string
	}{
		{
			name: "Simple single token",
			snippet: `<span title="corenlp/p:ART">
				<span title="marmot/m:case:nom">
				<span title="marmot/m:gender:masc">
				<span title="marmot/m:number:sg">
				<span title="marmot/p:ART">
				Der</span>
				</span>
				</span>
				</span>
				</span>`,
			expectedTokens:   1,
			expectedContains: []string{"Der"},
		},
		{
			name: "Multiple tokens",
			snippet: `<span title="corenlp/p:ART">
				<span title="marmot/m:case:nom">
				<span title="marmot/m:gender:masc">
				Der</span>
				</span>
				</span> 
				<span title="corenlp/p:ADJA">
				<span title="marmot/m:case:nom">
				<span title="marmot/m:gender:masc">
				alte</span>
				</span>
				</span>`,
			expectedTokens:   2,
			expectedContains: []string{"Der", "alte"},
		},
		{
			name: "Real-world example from test",
			snippet: `<span title="corenlp/p:ART">
				<span title="marmot/m:case:nom">
				<span title="marmot/m:gender:masc">
				<span title="marmot/m:number:sg">
				<span title="marmot/p:ART">
				<span title="opennlp/p:ART">
				<span title="tt/l:die">
				<span title="tt/p:ART">Der</span>
				</span>
				</span>
				</span>
				</span>
				</span>
				</span>
				</span>`,
			expectedTokens:   1,
			expectedContains: []string{"Der"},
		},
		{
			name:             "Empty snippet",
			snippet:          "",
			expectedTokens:   0,
			expectedContains: []string{},
		},
		{
			name:             "No span elements",
			snippet:          "Just some text",
			expectedTokens:   0,
			expectedContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := sm.ParseSnippet(tt.snippet)
			require.NoError(t, err)

			assert.Len(t, tokens, tt.expectedTokens)

			for i, expectedText := range tt.expectedContains {
				if i < len(tokens) {
					assert.Equal(t, expectedText, tokens[i].Text)
				}
			}
		})
	}
}

func TestSnippetMatcher_CheckToken(t *testing.T) {
	// Create a pattern that matches tokens with marmot/m:gender=masc
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "marmot",
			Layer:   "m",
			Key:     "gender",
			Value:   "masc",
			Match:   ast.MatchEqual,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Layer:   "m",
			Key:     "M",
			Value:   "",
			Match:   ast.MatchEqual,
		},
	}

	sm, err := NewSnippetMatcher(pattern, replacement)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       TokenSpan
		shouldMatch bool
	}{
		{
			name: "Token with matching annotation",
			token: TokenSpan{
				Text: "Der",
				Annotations: []string{
					"corenlp/p:ART",
					"marmot/m:case:nom",
					"marmot/m:gender:masc",
					"marmot/m:number:sg",
				},
			},
			shouldMatch: true,
		},
		{
			name: "Token without matching annotation",
			token: TokenSpan{
				Text: "und",
				Annotations: []string{
					"corenlp/p:KON",
					"marmot/p:KON",
					"opennlp/p:KON",
				},
			},
			shouldMatch: false,
		},
		{
			name: "Token with no annotations",
			token: TokenSpan{
				Text:        "text",
				Annotations: []string{},
			},
			shouldMatch: false,
		},
		{
			name: "Token with different gender value",
			token: TokenSpan{
				Text: "andere",
				Annotations: []string{
					"marmot/m:gender:fem",
					"marmot/m:case:nom",
				},
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := sm.CheckToken(tt.token)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestSnippetMatcher_FindMatchingTokens(t *testing.T) {
	// Create a pattern that matches tokens with marmot/m:gender=masc
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "marmot",
			Layer:   "m",
			Key:     "gender",
			Value:   "masc",
			Match:   ast.MatchEqual,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Layer:   "m",
			Key:     "M",
			Value:   "",
			Match:   ast.MatchEqual,
		},
	}

	sm, err := NewSnippetMatcher(pattern, replacement)
	require.NoError(t, err)

	// Test snippet with mixed tokens - some matching, some not
	snippet := `<span title="corenlp/p:ART">
		<span title="marmot/m:case:nom">
		<span title="marmot/m:gender:masc">
		<span title="marmot/m:number:sg">
		Der</span>
		</span>
		</span>
		</span> 
		<span title="corenlp/p:ADJA">
		<span title="marmot/m:case:nom">
		<span title="marmot/m:gender:masc">
		alte</span>
		</span>
		</span> 
		<span title="corenlp/p:NN">
		<span title="marmot/m:case:nom">
		<span title="marmot/m:gender:masc">
		Baum</span>
		</span>
		</span> 
		<span title="corenlp/p:KON">
		<span title="marmot/p:KON">
		und</span>
		</span>`

	matchingTokens, err := sm.FindMatchingTokens(snippet)
	require.NoError(t, err)

	// Should find 3 matching tokens: "Der", "alte", "Baum" (all with gender:masc)
	// but not "und" (no gender annotation)
	assert.Len(t, matchingTokens, 3)

	expectedTexts := []string{"Der", "alte", "Baum"}
	for i, token := range matchingTokens {
		assert.Equal(t, expectedTexts[i], token.Text)

		// Verify that each token has the required annotation
		hasGenderMasc := false
		for _, annotation := range token.Annotations {
			if annotation == "marmot/m:gender:masc" {
				hasGenderMasc = true
				break
			}
		}
		assert.True(t, hasGenderMasc, "Token %s should have marmot/m:gender:masc annotation", token.Text)
	}
}

func TestSnippetMatcher_RealWorldExample(t *testing.T) {
	// Test with the real-world example from the response test
	pattern := ast.Pattern{
		Root: &ast.Term{
			Foundry: "marmot",
			Layer:   "m",
			Key:     "gender",
			Value:   "masc",
			Match:   ast.MatchEqual,
		},
	}

	replacement := ast.Replacement{
		Root: &ast.Term{
			Foundry: "opennlp",
			Layer:   "m",
			Key:     "M",
			Value:   "",
			Match:   ast.MatchEqual,
		},
	}

	sm, err := NewSnippetMatcher(pattern, replacement)
	require.NoError(t, err)

	// Real snippet from the test file
	snippet := `<span title="corenlp/p:ART">` +
		`<span title="marmot/m:case:nom">` +
		`<span title="marmot/m:gender:masc">` +
		`<span title="marmot/m:number:sg">` +
		`<span title="marmot/p:ART">` +
		`<span title="opennlp/p:ART">` +
		`<span title="tt/l:die">` +
		`<span title="tt/p:ART">Der</span>` +
		`</span>` +
		`</span>` +
		`</span>` +
		`</span>` +
		`</span>` +
		`</span>` +
		`</span>`

	// Parse the snippet
	tokens, err := sm.ParseSnippet(snippet)
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "Der", token.Text)

	// Check that it has all expected annotations
	expectedAnnotations := []string{
		"corenlp/p:ART",
		"marmot/m:case:nom",
		"marmot/m:gender:masc",
		"marmot/m:number:sg",
		"marmot/p:ART",
		"opennlp/p:ART",
		"tt/l:die",
		"tt/p:ART",
	}

	assert.Len(t, token.Annotations, len(expectedAnnotations))
	for _, expected := range expectedAnnotations {
		assert.Contains(t, token.Annotations, expected)
	}

	// Check that it matches our pattern
	matches, err := sm.CheckToken(token)
	require.NoError(t, err)
	assert.True(t, matches)
}
