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

// TermGroup represents a koral:termGroup
type TermGroup struct {
	Operands []Node       `json:"operands"`
	Relation RelationType `json:"relation"`
	Rewrites []Rewrite    `json:"rewrites,omitempty"`
}

func (tg *TermGroup) Type() NodeType {
	return TermGroupNode
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
