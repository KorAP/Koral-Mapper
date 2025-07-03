package matcher

import (
	"fmt"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
)

// Matcher handles pattern matching and replacement in the AST
type Matcher struct {
	pattern     ast.Pattern
	replacement ast.Replacement
}

// validateNode checks if a node is valid for pattern/replacement ASTs
func validateNode(node ast.Node) error {
	if node == nil {
		return fmt.Errorf("nil node")
	}

	switch n := node.(type) {
	case *ast.Token:
		if n.Wrap != nil {
			return validateNode(n.Wrap)
		}
		return nil
	case *ast.Term:
		return nil
	case *ast.TermGroup:
		if len(n.Operands) == 0 {
			return fmt.Errorf("empty term group")
		}
		for _, op := range n.Operands {
			if err := validateNode(op); err != nil {
				return fmt.Errorf("invalid operand: %v", err)
			}
		}
		return nil
	case *ast.CatchallNode:
		return fmt.Errorf("catchall nodes are not allowed in pattern/replacement ASTs")
	default:
		return fmt.Errorf("unknown node type: %T", node)
	}
}

// NewMatcher creates a new Matcher with the given pattern and replacement
func NewMatcher(pattern ast.Pattern, replacement ast.Replacement) (*Matcher, error) {
	if err := validateNode(pattern.Root); err != nil {
		return nil, fmt.Errorf("invalid pattern: %v", err)
	}
	if err := validateNode(replacement.Root); err != nil {
		return nil, fmt.Errorf("invalid replacement: %v", err)
	}
	return &Matcher{
		pattern:     pattern,
		replacement: replacement,
	}, nil
}

// Match checks if the given node matches the pattern
func (m *Matcher) Match(node ast.Node) bool {
	return m.matchNode(node, m.pattern.Root)
}

// Replace replaces all occurrences of the pattern in the given node with the replacement
func (m *Matcher) Replace(node ast.Node) ast.Node {
	// First step: Create complete structure with replacements
	replaced := m.replaceNode(node)
	// Second step: Simplify the structure
	simplified := m.simplifyNode(replaced)
	// If the input was a Token, ensure the output is also a Token
	if _, isToken := node.(*ast.Token); isToken {
		if _, isToken := simplified.(*ast.Token); !isToken {
			return &ast.Token{Wrap: simplified}
		}
	}
	return simplified
}

// replaceNode creates a complete structure with replacements
func (m *Matcher) replaceNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	// First handle Token nodes specially to preserve their structure
	if token, ok := node.(*ast.Token); ok {
		if token.Wrap == nil {
			return token
		}
		// Process the wrapped node
		wrap := m.replaceNode(token.Wrap)
		return &ast.Token{Wrap: wrap}
	}

	// Handle TermGroup nodes
	if tg, ok := node.(*ast.TermGroup); ok {
		// Check if any operand matches the pattern
		hasMatch := false
		newOperands := make([]ast.Node, 0, len(tg.Operands))
		for _, op := range tg.Operands {
			if !hasMatch && m.matchNode(op, m.pattern.Root) {
				newOperands = append(newOperands, m.cloneNode(m.replacement.Root))
				hasMatch = true
			} else {
				newOperands = append(newOperands, m.replaceNode(op))
			}
		}
		// If we found a match, return the modified TermGroup
		if hasMatch {
			return &ast.TermGroup{
				Operands: newOperands,
				Relation: tg.Relation,
			}
		}
		// If this TermGroup matches the pattern exactly, replace it
		if m.matchNode(node, m.pattern.Root) {
			return m.cloneNode(m.replacement.Root)
		}
		// Otherwise, return the modified TermGroup
		return &ast.TermGroup{
			Operands: newOperands,
			Relation: tg.Relation,
		}
	}

	// Handle CatchallNode nodes
	if c, ok := node.(*ast.CatchallNode); ok {
		newNode := &ast.CatchallNode{
			NodeType:   c.NodeType,
			RawContent: c.RawContent,
		}
		if c.Wrap != nil {
			newNode.Wrap = m.replaceNode(c.Wrap)
		}
		if len(c.Operands) > 0 {
			newNode.Operands = make([]ast.Node, len(c.Operands))
			for i, op := range c.Operands {
				newNode.Operands[i] = m.replaceNode(op)
			}
		}
		return newNode
	}

	// If this node matches the pattern exactly, replace it
	if m.matchNode(node, m.pattern.Root) {
		return m.cloneNode(m.replacement.Root)
	}

	return node
}

// simplifyNode removes unnecessary wrappers and empty nodes
func (m *Matcher) simplifyNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Token:
		if n.Wrap == nil {
			return nil
		}
		simplified := m.simplifyNode(n.Wrap)
		if simplified == nil {
			return nil
		}
		return &ast.Token{Wrap: simplified}

	case *ast.TermGroup:
		// First simplify all operands
		simplified := make([]ast.Node, 0, len(n.Operands))
		for _, op := range n.Operands {
			if s := m.simplifyNode(op); s != nil {
				simplified = append(simplified, s)
			}
		}

		// Handle special cases
		if len(simplified) == 0 {
			return nil
		}
		if len(simplified) == 1 {
			return simplified[0]
		}

		return &ast.TermGroup{
			Operands: simplified,
			Relation: n.Relation,
		}

	case *ast.CatchallNode:
		newNode := &ast.CatchallNode{
			NodeType:   n.NodeType,
			RawContent: n.RawContent,
		}
		if n.Wrap != nil {
			newNode.Wrap = m.simplifyNode(n.Wrap)
		}
		if len(n.Operands) > 0 {
			simplified := make([]ast.Node, 0, len(n.Operands))
			for _, op := range n.Operands {
				if s := m.simplifyNode(op); s != nil {
					simplified = append(simplified, s)
				}
			}
			if len(simplified) > 0 {
				newNode.Operands = simplified
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

	// Handle wrapped nodes (Token and CatchallNode)
	if m.tryMatchWrapped(node, pattern) {
		return true
	}

	switch p := pattern.(type) {
	case *ast.Token:
		if n, ok := node.(*ast.Token); ok {
			return m.matchNode(n.Wrap, p.Wrap)
		}

	case *ast.Term:
		return m.matchTerm(node, p)

	case *ast.TermGroup:
		if p.Relation == ast.OrRelation {
			// For OR relations, check if any operand matches
			for _, pOp := range p.Operands {
				if m.matchNode(node, pOp) {
					return true
				}
			}
		} else if tg, ok := node.(*ast.TermGroup); ok && tg.Relation == p.Relation {
			// For AND relations, all pattern operands must match in any order
			return m.matchAndTermGroup(tg, p)
		}
	}

	return false
}

// tryMatchWrapped attempts to match a node that might wrap other nodes
func (m *Matcher) tryMatchWrapped(node, pattern ast.Node) bool {
	switch n := node.(type) {
	case *ast.Token:
		return n.Wrap != nil && m.matchNode(n.Wrap, pattern)
	case *ast.CatchallNode:
		if n.Wrap != nil && m.matchNode(n.Wrap, pattern) {
			return true
		}
		for _, op := range n.Operands {
			if m.matchNode(op, pattern) {
				return true
			}
		}
	case *ast.TermGroup:
		for _, op := range n.Operands {
			if m.matchNode(op, pattern) {
				return true
			}
		}
	}
	return false
}

// matchTerm checks if a node matches a term pattern
func (m *Matcher) matchTerm(node ast.Node, pattern *ast.Term) bool {
	if t, ok := node.(*ast.Term); ok {
		return t.Foundry == pattern.Foundry &&
			t.Key == pattern.Key &&
			t.Layer == pattern.Layer &&
			t.Match == pattern.Match &&
			(pattern.Value == "" || t.Value == pattern.Value)
	}
	return m.tryMatchWrapped(node, pattern)
}

// matchAndTermGroup checks if a TermGroup matches an AND pattern
func (m *Matcher) matchAndTermGroup(node *ast.TermGroup, pattern *ast.TermGroup) bool {
	if len(node.Operands) < len(pattern.Operands) {
		return false
	}
	matched := make([]bool, len(node.Operands))
	for _, pOp := range pattern.Operands {
		found := false
		for j, tOp := range node.Operands {
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
