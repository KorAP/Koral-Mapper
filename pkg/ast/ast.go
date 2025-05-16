package ast

// NodeType represents the type of a node in the AST
type NodeType string

const (
	TokenNode     NodeType = "token"
	TermGroupNode NodeType = "termGroup"
	TermNode      NodeType = "term"
)

// RelationType represents the type of relation between nodes
type RelationType string

const (
	AndRelation RelationType = "and"
	OrRelation  RelationType = "or"
)

// MatchType represents the type of match operation
type MatchType string

const (
	MatchEqual    MatchType = "eq"
	MatchNotEqual MatchType = "ne"
)

// Node represents a node in the AST
type Node interface {
	Type() NodeType
}

// Token represents a token node in the query
type Token struct {
	Wrap Node `json:"wrap"`
}

func (t *Token) Type() NodeType {
	return TokenNode
}

// TermGroup represents a group of terms with a relation
type TermGroup struct {
	Operands []Node       `json:"operands"`
	Relation RelationType `json:"relation"`
}

func (tg *TermGroup) Type() NodeType {
	return TermGroupNode
}

// Term represents a terminal node with matching criteria
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
