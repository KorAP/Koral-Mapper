package matcher

import (
	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
)

// Matcher handles pattern matching and replacement in the AST
type Matcher struct {
	pattern     ast.Pattern
	replacement ast.Replacement
}

// NewMatcher creates a new Matcher with the given pattern and replacement
func NewMatcher(pattern ast.Pattern, replacement ast.Replacement) *Matcher {
	return &Matcher{
		pattern:     pattern,
		replacement: replacement,
	}
}

// Match checks if the given node matches the pattern
func (m *Matcher) Match(node ast.Node) bool {
	return m.matchNode(node, m.pattern.Root)
}

// Replace replaces all occurrences of the pattern in the given node with the replacement
func (m *Matcher) Replace(node ast.Node) ast.Node {
	// If this node matches the pattern, create replacement while preserving outer structure
	if m.Match(node) {
		switch node.(type) {
		case *ast.Token:
			// For Token nodes, preserve the Token wrapper but replace its wrap
			newToken := &ast.Token{
				Wrap: m.cloneNode(m.replacement.Root),
			}
			return newToken
		default:
			return m.cloneNode(m.replacement.Root)
		}
	}

	// Otherwise recursively process children
	switch n := node.(type) {
	case *ast.Token:
		newToken := &ast.Token{
			Wrap: m.Replace(n.Wrap),
		}
		return newToken

	case *ast.TermGroup:
		newOperands := make([]ast.Node, len(n.Operands))
		for i, op := range n.Operands {
			newOperands[i] = m.Replace(op)
		}
		return &ast.TermGroup{
			Operands: newOperands,
			Relation: n.Relation,
		}

	case *ast.CatchallNode:
		newNode := &ast.CatchallNode{
			NodeType:   n.NodeType,
			RawContent: n.RawContent,
		}
		if n.Wrap != nil {
			newNode.Wrap = m.Replace(n.Wrap)
		}
		if len(n.Operands) > 0 {
			newNode.Operands = make([]ast.Node, len(n.Operands))
			for i, op := range n.Operands {
				newNode.Operands[i] = m.Replace(op)
			}
		}
		return newNode

	default:
		return node
	}
}

// matchNode recursively checks if two nodes match
func (m *Matcher) matchNode(node, pattern ast.Node) bool {
	if pattern == nil {
		return true
	}
	if node == nil {
		return false
	}

	// Handle pattern being a Token
	if pToken, ok := pattern.(*ast.Token); ok {
		if nToken, ok := node.(*ast.Token); ok {
			return m.matchNode(nToken.Wrap, pToken.Wrap)
		}
		return false
	}

	// Handle pattern being a Term
	if pTerm, ok := pattern.(*ast.Term); ok {
		// Direct term to term matching
		if t, ok := node.(*ast.Term); ok {
			return t.Foundry == pTerm.Foundry &&
				t.Key == pTerm.Key &&
				t.Layer == pTerm.Layer &&
				t.Match == pTerm.Match &&
				(pTerm.Value == "" || t.Value == pTerm.Value)
		}
		// If node is a Token, check its wrap
		if tkn, ok := node.(*ast.Token); ok {
			if tkn.Wrap == nil {
				return false
			}
			return m.matchNode(tkn.Wrap, pattern)
		}
		// If node is a TermGroup, check its operands
		if tg, ok := node.(*ast.TermGroup); ok {
			for _, op := range tg.Operands {
				if m.matchNode(op, pattern) {
					return true
				}
			}
			return false
		}
		// If node is a CatchallNode, check its wrap and operands
		if c, ok := node.(*ast.CatchallNode); ok {
			if c.Wrap != nil && m.matchNode(c.Wrap, pattern) {
				return true
			}
			for _, op := range c.Operands {
				if m.matchNode(op, pattern) {
					return true
				}
			}
			return false
		}
		return false
	}

	// Handle pattern being a TermGroup
	if pGroup, ok := pattern.(*ast.TermGroup); ok {
		// For OR relations, check if any operand matches the node
		if pGroup.Relation == ast.OrRelation {
			for _, pOp := range pGroup.Operands {
				if m.matchNode(node, pOp) {
					return true
				}
			}
			return false
		}

		// For AND relations, node must be a TermGroup with matching relation
		if tg, ok := node.(*ast.TermGroup); ok {
			if tg.Relation != pGroup.Relation {
				return false
			}
			// Check that all pattern operands match in any order
			if len(tg.Operands) < len(pGroup.Operands) {
				return false
			}
			matched := make([]bool, len(tg.Operands))
			for _, pOp := range pGroup.Operands {
				found := false
				for j, tOp := range tg.Operands {
					if !matched[j] && m.matchNode(tOp, pOp) {
						matched[j] = true
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}

		// If node is a Token, check its wrap
		if tkn, ok := node.(*ast.Token); ok {
			if tkn.Wrap == nil {
				return false
			}
			return m.matchNode(tkn.Wrap, pattern)
		}

		// If node is a CatchallNode, check its wrap and operands
		if c, ok := node.(*ast.CatchallNode); ok {
			if c.Wrap != nil && m.matchNode(c.Wrap, pattern) {
				return true
			}
			for _, op := range c.Operands {
				if m.matchNode(op, pattern) {
					return true
				}
			}
			return false
		}

		return false
	}

	return false
}

// cloneNode creates a deep copy of a node
func (m *Matcher) cloneNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Token:
		return &ast.Token{
			Wrap: m.cloneNode(n.Wrap),
		}

	case *ast.TermGroup:
		operands := make([]ast.Node, len(n.Operands))
		for i, op := range n.Operands {
			operands[i] = m.cloneNode(op)
		}
		return &ast.TermGroup{
			Operands: operands,
			Relation: n.Relation,
		}

	case *ast.Term:
		return &ast.Term{
			Foundry: n.Foundry,
			Key:     n.Key,
			Layer:   n.Layer,
			Match:   n.Match,
			Value:   n.Value,
		}

	case *ast.CatchallNode:
		newNode := &ast.CatchallNode{
			NodeType:   n.NodeType,
			RawContent: n.RawContent,
		}
		if n.Wrap != nil {
			newNode.Wrap = m.cloneNode(n.Wrap)
		}
		if len(n.Operands) > 0 {
			newNode.Operands = make([]ast.Node, len(n.Operands))
			for i, op := range n.Operands {
				newNode.Operands[i] = m.cloneNode(op)
			}
		}
		return newNode

	default:
		return nil
	}
}
