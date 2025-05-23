package matcher

import (
	"fmt"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
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

	// If this node matches the pattern
	if m.Match(node) {
		// For TermGroups that contain a matching Term, preserve unmatched operands
		if tg, ok := node.(*ast.TermGroup); ok {
			// Check if any operand matches the pattern exactly
			hasExactMatch := false
			for _, op := range tg.Operands {
				if m.matchNode(op, m.pattern.Root) {
					hasExactMatch = true
					break
				}
			}

			// If we have an exact match, replace matching operands
			if hasExactMatch {
				hasMatch := false
				newOperands := make([]ast.Node, 0, len(tg.Operands))
				for _, op := range tg.Operands {
					if m.matchNode(op, m.pattern.Root) {
						if !hasMatch {
							newOperands = append(newOperands, m.cloneNode(m.replacement.Root))
							hasMatch = true
						} else {
							newOperands = append(newOperands, m.replaceNode(op))
						}
					} else {
						newOperands = append(newOperands, m.replaceNode(op))
					}
				}
				return &ast.TermGroup{
					Operands: newOperands,
					Relation: tg.Relation,
				}
			}
			// Otherwise, replace the entire TermGroup
			return m.cloneNode(m.replacement.Root)
		}
		// For other nodes, return the replacement
		return m.cloneNode(m.replacement.Root)
	}

	// Otherwise recursively process children
	switch n := node.(type) {
	case *ast.TermGroup:
		// Check if any operand matches the pattern exactly
		hasExactMatch := false
		for _, op := range n.Operands {
			if m.matchNode(op, m.pattern.Root) {
				hasExactMatch = true
				break
			}
		}

		// If we have an exact match, replace matching operands
		if hasExactMatch {
			hasMatch := false
			newOperands := make([]ast.Node, 0, len(n.Operands))
			for _, op := range n.Operands {
				if m.matchNode(op, m.pattern.Root) {
					if !hasMatch {
						newOperands = append(newOperands, m.cloneNode(m.replacement.Root))
						hasMatch = true
					} else {
						newOperands = append(newOperands, m.replaceNode(op))
					}
				} else {
					newOperands = append(newOperands, m.replaceNode(op))
				}
			}
			return &ast.TermGroup{
				Operands: newOperands,
				Relation: n.Relation,
			}
		}
		// Otherwise, recursively process operands
		newOperands := make([]ast.Node, len(n.Operands))
		for i, op := range n.Operands {
			newOperands[i] = m.replaceNode(op)
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
			newNode.Wrap = m.replaceNode(n.Wrap)
		}
		if len(n.Operands) > 0 {
			newNode.Operands = make([]ast.Node, len(n.Operands))
			for i, op := range n.Operands {
				newNode.Operands[i] = m.replaceNode(op)
			}
		}
		return newNode

	default:
		return node
	}
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
			// If we have a single operand, return it directly
			// But only if we're not inside a Token
			if _, isToken := node.(*ast.Token); !isToken {
				return simplified[0]
			}
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
