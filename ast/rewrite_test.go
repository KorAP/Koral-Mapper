package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Rewrite
	}{
		{
			name: "Standard rewrite with editor and original",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"original": {
					"@type": "koral:term",
					"foundry": "opennlp",
					"key": "PIDAT",
					"layer": "p",
					"match": "match:eq"
				}
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Original: map[string]any{
					"@type":   "koral:term",
					"foundry": "opennlp",
					"key":     "PIDAT",
					"layer":   "p",
					"match":   "match:eq",
				},
			},
		},
		{
			name: "Legacy rewrite with source instead of editor",
			input: `{
				"@type": "koral:rewrite",
				"source": "legacy-mapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"src": "legacy-source"
			}`,
			expected: Rewrite{
				Editor:    "legacy-mapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "legacy-source",
			},
		},
		{
			name: "Legacy rewrite with origin instead of original/src",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"origin": "legacy-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "legacy-origin",
			},
		},
		{
			name: "Precedence test: editor over source",
			input: `{
				"@type": "koral:rewrite",
				"editor": "preferred-editor",
				"source": "legacy-source",
				"operation": "operation:mapping"
			}`,
			expected: Rewrite{
				Editor:    "preferred-editor",
				Operation: "operation:mapping",
			},
		},
		{
			name: "Precedence test: original over src over origin",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"original": "preferred-original",
				"src": "middle-src",
				"origin": "lowest-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Original:  "preferred-original",
			},
		},
		{
			name: "Precedence test: src over origin when no original",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"src": "preferred-src",
				"origin": "lowest-origin"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Src:       "preferred-src",
			},
		},
		{
			name: "Only legacy fields",
			input: `{
				"@type": "koral:rewrite",
				"source": "legacy-editor",
				"operation": "operation:mapping",
				"origin": "legacy-origin",
				"_comment": "Legacy rewrite"
			}`,
			expected: Rewrite{
				Editor:    "legacy-editor",
				Operation: "operation:mapping",
				Src:       "legacy-origin",
				Comment:   "Legacy rewrite",
			},
		},
		{
			name: "Mixed with comment",
			input: `{
				"@type": "koral:rewrite",
				"editor": "termMapper",
				"operation": "operation:mapping",
				"scope": "foundry",
				"src": "original-source",
				"_comment": "This is a comment"
			}`,
			expected: Rewrite{
				Editor:    "termMapper",
				Operation: "operation:mapping",
				Scope:     "foundry",
				Src:       "original-source",
				Comment:   "This is a comment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rewrite Rewrite
			err := json.Unmarshal([]byte(tt.input), &rewrite)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, rewrite)
		})
	}
}

func TestRewriteArrayUnmarshal(t *testing.T) {
	// Test unmarshaling an array of rewrites with mixed legacy and modern fields
	input := `[
		{
			"@type": "koral:rewrite",
			"editor": "termMapper",
			"operation": "operation:mapping",
			"original": "modern-original"
		},
		{
			"@type": "koral:rewrite",
			"source": "legacy-editor",
			"operation": "operation:legacy",
			"origin": "legacy-origin"
		}
	]`

	var rewrites []Rewrite
	err := json.Unmarshal([]byte(input), &rewrites)
	require.NoError(t, err)
	require.Len(t, rewrites, 2)

	// Check first rewrite (modern)
	assert.Equal(t, "termMapper", rewrites[0].Editor)
	assert.Equal(t, "operation:mapping", rewrites[0].Operation)
	assert.Equal(t, "modern-original", rewrites[0].Original)

	// Check second rewrite (legacy)
	assert.Equal(t, "legacy-editor", rewrites[1].Editor)
	assert.Equal(t, "operation:legacy", rewrites[1].Operation)
	assert.Equal(t, "legacy-origin", rewrites[1].Src)
}

func TestRewriteableInterface(t *testing.T) {
	t.Run("Term implements Rewriteable", func(t *testing.T) {
		term := &Term{Foundry: "opennlp", Key: "DET", Layer: "p", Match: MatchEqual}

		var r Rewriteable = term
		assert.Nil(t, r.GetRewrites())

		rewrites := []Rewrite{{Editor: "test", Scope: "foundry"}}
		r.SetRewrites(rewrites)
		assert.Equal(t, rewrites, r.GetRewrites())
		assert.Equal(t, rewrites, term.Rewrites)
	})

	t.Run("TermGroup implements Rewriteable", func(t *testing.T) {
		tg := &TermGroup{
			Operands: []Node{&Term{Key: "A", Match: MatchEqual}},
			Relation: AndRelation,
		}

		var r Rewriteable = tg
		assert.Nil(t, r.GetRewrites())

		rewrites := []Rewrite{{Editor: "editor", Scope: "layer", Original: "old"}}
		r.SetRewrites(rewrites)
		assert.Equal(t, rewrites, r.GetRewrites())
		assert.Equal(t, rewrites, tg.Rewrites)
	})

	t.Run("Token implements Rewriteable", func(t *testing.T) {
		tok := &Token{Wrap: &Term{Key: "X", Match: MatchEqual}}

		var r Rewriteable = tok
		assert.Nil(t, r.GetRewrites())

		rewrites := []Rewrite{{Editor: "mapper", Operation: "op"}}
		r.SetRewrites(rewrites)
		assert.Equal(t, rewrites, r.GetRewrites())
		assert.Equal(t, rewrites, tok.Rewrites)
	})

	t.Run("SetRewrites to nil clears slice", func(t *testing.T) {
		term := &Term{
			Key:      "DET",
			Match:    MatchEqual,
			Rewrites: []Rewrite{{Editor: "x"}},
		}
		term.SetRewrites(nil)
		assert.Nil(t, term.GetRewrites())
	})
}

func TestAppendRewrite(t *testing.T) {
	t.Run("Append to Term", func(t *testing.T) {
		term := &Term{Key: "DET", Match: MatchEqual}
		rw := Rewrite{Editor: "Koral-Mapper", Scope: "foundry", Original: "opennlp"}

		AppendRewrite(term, rw)
		assert.Equal(t, []Rewrite{rw}, term.Rewrites)

		rw2 := Rewrite{Editor: "Koral-Mapper", Scope: "key", Original: "PIDAT"}
		AppendRewrite(term, rw2)
		assert.Equal(t, []Rewrite{rw, rw2}, term.Rewrites)
	})

	t.Run("Append to TermGroup", func(t *testing.T) {
		tg := &TermGroup{
			Operands: []Node{&Term{Key: "A", Match: MatchEqual}},
			Relation: AndRelation,
		}
		rw := Rewrite{Editor: "editor", Original: "orig"}
		AppendRewrite(tg, rw)
		assert.Equal(t, []Rewrite{rw}, tg.Rewrites)
	})

	t.Run("Append to Token", func(t *testing.T) {
		tok := &Token{Wrap: &Term{Key: "X", Match: MatchEqual}}
		rw := Rewrite{Editor: "ed"}
		AppendRewrite(tok, rw)
		assert.Equal(t, []Rewrite{rw}, tok.Rewrites)
	})

	t.Run("Append to non-Rewriteable is no-op", func(t *testing.T) {
		catchall := &CatchallNode{NodeType: "koral:span"}
		rw := Rewrite{Editor: "test"}
		AppendRewrite(catchall, rw)
		// CatchallNode doesn't implement Rewriteable, so nothing happens
	})

	t.Run("Append to nil is no-op", func(t *testing.T) {
		assert.NotPanics(t, func() {
			AppendRewrite(nil, Rewrite{Editor: "x"})
		})
	})
}

func TestStripRewrites(t *testing.T) {
	t.Run("Strips from Term", func(t *testing.T) {
		term := &Term{
			Key:      "DET",
			Match:    MatchEqual,
			Rewrites: []Rewrite{{Editor: "a"}, {Editor: "b"}},
		}
		StripRewrites(term)
		assert.Nil(t, term.Rewrites)
	})

	t.Run("Strips from Token and its Wrap", func(t *testing.T) {
		tok := &Token{
			Wrap: &Term{
				Key:      "DET",
				Match:    MatchEqual,
				Rewrites: []Rewrite{{Editor: "inner"}},
			},
			Rewrites: []Rewrite{{Editor: "outer"}},
		}
		StripRewrites(tok)
		assert.Nil(t, tok.Rewrites)
		assert.Nil(t, tok.Wrap.(*Term).Rewrites)
	})

	t.Run("Strips from TermGroup and all operands", func(t *testing.T) {
		tg := &TermGroup{
			Operands: []Node{
				&Term{Key: "A", Match: MatchEqual, Rewrites: []Rewrite{{Editor: "e1"}}},
				&Term{Key: "B", Match: MatchEqual, Rewrites: []Rewrite{{Editor: "e2"}}},
			},
			Relation: AndRelation,
			Rewrites: []Rewrite{{Editor: "group"}},
		}
		StripRewrites(tg)
		assert.Nil(t, tg.Rewrites)
		assert.Nil(t, tg.Operands[0].(*Term).Rewrites)
		assert.Nil(t, tg.Operands[1].(*Term).Rewrites)
	})

	t.Run("Strips recursively from CatchallNode", func(t *testing.T) {
		catchall := &CatchallNode{
			NodeType: "koral:group",
			Wrap: &Term{
				Key:      "W",
				Match:    MatchEqual,
				Rewrites: []Rewrite{{Editor: "wrap-ed"}},
			},
			Operands: []Node{
				&Token{
					Wrap:     &Term{Key: "X", Match: MatchEqual, Rewrites: []Rewrite{{Editor: "deep"}}},
					Rewrites: []Rewrite{{Editor: "tok"}},
				},
			},
		}
		StripRewrites(catchall)
		assert.Nil(t, catchall.Wrap.(*Term).Rewrites)
		tok := catchall.Operands[0].(*Token)
		assert.Nil(t, tok.Rewrites)
		assert.Nil(t, tok.Wrap.(*Term).Rewrites)
	})

	t.Run("Nil node does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			StripRewrites(nil)
		})
	})

	t.Run("Already empty rewrites stays nil", func(t *testing.T) {
		term := &Term{Key: "DET", Match: MatchEqual}
		StripRewrites(term)
		assert.Nil(t, term.Rewrites)
	})
}

func TestRewriteMarshalJSON(t *testing.T) {
	// Test that marshaling works correctly and maintains the modern field names
	rewrite := Rewrite{
		Editor:    "termMapper",
		Operation: "operation:mapping",
		Scope:     "foundry",
		Src:       "source-value",
		Comment:   "Test comment",
		Original:  "original-value",
	}

	data, err := json.Marshal(rewrite)
	require.NoError(t, err)

	// Parse back to verify structure
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "koral:rewrite", result["@type"])
	assert.Equal(t, "termMapper", result["editor"])
	assert.Equal(t, "operation:mapping", result["operation"])
	assert.Equal(t, "foundry", result["scope"])
	assert.Equal(t, "source-value", result["src"])
	assert.Equal(t, "Test comment", result["_comment"])
	assert.Equal(t, "original-value", result["original"])

	// Ensure legacy fields are not present in output
	assert.NotContains(t, result, "source")
	assert.NotContains(t, result, "origin")
}

func TestRewriteMarshalJSONValueAndPointerConsistent(t *testing.T) {
	rw := Rewrite{
		Editor:   "Koral-Mapper",
		Scope:    "key",
		Original: "textClass",
	}

	valueBytes, err := json.Marshal(rw)
	require.NoError(t, err)

	pointerBytes, err := json.Marshal(&rw)
	require.NoError(t, err)

	assert.JSONEq(t, string(pointerBytes), string(valueBytes))
}

func TestRewriteToMap(t *testing.T) {
	t.Run("All fields set", func(t *testing.T) {
		rw := Rewrite{
			Editor:    "termMapper",
			Operation: "operation:mapping",
			Scope:     "foundry",
			Src:       "source-value",
			Comment:   "Test comment",
			Original:  "original-value",
		}

		m := rw.ToMap()
		assert.Equal(t, "koral:rewrite", m["@type"])
		assert.Equal(t, "termMapper", m["editor"])
		assert.Equal(t, "operation:mapping", m["operation"])
		assert.Equal(t, "foundry", m["scope"])
		assert.Equal(t, "source-value", m["src"])
		assert.Equal(t, "Test comment", m["_comment"])
		assert.Equal(t, "original-value", m["original"])
	})

	t.Run("Only editor and scope", func(t *testing.T) {
		rw := Rewrite{
			Editor: "Koral-Mapper",
			Scope:  "key",
		}

		m := rw.ToMap()
		assert.Equal(t, "koral:rewrite", m["@type"])
		assert.Equal(t, "Koral-Mapper", m["editor"])
		assert.Equal(t, "key", m["scope"])
		assert.NotContains(t, m, "operation")
		assert.NotContains(t, m, "src")
		assert.NotContains(t, m, "_comment")
		assert.NotContains(t, m, "original")
	})

	t.Run("With complex original", func(t *testing.T) {
		original := map[string]any{
			"@type":     "koral:docGroup",
			"operation": "operation:and",
		}
		rw := Rewrite{
			Editor:   "Koral-Mapper",
			Original: original,
		}

		m := rw.ToMap()
		assert.Equal(t, "koral:rewrite", m["@type"])
		assert.Equal(t, "Koral-Mapper", m["editor"])
		assert.Equal(t, original, m["original"])
	})

	t.Run("Matches MarshalJSON output", func(t *testing.T) {
		rw := Rewrite{
			Editor:   "Koral-Mapper",
			Scope:    "key",
			Original: "textClass",
		}

		toMapResult := rw.ToMap()

		data, err := json.Marshal(&rw)
		require.NoError(t, err)
		var fromJSON map[string]any
		require.NoError(t, json.Unmarshal(data, &fromJSON))

		assert.Equal(t, fromJSON, toMapResult)
	})
}
