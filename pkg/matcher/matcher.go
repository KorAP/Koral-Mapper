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
	if m.Match(node) {
		return m.cloneNode(m.replacement.Root)
	}

	switch n := node.(type) {
	case *ast.Token:
		n.Wrap = m.Replace(n.Wrap)
		return n

	case *ast.TermGroup:
		newOperands := make([]ast.Node, len(n.Operands))
		for i, op := range n.Operands {
			newOperands[i] = m.Replace(op)
		}
		n.Operands = newOperands
		return n

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

	switch p := pattern.(type) {
	case *ast.Token:
		if t, ok := node.(*ast.Token); ok {
			return m.matchNode(t.Wrap, p.Wrap)
		}
		return false

	case *ast.TermGroup:
		// If we're matching against a term, try to match it against any operand
		if t, ok := node.(*ast.Term); ok && p.Relation == ast.OrRelation {
			for _, op := range p.Operands {
				if m.matchNode(t, op) {
					return true
				}
			}
			return false
		}

		// If we're matching against a term group
		if t, ok := node.(*ast.TermGroup); ok {
			if t.Relation != p.Relation {
				return false
			}

			if p.Relation == ast.OrRelation {
				// For OR relation, at least one operand must match
				for _, pOp := range p.Operands {
					for _, tOp := range t.Operands {
						if m.matchNode(tOp, pOp) {
							return true
						}
					}
				}
				return false
			}

			// For AND relation, all pattern operands must match
			if len(t.Operands) < len(p.Operands) {
				return false
			}

			// Try to match pattern operands against node operands in any order
			matched := make([]bool, len(t.Operands))
			for _, pOp := range p.Operands {
				found := false
				for j, tOp := range t.Operands {
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
		return false

	case *ast.CatchallNode:
		// For catchall nodes, we need to check both wrap and operands
		if t, ok := node.(*ast.CatchallNode); ok {
			// If pattern has wrap, match it
			if p.Wrap != nil && !m.matchNode(t.Wrap, p.Wrap) {
				return false
			}

			// If pattern has operands, match them
			if len(p.Operands) > 0 {
				if len(t.Operands) < len(p.Operands) {
					return false
				}

				// Try to match pattern operands against node operands in any order
				matched := make([]bool, len(t.Operands))
				for _, pOp := range p.Operands {
					found := false
					for j, tOp := range t.Operands {
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

			// If no wrap or operands to match, it's a match
			return true
		}
		return false

	case *ast.Term:
		// If we're matching against a term group with OR relation,
		// try to match against any of its operands
		if t, ok := node.(*ast.TermGroup); ok && t.Relation == ast.OrRelation {
			for _, op := range t.Operands {
				if m.matchNode(op, p) {
					return true
				}
			}
			return false
		}

		// Direct term to term matching
		if t, ok := node.(*ast.Term); ok {
			return t.Foundry == p.Foundry &&
				t.Key == p.Key &&
				t.Layer == p.Layer &&
				t.Match == p.Match &&
				(p.Value == "" || t.Value == p.Value)
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
