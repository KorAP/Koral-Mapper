package ast

// ast is the abstract syntax tree for the query term mapper.

import (
	"encoding/json"
)

// NodeType represents the type of a node in the AST
type NodeType string

// RelationType represents the type of relation between nodes
type RelationType string

// MatchType represents the type of match operation
type MatchType string

const (
	TokenNode     NodeType     = "token"
	TermGroupNode NodeType     = "termGroup"
	TermNode      NodeType     = "term"
	RewriteNode   NodeType     = "rewrite"
	AndRelation   RelationType = "and"
	OrRelation    RelationType = "or"
	MatchEqual    MatchType    = "eq"
	MatchNotEqual MatchType    = "ne"
)

// Node represents a node in the AST
type Node interface {
	Type() NodeType
	Clone() Node
}

// Rewrite represents a koral:rewrite
type Rewrite struct {
	Editor    string `json:"editor,omitempty"`
	Operation string `json:"operation,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Src       string `json:"src,omitempty"`
	Comment   string `json:"_comment,omitempty"`
	Original  any    `json:"original,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for backward compatibility
func (r *Rewrite) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to hold all possible fields
	var temp struct {
		Type      string `json:"@type,omitempty"`
		Editor    string `json:"editor,omitempty"`
		Source    string `json:"source,omitempty"`    // legacy field
		Operation string `json:"operation,omitempty"` // legacy field
		Scope     string `json:"scope,omitempty"`
		Src       string `json:"src,omitempty"`
		Origin    string `json:"origin,omitempty"` // legacy field
		Original  any    `json:"original,omitempty"`
		Comment   string `json:"_comment,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Apply precedence for editor field: editor >> source
	if temp.Editor != "" {
		r.Editor = temp.Editor
	} else if temp.Source != "" {
		r.Editor = temp.Source
	}

	// Apply precedence for original/src/origin: original >> src >> origin
	if temp.Original != nil {
		r.Original = temp.Original
	} else if temp.Src != "" {
		r.Src = temp.Src
	} else if temp.Origin != "" {
		r.Src = temp.Origin
	}

	// Copy other fields
	r.Operation = temp.Operation
	r.Scope = temp.Scope
	r.Comment = temp.Comment

	return nil
}

func (r *Rewrite) Type() NodeType {
	return RewriteNode
}

// Clone creates a deep copy of the Rewrite node
func (r *Rewrite) Clone() Node {
	return &Rewrite{
		Editor:    r.Editor,
		Operation: r.Operation,
		Scope:     r.Scope,
		Src:       r.Src,
		Comment:   r.Comment,
		Original:  r.Original, // Note: this is a shallow copy of the Original field
	}
}

// MarshalJSON implements custom JSON marshaling to ensure clean output
func (r *Rewrite) MarshalJSON() ([]byte, error) {
	// Create a map with only the modern field names
	result := make(map[string]any)

	// Always include @type if this is a rewrite
	result["@type"] = "koral:rewrite"

	if r.Editor != "" {
		result["editor"] = r.Editor
	}
	if r.Operation != "" {
		result["operation"] = r.Operation
	}
	if r.Scope != "" {
		result["scope"] = r.Scope
	}
	if r.Src != "" {
		result["src"] = r.Src
	}
	if r.Comment != "" {
		result["_comment"] = r.Comment
	}
	if r.Original != nil {
		result["original"] = r.Original
	}

	return json.Marshal(result)
}

// Token represents a koral:token
type Token struct {
	Wrap     Node      `json:"wrap"`
	Rewrites []Rewrite `json:"rewrites,omitempty"`
}

func (t *Token) Type() NodeType {
	return TokenNode
}

// Clone creates a deep copy of the Token node
func (t *Token) Clone() Node {
	var clonedWrap Node
	if t.Wrap != nil {
		clonedWrap = t.Wrap.Clone()
	}
	tc := &Token{
		Wrap: clonedWrap,
	}

	if t.Rewrites != nil {
		clonedRewrites := make([]Rewrite, len(t.Rewrites))
		for i, rewrite := range t.Rewrites {
			clonedRewrites[i] = *rewrite.Clone().(*Rewrite)
		}
		tc.Rewrites = clonedRewrites
	}

	return tc
}

// TermGroup represents a koral:termGroup
type TermGroup struct {
	Operands []Node       `json:"operands"`
	Relation RelationType `json:"relation"`
	Rewrites []Rewrite    `json:"rewrites,omitempty"`
}

func (tg *TermGroup) Type() NodeType {
	return TermGroupNode
}

// Clone creates a deep copy of the TermGroup node
func (tg *TermGroup) Clone() Node {
	clonedOperands := make([]Node, len(tg.Operands))
	for i, operand := range tg.Operands {
		clonedOperands[i] = operand.Clone()
	}
	tgc := &TermGroup{
		Operands: clonedOperands,
		Relation: tg.Relation,
	}
	if tg.Rewrites != nil {
		clonedRewrites := make([]Rewrite, len(tg.Rewrites))
		for i, rewrite := range tg.Rewrites {
			clonedRewrites[i] = *rewrite.Clone().(*Rewrite)
		}
		tgc.Rewrites = clonedRewrites
	}

	return tgc
}

// Term represents a koral:term
type Term struct {
	Foundry  string    `json:"foundry"`
	Key      string    `json:"key"`
	Layer    string    `json:"layer"`
	Match    MatchType `json:"match"`
	Value    string    `json:"value,omitempty"`
	Rewrites []Rewrite `json:"rewrites,omitempty"`
}

func (t *Term) Type() NodeType {
	return TermNode
}

// Clone creates a deep copy of the Term node
func (t *Term) Clone() Node {

	tc := &Term{
		Foundry: t.Foundry,
		Key:     t.Key,
		Layer:   t.Layer,
		Match:   t.Match,
		Value:   t.Value,
	}

	if t.Rewrites != nil {
		clonedRewrites := make([]Rewrite, len(t.Rewrites))
		for i, rewrite := range t.Rewrites {
			clonedRewrites[i] = *rewrite.Clone().(*Rewrite)
		}
		tc.Rewrites = clonedRewrites
	}
	return tc
}

// Pattern represents a pattern to match in the AST
type Pattern struct {
	Root Node
}

// Replacement represents a replacement pattern
type Replacement struct {
	Root Node
}

// CatchallNode represents any node type not explicitly handled
type CatchallNode struct {
	NodeType   string          // The original @type value
	RawContent json.RawMessage // The original JSON content
	Wrap       Node            // Optional wrapped node
	Operands   []Node          // Optional operands
}

func (c *CatchallNode) Type() NodeType {
	return NodeType(c.NodeType)
}

// Clone creates a deep copy of the CatchallNode
func (c *CatchallNode) Clone() Node {
	newNode := &CatchallNode{
		NodeType: c.NodeType,
	}

	// Handle RawContent properly - preserve nil if it's nil
	if c.RawContent != nil {
		newNode.RawContent = make(json.RawMessage, len(c.RawContent))
		copy(newNode.RawContent, c.RawContent)
	}

	if c.Wrap != nil {
		newNode.Wrap = c.Wrap.Clone()
	}

	if len(c.Operands) > 0 {
		newNode.Operands = make([]Node, len(c.Operands))
		for i, operand := range c.Operands {
			newNode.Operands[i] = operand.Clone()
		}
	}

	return newNode
}

// ApplyFoundryAndLayerOverrides recursively applies foundry and layer overrides to terms
func ApplyFoundryAndLayerOverrides(node Node, foundry, layer string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *Term:
		if foundry != "" {
			n.Foundry = foundry
		}
		if layer != "" {
			n.Layer = layer
		}
	case *TermGroup:
		for _, op := range n.Operands {
			ApplyFoundryAndLayerOverrides(op, foundry, layer)
		}
	case *Token:
		if n.Wrap != nil {
			ApplyFoundryAndLayerOverrides(n.Wrap, foundry, layer)
		}
	case *CatchallNode:
		if n.Wrap != nil {
			ApplyFoundryAndLayerOverrides(n.Wrap, foundry, layer)
		}
		for _, op := range n.Operands {
			ApplyFoundryAndLayerOverrides(op, foundry, layer)
		}
	}
}

// ApplyFoundryAndLayerOverridesWithPrecedence applies foundry and layer overrides while respecting precedence:
// 1. Mapping rule foundry/layer (highest priority - don't override if already set)
// 2. Passed overwrite foundry/layer (from MappingOptions)
// 3. Mapping list foundry/layer (lowest priority - defaults)
func ApplyFoundryAndLayerOverridesWithPrecedence(node Node, foundry, layer string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *Term:
		// Only override if the term doesn't already have explicit values (respecting precedence)
		if foundry != "" && n.Foundry == "" {
			n.Foundry = foundry
		}
		if layer != "" && n.Layer == "" {
			n.Layer = layer
		}
	case *TermGroup:
		for _, op := range n.Operands {
			ApplyFoundryAndLayerOverridesWithPrecedence(op, foundry, layer)
		}
	case *Token:
		if n.Wrap != nil {
			ApplyFoundryAndLayerOverridesWithPrecedence(n.Wrap, foundry, layer)
		}
	case *CatchallNode:
		if n.Wrap != nil {
			ApplyFoundryAndLayerOverridesWithPrecedence(n.Wrap, foundry, layer)
		}
		for _, op := range n.Operands {
			ApplyFoundryAndLayerOverridesWithPrecedence(op, foundry, layer)
		}
	}
}

// RestrictToObligatory takes a replacement node from a mapping rule and reduces the boolean structure
// to only obligatory operations by removing optional OR-relations and keeping required AND-relations.
// It also applies foundry and layer overrides like ApplyFoundryAndLayerOverrides().
// Note: This function is designed for mapping rule replacement nodes and does not handle CatchallNodes.
// For efficiency, restriction is performed first, then foundry/layer overrides are applied to the smaller result.
//
// Examples:
//   - (a & b & c) -> (a & b & c) (kept as is)
//   - (a & b & (c | d) & e) -> (a & b & e) (OR-relation removed)
//   - (a | b) -> nil (completely optional)
func RestrictToObligatory(node Node, foundry, layer string) Node {
	if node == nil {
		return nil
	}

	// First, clone and restrict to obligatory operations
	cloned := node.Clone()
	restricted := restrictToObligatoryRecursive(cloned)

	// Then apply foundry and layer overrides to the smaller, restricted tree
	if restricted != nil {
		ApplyFoundryAndLayerOverrides(restricted, foundry, layer)
	}

	return restricted
}

// RestrictToObligatoryWithPrecedence is like RestrictToObligatory but respects precedence rules
// when applying foundry and layer overrides
func RestrictToObligatoryWithPrecedence(node Node, foundry, layer string) Node {
	if node == nil {
		return nil
	}

	// First, clone and restrict to obligatory operations
	cloned := node.Clone()
	restricted := restrictToObligatoryRecursive(cloned)

	// Then apply foundry and layer overrides with precedence to the smaller, restricted tree
	if restricted != nil {
		ApplyFoundryAndLayerOverridesWithPrecedence(restricted, foundry, layer)
	}

	return restricted
}

// restrictToObligatoryRecursive performs the actual restriction logic
func restrictToObligatoryRecursive(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *Term:
		// Terms are always obligatory
		return n

	case *Token:
		// Process the wrapped node
		if n.Wrap != nil {
			restricted := restrictToObligatoryRecursive(n.Wrap)
			if restricted == nil {
				return nil
			}
			return &Token{
				Wrap:     restricted,
				Rewrites: n.Rewrites,
			}
		}
		return n

	case *TermGroup:
		if n.Relation == OrRelation {
			// OR-relations are optional, so remove them
			return nil
		} else if n.Relation == AndRelation {
			// AND-relations are obligatory, but we need to process operands
			var obligatoryOperands []Node
			for _, operand := range n.Operands {
				restricted := restrictToObligatoryRecursive(operand)
				if restricted != nil {
					obligatoryOperands = append(obligatoryOperands, restricted)
				}
			}

			// If no operands remain, return nil
			if len(obligatoryOperands) == 0 {
				return nil
			}

			// If only one operand remains, return it directly
			if len(obligatoryOperands) == 1 {
				return obligatoryOperands[0]
			}

			// Return the group with obligatory operands
			return &TermGroup{
				Operands: obligatoryOperands,
				Relation: AndRelation,
				Rewrites: n.Rewrites,
			}
		}
	}

	// For unknown node types, return as is
	return node
}
