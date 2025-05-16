package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
)

// rawNode represents the raw JSON structure
type rawNode struct {
	Type     string          `json:"@type"`
	Wrap     json.RawMessage `json:"wrap,omitempty"`
	Operands []rawNode       `json:"operands,omitempty"`
	Relation string          `json:"relation,omitempty"`
	Foundry  string          `json:"foundry,omitempty"`
	Key      string          `json:"key,omitempty"`
	Layer    string          `json:"layer,omitempty"`
	Match    string          `json:"match,omitempty"`
	Value    string          `json:"value,omitempty"`
}

// ParseJSON parses a JSON string into our AST representation
func ParseJSON(data []byte) (ast.Node, error) {
	var raw rawNode
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if raw.Type == "" {
		return nil, fmt.Errorf("missing @type field")
	}
	return parseNode(raw)
}

// parseNode converts a raw node into an AST node
func parseNode(raw rawNode) (ast.Node, error) {
	switch raw.Type {
	case "koral:token":
		if raw.Wrap == nil {
			return nil, fmt.Errorf("token node missing wrap field")
		}
		var wrapRaw rawNode
		if err := json.Unmarshal(raw.Wrap, &wrapRaw); err != nil {
			return nil, fmt.Errorf("failed to parse wrap: %w", err)
		}
		wrap, err := parseNode(wrapRaw)
		if err != nil {
			return nil, err
		}
		return &ast.Token{Wrap: wrap}, nil

	case "koral:termGroup":
		operands := make([]ast.Node, len(raw.Operands))
		for i, op := range raw.Operands {
			node, err := parseNode(op)
			if err != nil {
				return nil, err
			}
			operands[i] = node
		}

		relation := ast.AndRelation
		if strings.HasSuffix(raw.Relation, "or") {
			relation = ast.OrRelation
		}

		return &ast.TermGroup{
			Operands: operands,
			Relation: relation,
		}, nil

	case "koral:term":
		match := ast.MatchEqual
		if strings.HasSuffix(raw.Match, "ne") {
			match = ast.MatchNotEqual
		}

		return &ast.Term{
			Foundry: raw.Foundry,
			Key:     raw.Key,
			Layer:   raw.Layer,
			Match:   match,
			Value:   raw.Value,
		}, nil

	default:
		// Store the original JSON content
		rawContent, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal unknown node: %w", err)
		}

		// Create a catchall node
		catchall := &ast.CatchallNode{
			NodeType:   raw.Type,
			RawContent: rawContent,
		}

		// Parse wrap if present
		if raw.Wrap != nil {
			var wrapRaw rawNode
			if err := json.Unmarshal(raw.Wrap, &wrapRaw); err != nil {
				return nil, fmt.Errorf("failed to parse wrap in unknown node: %w", err)
			}
			wrap, err := parseNode(wrapRaw)
			if err != nil {
				return nil, err
			}
			catchall.Wrap = wrap
		}

		// Parse operands if present
		if len(raw.Operands) > 0 {
			operands := make([]ast.Node, len(raw.Operands))
			for i, op := range raw.Operands {
				node, err := parseNode(op)
				if err != nil {
					return nil, err
				}
				operands[i] = node
			}
			catchall.Operands = operands
		}

		return catchall, nil
	}
}

// SerializeToJSON converts an AST node back to JSON
func SerializeToJSON(node ast.Node) ([]byte, error) {
	raw := nodeToRaw(node)
	return json.MarshalIndent(raw, "", "  ")
}

// nodeToRaw converts an AST node to a raw node for JSON serialization
func nodeToRaw(node ast.Node) rawNode {
	switch n := node.(type) {
	case *ast.Token:
		return rawNode{
			Type: "koral:token",
			Wrap: json.RawMessage(nodeToRaw(n.Wrap).toJSON()),
		}

	case *ast.TermGroup:
		operands := make([]rawNode, len(n.Operands))
		for i, op := range n.Operands {
			operands[i] = nodeToRaw(op)
		}
		return rawNode{
			Type:     "koral:termGroup",
			Operands: operands,
			Relation: "relation:" + string(n.Relation),
		}

	case *ast.Term:
		return rawNode{
			Type:    "koral:term",
			Foundry: n.Foundry,
			Key:     n.Key,
			Layer:   n.Layer,
			Match:   "match:" + string(n.Match),
			Value:   n.Value,
		}

	case *ast.CatchallNode:
		// For catchall nodes, use the stored raw content
		if n.RawContent != nil {
			// If we have operands or wrap that were modified, we need to update the raw content
			if len(n.Operands) > 0 || n.Wrap != nil {
				var raw rawNode
				if err := json.Unmarshal(n.RawContent, &raw); err != nil {
					return rawNode{}
				}

				// Update operands if present
				if len(n.Operands) > 0 {
					raw.Operands = make([]rawNode, len(n.Operands))
					for i, op := range n.Operands {
						raw.Operands[i] = nodeToRaw(op)
					}
				}

				// Update wrap if present
				if n.Wrap != nil {
					raw.Wrap = json.RawMessage(nodeToRaw(n.Wrap).toJSON())
				}

				return raw
			}
			// If no modifications, return the original content as is
			var raw rawNode
			_ = json.Unmarshal(n.RawContent, &raw)
			return raw
		}
		return rawNode{}

	default:
		return rawNode{}
	}
}

// toJSON converts a raw node to JSON bytes
func (r rawNode) toJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}
