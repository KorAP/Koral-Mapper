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
				n1.Value == n2.Value &&
				rewritesEqual(n1.Rewrites, n2.Rewrites)
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
			return rewritesEqual(n1.Rewrites, n2.Rewrites)
		}
	case *Token:
		if n2, ok := b.(*Token); ok {
			return NodesEqual(n1.Wrap, n2.Wrap) &&
				rewritesEqual(n1.Rewrites, n2.Rewrites)
		}
	case *CatchallNode:
		if n2, ok := b.(*CatchallNode); ok {
			if n1.NodeType != n2.NodeType ||
				!reflect.DeepEqual(n1.RawContent, n2.RawContent) ||
				!NodesEqual(n1.Wrap, n2.Wrap) {
				return false
			}
			// Compare operands
			if len(n1.Operands) != len(n2.Operands) {
				return false
			}
			for i := range n1.Operands {
				if !NodesEqual(n1.Operands[i], n2.Operands[i]) {
					return false
				}
			}
			return true
		}
	case *Rewrite:
		if n2, ok := b.(*Rewrite); ok {
			return n1.Editor == n2.Editor &&
				n1.Operation == n2.Operation &&
				n1.Scope == n2.Scope &&
				n1.Src == n2.Src &&
				n1.Comment == n2.Comment &&
				reflect.DeepEqual(n1.Original, n2.Original)
		}
	}
	return false
}

// rewritesEqual compares two slices of Rewrite structs for equality
func rewritesEqual(a, b []Rewrite) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Editor != b[i].Editor ||
			a[i].Operation != b[i].Operation ||
			a[i].Scope != b[i].Scope ||
			a[i].Src != b[i].Src ||
			a[i].Comment != b[i].Comment ||
			!reflect.DeepEqual(a[i].Original, b[i].Original) {
			return false
		}
	}
	return true
}

// IsTermNode checks if a node is a Term node
func IsTermNode(node Node) bool {
	_, ok := node.(*Term)
	return ok
}
