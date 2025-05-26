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
	AndRelation   RelationType = "and"
	OrRelation    RelationType = "or"
	MatchEqual    MatchType    = "eq"
	MatchNotEqual MatchType    = "ne"
)

// Node represents a node in the AST
type Node interface {
	Type() NodeType
}

// Token represents a koral:token
type Token struct {
	Wrap Node `json:"wrap"`
}

func (t *Token) Type() NodeType {
	return TokenNode
}

// TermGroup represents a koral:termGroup
type TermGroup struct {
	Operands []Node       `json:"operands"`
	Relation RelationType `json:"relation"`
}

func (tg *TermGroup) Type() NodeType {
	return TermGroupNode
}

// Term represents a koral:term
type Term struct {
	Foundry string    `json:"foundry"`
	Key     string    `json:"key"`
	Layer   string    `json:"layer"`
	Match   MatchType `json:"match"`
	Value   string    `json:"value,omitempty"`
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
