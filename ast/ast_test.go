package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected NodeType
	}{
		{
			name:     "Token node returns correct type",
			node:     &Token{Wrap: &Term{}},
			expected: TokenNode,
		},
		{
			name: "TermGroup node returns correct type",
			node: &TermGroup{
				Operands: []Node{&Term{}},
				Relation: AndRelation,
			},
			expected: TermGroupNode,
		},
		{
			name: "Term node returns correct type",
			node: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
			},
			expected: TermNode,
		},
		{
			name: "Rewrite node returns correct type",
			node: &Rewrite{
				Editor:    "Kustvakt",
				Operation: "operation:injection",
				Scope:     "foundry",
				Src:       "Kustvakt",
			},
			expected: RewriteNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.Type())
		})
	}
}

func TestTermGroupConstruction(t *testing.T) {
	term1 := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}

	term2 := &Term{
		Foundry: "opennlp",
		Key:     "AdjType",
		Layer:   "m",
		Match:   MatchEqual,
		Value:   "Pdt",
	}

	rewrites := []Rewrite{
		{
			Editor:    "Kustvakt",
			Operation: "operation:injection",
			Scope:     "foundry",
			Src:       "Kustvakt",
			Comment:   "Default foundry has been added.",
		},
	}

	group := &TermGroup{
		Operands: []Node{term1, term2},
		Relation: AndRelation,
		Rewrites: rewrites,
	}

	assert.Len(t, group.Operands, 2)
	assert.Equal(t, AndRelation, group.Relation)
	assert.Equal(t, TermGroupNode, group.Type())
	assert.Equal(t, rewrites, group.Rewrites)

	// Test operands are correctly set
	assert.Equal(t, term1, group.Operands[0])
	assert.Equal(t, term2, group.Operands[1])
}

func TestTokenConstruction(t *testing.T) {
	term := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}

	rewrites := []Rewrite{
		{
			Editor:    "Kustvakt",
			Operation: "operation:injection",
			Scope:     "foundry",
			Src:       "Kustvakt",
			Comment:   "Default foundry has been added.",
		},
	}

	token := &Token{
		Wrap:     term,
		Rewrites: rewrites,
	}

	assert.Equal(t, TokenNode, token.Type())
	assert.Equal(t, term, token.Wrap)
	assert.Equal(t, rewrites, token.Rewrites)
}

func TestTermConstruction(t *testing.T) {
	tests := []struct {
		name     string
		term     *Term
		foundry  string
		key      string
		layer    string
		match    MatchType
		hasValue bool
		value    string
		rewrites []Rewrite
	}{
		{
			name: "Term without value",
			term: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
			},
			foundry:  "opennlp",
			key:      "DET",
			layer:    "p",
			match:    MatchEqual,
			hasValue: false,
		},
		{
			name: "Term with value",
			term: &Term{
				Foundry: "opennlp",
				Key:     "AdjType",
				Layer:   "m",
				Match:   MatchEqual,
				Value:   "Pdt",
			},
			foundry:  "opennlp",
			key:      "AdjType",
			layer:    "m",
			match:    MatchEqual,
			hasValue: true,
			value:    "Pdt",
		},
		{
			name: "Term with not equal match",
			term: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchNotEqual,
			},
			foundry:  "opennlp",
			key:      "DET",
			layer:    "p",
			match:    MatchNotEqual,
			hasValue: false,
		},
		{
			name: "Term with rewrites",
			term: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
				Rewrites: []Rewrite{
					{
						Editor:    "Kustvakt",
						Operation: "operation:injection",
						Scope:     "foundry",
						Src:       "Kustvakt",
						Comment:   "Default foundry has been added.",
					},
				},
			},
			foundry:  "opennlp",
			key:      "DET",
			layer:    "p",
			match:    MatchEqual,
			hasValue: false,
			rewrites: []Rewrite{
				{
					Editor:    "Kustvakt",
					Operation: "operation:injection",
					Scope:     "foundry",
					Src:       "Kustvakt",
					Comment:   "Default foundry has been added.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, TermNode, tt.term.Type())
			assert.Equal(t, tt.foundry, tt.term.Foundry)
			assert.Equal(t, tt.key, tt.term.Key)
			assert.Equal(t, tt.layer, tt.term.Layer)
			assert.Equal(t, tt.match, tt.term.Match)
			if tt.hasValue {
				assert.Equal(t, tt.value, tt.term.Value)
			} else {
				assert.Empty(t, tt.term.Value)
			}
			if tt.rewrites != nil {
				assert.Equal(t, tt.rewrites, tt.term.Rewrites)
			} else {
				assert.Empty(t, tt.term.Rewrites)
			}
		})
	}
}

func TestPatternAndReplacement(t *testing.T) {
	// Create a simple pattern
	patternTerm := &Term{
		Foundry: "opennlp",
		Key:     "DET",
		Layer:   "p",
		Match:   MatchEqual,
	}
	pattern := Pattern{Root: patternTerm}

	// Create a simple replacement
	replacementTerm := &Term{
		Foundry: "opennlp",
		Key:     "COMBINED_DET",
		Layer:   "p",
		Match:   MatchEqual,
	}
	replacement := Replacement{Root: replacementTerm}

	// Test pattern
	assert.NotNil(t, pattern.Root)
	assert.Equal(t, patternTerm, pattern.Root)

	// Test replacement
	assert.NotNil(t, replacement.Root)
	assert.Equal(t, replacementTerm, replacement.Root)
}

func TestCatchallNode(t *testing.T) {
	tests := []struct {
		name       string
		nodeType   string
		content    string
		wrap       Node
		operands   []Node
		expectType NodeType
	}{
		{
			name:       "CatchallNode with custom type",
			nodeType:   "customType",
			content:    `{"key": "value"}`,
			expectType: NodeType("customType"),
		},
		{
			name:     "CatchallNode with wrapped term",
			nodeType: "wrapper",
			content:  `{"key": "value"}`,
			wrap: &Term{
				Foundry: "test",
				Key:     "TEST",
				Layer:   "x",
				Match:   MatchEqual,
			},
			expectType: NodeType("wrapper"),
		},
		{
			name:     "CatchallNode with operands",
			nodeType: "custom_group",
			content:  `{"key": "value"}`,
			operands: []Node{
				&Term{Foundry: "test1", Key: "TEST1", Layer: "x", Match: MatchEqual},
				&Term{Foundry: "test2", Key: "TEST2", Layer: "y", Match: MatchEqual},
			},
			expectType: NodeType("custom_group"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawContent := json.RawMessage(tt.content)
			node := &CatchallNode{
				NodeType:   tt.nodeType,
				RawContent: rawContent,
				Wrap:       tt.wrap,
				Operands:   tt.operands,
			}

			assert.Equal(t, tt.expectType, node.Type())
			if tt.wrap != nil {
				assert.Equal(t, tt.wrap, node.Wrap)
			}
			if tt.operands != nil {
				assert.Equal(t, tt.operands, node.Operands)
			}
			assert.Equal(t, rawContent, node.RawContent)
		})
	}
}

func TestRewriteConstruction(t *testing.T) {
	rewrite := &Rewrite{
		Editor:    "Kustvakt",
		Operation: "operation:injection",
		Scope:     "foundry",
		Src:       "Kustvakt",
		Comment:   "Default foundry has been added.",
	}

	assert.Equal(t, RewriteNode, rewrite.Type())
	assert.Equal(t, "Kustvakt", rewrite.Editor)
	assert.Equal(t, "operation:injection", rewrite.Operation)
	assert.Equal(t, "foundry", rewrite.Scope)
	assert.Equal(t, "Kustvakt", rewrite.Src)
	assert.Equal(t, "Default foundry has been added.", rewrite.Comment)
}

func TestComplexNestedStructures(t *testing.T) {
	// Test nested tokens and term groups
	termGroup := &TermGroup{
		Operands: []Node{
			&Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
			},
			&Term{
				Foundry: "opennlp",
				Key:     "AdjType",
				Layer:   "m",
				Match:   MatchEqual,
				Value:   "Pdt",
			},
		},
		Relation: AndRelation,
	}

	token := &Token{
		Wrap: termGroup,
	}

	assert.Equal(t, TokenNode, token.Type())
	assert.NotNil(t, token.Wrap)
	assert.Equal(t, TermGroupNode, token.Wrap.Type())

	// Test that the nested structure is correct
	if tg, ok := token.Wrap.(*TermGroup); ok {
		assert.Equal(t, 2, len(tg.Operands))
		assert.Equal(t, AndRelation, tg.Relation)
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Empty TermGroup",
			test: func(t *testing.T) {
				group := &TermGroup{
					Operands: []Node{},
					Relation: AndRelation,
				}
				assert.Empty(t, group.Operands)
				assert.Equal(t, AndRelation, group.Relation)
			},
		},
		{
			name: "Token with nil wrap",
			test: func(t *testing.T) {
				token := &Token{Wrap: nil}
				assert.Equal(t, TokenNode, token.Type())
				assert.Nil(t, token.Wrap)
			},
		},
		{
			name: "Term with empty strings",
			test: func(t *testing.T) {
				term := &Term{
					Foundry: "",
					Key:     "",
					Layer:   "",
					Match:   MatchEqual,
					Value:   "",
				}
				assert.Equal(t, TermNode, term.Type())
				assert.Empty(t, term.Foundry)
				assert.Empty(t, term.Key)
				assert.Empty(t, term.Layer)
				assert.Empty(t, term.Value)
			},
		},
		{
			name: "Complex Pattern and Replacement",
			test: func(t *testing.T) {
				// Create a complex pattern
				patternGroup := &TermGroup{
					Operands: []Node{
						&Term{Foundry: "f1", Key: "k1", Layer: "l1", Match: MatchEqual},
						&Token{Wrap: &Term{Foundry: "f2", Key: "k2", Layer: "l2", Match: MatchNotEqual}},
					},
					Relation: OrRelation,
				}
				pattern := Pattern{Root: patternGroup}

				// Create a complex replacement
				replacementGroup := &TermGroup{
					Operands: []Node{
						&Term{Foundry: "f3", Key: "k3", Layer: "l3", Match: MatchEqual},
						&Term{Foundry: "f4", Key: "k4", Layer: "l4", Match: MatchEqual},
					},
					Relation: AndRelation,
				}
				replacement := Replacement{Root: replacementGroup}

				assert.Equal(t, TermGroupNode, pattern.Root.Type())
				assert.Equal(t, TermGroupNode, replacement.Root.Type())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestCloneMethod(t *testing.T) {
	tests := []struct {
		name string
		node Node
	}{
		{
			name: "Clone Term",
			node: &Term{
				Foundry: "opennlp",
				Key:     "DET",
				Layer:   "p",
				Match:   MatchEqual,
				Value:   "test",
				Rewrites: []Rewrite{
					{
						Editor: "test",
						Scope:  "foundry",
					},
				},
			},
		},
		{
			name: "Clone Token",
			node: &Token{
				Wrap: &Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   MatchEqual,
				},
				Rewrites: []Rewrite{
					{
						Editor: "test",
						Scope:  "layer",
					},
				},
			},
		},
		{
			name: "Clone TermGroup",
			node: &TermGroup{
				Operands: []Node{
					&Term{
						Foundry: "opennlp",
						Key:     "DET",
						Layer:   "p",
						Match:   MatchEqual,
					},
					&Term{
						Foundry: "opennlp",
						Key:     "AdjType",
						Layer:   "m",
						Match:   MatchEqual,
						Value:   "Pdt",
					},
				},
				Relation: AndRelation,
				Rewrites: []Rewrite{
					{
						Editor: "test",
						Scope:  "foundry",
					},
				},
			},
		},
		{
			name: "Clone CatchallNode",
			node: &CatchallNode{
				NodeType:   "koral:unknown",
				RawContent: []byte(`{"@type":"koral:unknown","test":"value"}`),
				Wrap: &Term{
					Foundry: "opennlp",
					Key:     "DET",
					Layer:   "p",
					Match:   MatchEqual,
				},
				Operands: []Node{
					&Term{
						Foundry: "opennlp",
						Key:     "AdjType",
						Layer:   "m",
						Match:   MatchEqual,
						Value:   "Pdt",
					},
				},
			},
		},
		{
			name: "Clone Rewrite",
			node: &Rewrite{
				Editor:    "termMapper",
				Operation: "injection",
				Scope:     "foundry",
				Src:       "test",
				Comment:   "test comment",
				Original:  "original_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.node.Clone()

			// Check that the clone is not the same instance
			assert.NotSame(t, tt.node, cloned)

			// Check that the clone has the same type
			assert.Equal(t, tt.node.Type(), cloned.Type())

			// Check that nodes are equal (deep comparison)
			assert.True(t, NodesEqual(tt.node, cloned))

			// Test that modifying the clone doesn't affect the original
			switch original := tt.node.(type) {
			case *Term:
				clonedTerm := cloned.(*Term)
				clonedTerm.Foundry = "modified"
				assert.NotEqual(t, original.Foundry, clonedTerm.Foundry)

			case *Token:
				clonedToken := cloned.(*Token)
				if clonedToken.Wrap != nil {
					if termWrap, ok := clonedToken.Wrap.(*Term); ok {
						termWrap.Foundry = "modified"
						if originalWrap, ok := original.Wrap.(*Term); ok {
							assert.NotEqual(t, originalWrap.Foundry, termWrap.Foundry)
						}
					}
				}

			case *TermGroup:
				clonedGroup := cloned.(*TermGroup)
				clonedGroup.Relation = OrRelation
				assert.NotEqual(t, original.Relation, clonedGroup.Relation)

			case *CatchallNode:
				clonedCatchall := cloned.(*CatchallNode)
				clonedCatchall.NodeType = "modified"
				assert.NotEqual(t, original.NodeType, clonedCatchall.NodeType)

			case *Rewrite:
				clonedRewrite := cloned.(*Rewrite)
				clonedRewrite.Editor = "modified"
				assert.NotEqual(t, original.Editor, clonedRewrite.Editor)
			}
		})
	}
}

func TestCloneNilNodes(t *testing.T) {
	// Test cloning nodes with nil fields
	tests := []struct {
		name string
		node Node
	}{
		{
			name: "Token with nil wrap",
			node: &Token{Wrap: nil},
		},
		{
			name: "TermGroup with empty operands",
			node: &TermGroup{
				Operands: []Node{},
				Relation: AndRelation,
			},
		},
		{
			name: "CatchallNode with nil wrap and operands",
			node: &CatchallNode{
				NodeType:   "koral:unknown",
				RawContent: nil,
				Wrap:       nil,
				Operands:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.node.Clone()
			assert.NotSame(t, tt.node, cloned)
			assert.Equal(t, tt.node.Type(), cloned.Type())
			assert.True(t, NodesEqual(tt.node, cloned))
		})
	}
}

func TestApplyFoundryAndLayerOverrides(t *testing.T) {
	tests := []struct {
		name            string
		node            Node
		foundry         string
		layer           string
		expectedChanges func(t *testing.T, node Node)
	}{
		{
			name: "Apply foundry and layer to Term",
			node: &Term{
				Foundry: "original",
				Key:     "DET",
				Layer:   "original",
				Match:   MatchEqual,
			},
			foundry: "new_foundry",
			layer:   "new_layer",
			expectedChanges: func(t *testing.T, node Node) {
				term := node.(*Term)
				assert.Equal(t, "new_foundry", term.Foundry)
				assert.Equal(t, "new_layer", term.Layer)
			},
		},
		{
			name: "Apply only foundry to Term",
			node: &Term{
				Foundry: "original",
				Key:     "DET",
				Layer:   "original",
				Match:   MatchEqual,
			},
			foundry: "new_foundry",
			layer:   "",
			expectedChanges: func(t *testing.T, node Node) {
				term := node.(*Term)
				assert.Equal(t, "new_foundry", term.Foundry)
				assert.Equal(t, "original", term.Layer) // Should remain unchanged
			},
		},
		{
			name: "Apply to TermGroup",
			node: &TermGroup{
				Operands: []Node{
					&Term{
						Foundry: "original1",
						Key:     "DET",
						Layer:   "original1",
						Match:   MatchEqual,
					},
					&Term{
						Foundry: "original2",
						Key:     "AdjType",
						Layer:   "original2",
						Match:   MatchEqual,
						Value:   "Pdt",
					},
				},
				Relation: AndRelation,
			},
			foundry: "new_foundry",
			layer:   "new_layer",
			expectedChanges: func(t *testing.T, node Node) {
				termGroup := node.(*TermGroup)
				for _, operand := range termGroup.Operands {
					if term, ok := operand.(*Term); ok {
						assert.Equal(t, "new_foundry", term.Foundry)
						assert.Equal(t, "new_layer", term.Layer)
					}
				}
			},
		},
		{
			name: "Apply to Token with wrapped Term",
			node: &Token{
				Wrap: &Term{
					Foundry: "original",
					Key:     "DET",
					Layer:   "original",
					Match:   MatchEqual,
				},
			},
			foundry: "new_foundry",
			layer:   "new_layer",
			expectedChanges: func(t *testing.T, node Node) {
				token := node.(*Token)
				if term, ok := token.Wrap.(*Term); ok {
					assert.Equal(t, "new_foundry", term.Foundry)
					assert.Equal(t, "new_layer", term.Layer)
				}
			},
		},
		{
			name: "Apply to CatchallNode",
			node: &CatchallNode{
				NodeType: "koral:unknown",
				Wrap: &Term{
					Foundry: "original",
					Key:     "DET",
					Layer:   "original",
					Match:   MatchEqual,
				},
				Operands: []Node{
					&Term{
						Foundry: "original2",
						Key:     "AdjType",
						Layer:   "original2",
						Match:   MatchEqual,
						Value:   "Pdt",
					},
				},
			},
			foundry: "new_foundry",
			layer:   "new_layer",
			expectedChanges: func(t *testing.T, node Node) {
				catchall := node.(*CatchallNode)
				if term, ok := catchall.Wrap.(*Term); ok {
					assert.Equal(t, "new_foundry", term.Foundry)
					assert.Equal(t, "new_layer", term.Layer)
				}
				for _, operand := range catchall.Operands {
					if term, ok := operand.(*Term); ok {
						assert.Equal(t, "new_foundry", term.Foundry)
						assert.Equal(t, "new_layer", term.Layer)
					}
				}
			},
		},
		{
			name: "Apply to nested structure",
			node: &Token{
				Wrap: &TermGroup{
					Operands: []Node{
						&Term{
							Foundry: "original1",
							Key:     "DET",
							Layer:   "original1",
							Match:   MatchEqual,
						},
						&Token{
							Wrap: &Term{
								Foundry: "original2",
								Key:     "AdjType",
								Layer:   "original2",
								Match:   MatchEqual,
								Value:   "Pdt",
							},
						},
					},
					Relation: AndRelation,
				},
			},
			foundry: "new_foundry",
			layer:   "new_layer",
			expectedChanges: func(t *testing.T, node Node) {
				token := node.(*Token)
				if termGroup, ok := token.Wrap.(*TermGroup); ok {
					for _, operand := range termGroup.Operands {
						switch op := operand.(type) {
						case *Term:
							assert.Equal(t, "new_foundry", op.Foundry)
							assert.Equal(t, "new_layer", op.Layer)
						case *Token:
							if innerTerm, ok := op.Wrap.(*Term); ok {
								assert.Equal(t, "new_foundry", innerTerm.Foundry)
								assert.Equal(t, "new_layer", innerTerm.Layer)
							}
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clone the node to avoid modifying the original test data
			cloned := tt.node.Clone()

			// Apply the overrides
			ApplyFoundryAndLayerOverrides(cloned, tt.foundry, tt.layer)

			// Check the expected changes
			tt.expectedChanges(t, cloned)
		})
	}
}

func TestApplyFoundryAndLayerOverridesNilNode(t *testing.T) {
	// Test that applying overrides to a nil node doesn't panic
	assert.NotPanics(t, func() {
		ApplyFoundryAndLayerOverrides(nil, "foundry", "layer")
	})
}

func TestApplyFoundryAndLayerOverridesEmptyValues(t *testing.T) {
	// Test applying empty foundry and layer values
	term := &Term{
		Foundry: "original_foundry",
		Key:     "DET",
		Layer:   "original_layer",
		Match:   MatchEqual,
	}

	ApplyFoundryAndLayerOverrides(term, "", "")

	// Values should remain unchanged
	assert.Equal(t, "original_foundry", term.Foundry)
	assert.Equal(t, "original_layer", term.Layer)
}
