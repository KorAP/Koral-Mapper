package parser

// parser is a function that takes a JSON string and returns an AST node.
// It is used to parse a JSON string into an AST node.

import (
	"encoding/json"
	"fmt"
	"strings"

	"maps"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
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
	Rewrites []ast.Rewrite   `json:"-"` // Handle manually
	// Store any additional fields
	Extra map[string]any `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *rawNode) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create a temporary struct without the problematic fields
	type tempNode struct {
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

	var temp tempNode
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy the fields
	r.Type = temp.Type
	r.Wrap = temp.Wrap
	r.Operands = temp.Operands
	r.Relation = temp.Relation
	r.Foundry = temp.Foundry
	r.Key = temp.Key
	r.Layer = temp.Layer
	r.Match = temp.Match
	r.Value = temp.Value

	// Handle rewrites manually
	if rewritesData, exists := raw["rewrites"]; exists && rewritesData != nil {
		if rewritesList, ok := rewritesData.([]any); ok {
			r.Rewrites = make([]ast.Rewrite, len(rewritesList))
			for i, rewriteData := range rewritesList {
				rewriteBytes, err := json.Marshal(rewriteData)
				if err != nil {
					return fmt.Errorf("failed to marshal rewrite %d: %w", i, err)
				}
				var rewrite ast.Rewrite
				if err := json.Unmarshal(rewriteBytes, &rewrite); err != nil {
					return fmt.Errorf("failed to unmarshal rewrite %d: %w", i, err)
				}
				r.Rewrites[i] = rewrite
			}
		}
	}

	// Store any fields not in the struct in Extra
	r.Extra = make(map[string]any)
	for k, v := range raw {
		switch k {
		case "@type", "wrap", "operands", "relation", "foundry", "key", "layer", "match", "value", "rewrites":
			continue
		default:
			r.Extra[k] = v
		}
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (r rawNode) MarshalJSON() ([]byte, error) {
	// Create a map with all fields
	raw := make(map[string]any)

	// Add the known fields if they're not empty
	raw["@type"] = r.Type
	if r.Wrap != nil {
		raw["wrap"] = r.Wrap
	}
	if len(r.Operands) > 0 {
		raw["operands"] = r.Operands
	}
	if r.Relation != "" {
		raw["relation"] = r.Relation
	}
	if r.Foundry != "" {
		raw["foundry"] = r.Foundry
	}
	if r.Key != "" {
		raw["key"] = r.Key
	}
	if r.Layer != "" {
		raw["layer"] = r.Layer
	}
	if r.Match != "" {
		raw["match"] = r.Match
	}
	if r.Value != "" {
		raw["value"] = r.Value
	}
	if len(r.Rewrites) > 0 {
		raw["rewrites"] = r.Rewrites
	}

	// Add any extra fields
	maps.Copy(raw, r.Extra)

	return json.Marshal(raw)
}

// ParseJSON parses a JSON string into our AST representation
func ParseJSON(data []byte) (ast.Node, error) {
	var raw rawNode
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if raw.Type == "" {
		return nil, fmt.Errorf("missing required field '@type' in JSON")
	}
	return parseNode(raw)
}

// parseNode converts a raw node into an AST node
func parseNode(raw rawNode) (ast.Node, error) {
	switch raw.Type {
	case "koral:token":
		if raw.Wrap == nil {
			return nil, fmt.Errorf("token node of type '%s' missing required 'wrap' field", raw.Type)
		}
		var wrapRaw rawNode
		if err := json.Unmarshal(raw.Wrap, &wrapRaw); err != nil {
			return nil, fmt.Errorf("failed to parse 'wrap' field in token node: %w", err)
		}
		wrap, err := parseNode(wrapRaw)
		if err != nil {
			return nil, fmt.Errorf("error parsing wrapped node: %w", err)
		}
		return &ast.Token{Wrap: wrap, Rewrites: raw.Rewrites}, nil

	case "koral:termGroup":
		if len(raw.Operands) == 0 {
			return nil, fmt.Errorf("term group must have at least one operand")
		}

		operands := make([]ast.Node, len(raw.Operands))
		for i, op := range raw.Operands {
			node, err := parseNode(op)
			if err != nil {
				return nil, fmt.Errorf("error parsing operand %d: %w", i+1, err)
			}
			operands[i] = node
		}

		if raw.Relation == "" {
			return nil, fmt.Errorf("term group must have a 'relation' field")
		}

		relation := ast.AndRelation
		if strings.HasSuffix(raw.Relation, "or") {
			relation = ast.OrRelation
		} else if !strings.HasSuffix(raw.Relation, "and") {
			return nil, fmt.Errorf("invalid relation type '%s', must be one of: 'relation:and', 'relation:or'", raw.Relation)
		}

		return &ast.TermGroup{
			Operands: operands,
			Relation: relation,
			Rewrites: raw.Rewrites,
		}, nil

	case "koral:term":
		if raw.Key == "" {
			return nil, fmt.Errorf("term must have a 'key' field")
		}

		match := ast.MatchEqual
		if raw.Match != "" {
			if strings.HasSuffix(raw.Match, "ne") {
				match = ast.MatchNotEqual
			} else if !strings.HasSuffix(raw.Match, "eq") {
				return nil, fmt.Errorf("invalid match type '%s', must be one of: 'match:eq', 'match:ne'", raw.Match)
			}
		}

		return &ast.Term{
			Foundry:  raw.Foundry,
			Key:      raw.Key,
			Layer:    raw.Layer,
			Match:    match,
			Value:    raw.Value,
			Rewrites: raw.Rewrites,
		}, nil

	default:
		// Store the original JSON content
		rawContent, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal unknown node type '%s': %w", raw.Type, err)
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
				return nil, fmt.Errorf("failed to parse 'wrap' field in unknown node type '%s': %w", raw.Type, err)
			}

			// Check if the wrapped node is a known type
			if wrapRaw.Type == "koral:term" || wrapRaw.Type == "koral:token" || wrapRaw.Type == "koral:termGroup" {
				wrap, err := parseNode(wrapRaw)
				if err != nil {
					return nil, fmt.Errorf("error parsing wrapped node in unknown node type '%s': %w", raw.Type, err)
				}
				catchall.Wrap = wrap
			} else {
				// For unknown types, recursively parse
				wrap, err := parseNode(wrapRaw)
				if err != nil {
					return nil, fmt.Errorf("error parsing wrapped node in unknown node type '%s': %w", raw.Type, err)
				}
				catchall.Wrap = wrap
			}
		}

		// Parse operands if present
		if len(raw.Operands) > 0 {
			operands := make([]ast.Node, len(raw.Operands))
			for i, op := range raw.Operands {
				// Check if the operand is a known type
				if op.Type == "koral:term" || op.Type == "koral:token" || op.Type == "koral:termGroup" {
					node, err := parseNode(op)
					if err != nil {
						return nil, fmt.Errorf("error parsing operand %d in unknown node type '%s': %w", i+1, raw.Type, err)
					}
					operands[i] = node
				} else {
					// For unknown types, recursively parse
					node, err := parseNode(op)
					if err != nil {
						return nil, fmt.Errorf("error parsing operand %d in unknown node type '%s': %w", i+1, raw.Type, err)
					}
					operands[i] = node
				}
			}
			catchall.Operands = operands
		}

		return catchall, nil
	}
}

// SerializeToJSON converts an AST node back to JSON
func SerializeToJSON(node ast.Node) ([]byte, error) {
	return json.MarshalIndent(nodeToRaw(node), "", "  ")
}

// nodeToRaw converts an AST node to a raw node for JSON serialization
func nodeToRaw(node ast.Node) rawNode {
	switch n := node.(type) {
	case *ast.Token:
		if n.Wrap == nil {
			return rawNode{
				Type: "koral:token",
			}
		}
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
		raw := rawNode{
			Type:  "koral:term",
			Key:   n.Key,
			Match: "match:" + string(n.Match),
		}
		if n.Foundry != "" {
			raw.Foundry = n.Foundry
		}
		if n.Layer != "" {
			raw.Layer = n.Layer
		}
		if n.Value != "" {
			raw.Value = n.Value
		}
		return raw

	case *ast.CatchallNode:
		// For catchall nodes, use the stored raw content if available
		if n.RawContent != nil {
			var raw rawNode
			if err := json.Unmarshal(n.RawContent, &raw); err == nil {
				// Ensure we preserve the node type
				raw.Type = n.NodeType

				// Handle wrap and operands if present
				if n.Wrap != nil {
					raw.Wrap = json.RawMessage(nodeToRaw(n.Wrap).toJSON())
				}
				if len(n.Operands) > 0 {
					operands := make([]rawNode, len(n.Operands))
					for i, op := range n.Operands {
						operands[i] = nodeToRaw(op)
					}
					raw.Operands = operands
				}
				return raw
			}
		}

		// If RawContent is nil or invalid, create a minimal raw node
		raw := rawNode{
			Type: n.NodeType,
		}
		if n.Wrap != nil {
			raw.Wrap = json.RawMessage(nodeToRaw(n.Wrap).toJSON())
		}
		if len(n.Operands) > 0 {
			operands := make([]rawNode, len(n.Operands))
			for i, op := range n.Operands {
				operands[i] = nodeToRaw(op)
			}
			raw.Operands = operands
		}
		return raw
	}

	// Return a minimal raw node for unknown types
	return rawNode{
		Type: "koral:unknown",
	}
}

// toJSON converts a raw node to JSON bytes
func (r rawNode) toJSON() []byte {
	data, err := json.Marshal(r)
	if err != nil {
		// Return a minimal valid JSON object if marshaling fails
		return []byte(`{"@type":"koral:unknown"}`)
	}
	return data
}
