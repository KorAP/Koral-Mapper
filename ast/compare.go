package ast

import (
	"reflect"
)

// NodesEqual compares two AST nodes for equality
func NodesEqual(a, b Node) bool {
	if a == nil || b == nil {
		return a == b
	}

	if a.Type() != b.Type() {
		return false
	}

	switch n1 := a.(type) {
	case *Term:
		if n2, ok := b.(*Term); ok {
			return n1.Foundry == n2.Foundry &&
				n1.Key == n2.Key &&
				n1.Layer == n2.Layer &&
				n1.Match == n2.Match &&
				n1.Value == n2.Value
		}
	case *TermGroup:
		if n2, ok := b.(*TermGroup); ok {
			if n1.Relation != n2.Relation || len(n1.Operands) != len(n2.Operands) {
				return false
			}
			for i := range n1.Operands {
				if !NodesEqual(n1.Operands[i], n2.Operands[i]) {
					return false
				}
			}
			return true
		}
	case *Token:
		if n2, ok := b.(*Token); ok {
			return NodesEqual(n1.Wrap, n2.Wrap)
		}
	case *CatchallNode:
		if n2, ok := b.(*CatchallNode); ok {
			return n1.NodeType == n2.NodeType &&
				reflect.DeepEqual(n1.RawContent, n2.RawContent) &&
				NodesEqual(n1.Wrap, n2.Wrap)
		}
	}
	return false
}

// IsTermNode checks if a node is a Term node
func IsTermNode(node Node) bool {
	_, ok := node.(*Term)
	return ok
}
