package mapper

import (
	"encoding/json"
	"fmt"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/KorAP/KoralPipe-TermMapper/config"
	"github.com/KorAP/KoralPipe-TermMapper/matcher"
	"github.com/KorAP/KoralPipe-TermMapper/parser"
)

// Direction represents the mapping direction (A to B or B to A)
type Direction bool

const (
	AtoB Direction = true
	BtoA Direction = false
)

// String converts the Direction to its string representation
func (d Direction) String() string {
	if d {
		return "atob"
	}
	return "btoa"
}

// ParseDirection converts a string direction to Direction type
func ParseDirection(dir string) (Direction, error) {
	switch dir {
	case "atob":
		return AtoB, nil
	case "btoa":
		return BtoA, nil
	default:
		return false, fmt.Errorf("invalid direction: %s", dir)
	}
}

// Mapper handles the application of mapping rules to JSON objects
type Mapper struct {
	mappingLists map[string]*config.MappingList
	parsedRules  map[string][]*parser.MappingResult
}

// NewMapper creates a new Mapper instance from a list of MappingLists
func NewMapper(lists []config.MappingList) (*Mapper, error) {
	m := &Mapper{
		mappingLists: make(map[string]*config.MappingList),
		parsedRules:  make(map[string][]*parser.MappingResult),
	}

	// Store mapping lists by ID
	for _, list := range lists {
		if _, exists := m.mappingLists[list.ID]; exists {
			return nil, fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
		}

		// Create a copy of the list to store
		listCopy := list
		m.mappingLists[list.ID] = &listCopy

		// Parse the rules immediately
		parsedRules, err := list.ParseMappings()
		if err != nil {
			return nil, fmt.Errorf("failed to parse mappings for list %s: %w", list.ID, err)
		}
		m.parsedRules[list.ID] = parsedRules
	}

	return m, nil
}

// MappingOptions contains the options for applying mappings
type MappingOptions struct {
	FoundryA  string
	LayerA    string
	FoundryB  string
	LayerB    string
	Direction Direction
}

// ApplyMappings applies the specified mapping rules to a JSON object
func (m *Mapper) ApplyMappings(mappingID string, opts MappingOptions, jsonData any) (any, error) {
	// Validate mapping ID
	if _, exists := m.mappingLists[mappingID]; !exists {
		return nil, fmt.Errorf("mapping list with ID %s not found", mappingID)
	}

	// Get the parsed rules
	rules := m.parsedRules[mappingID]

	// Convert input JSON to AST
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input JSON: %w", err)
	}

	node, err := parser.ParseJSON(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON into AST: %w", err)
	}

	// Store whether the input was a Token
	isToken := false
	var tokenWrap ast.Node
	if token, ok := node.(*ast.Token); ok {
		isToken = true
		tokenWrap = token.Wrap
		node = tokenWrap
	}

	// Apply each rule to the AST
	for _, rule := range rules {
		// Create pattern and replacement based on direction
		var pattern, replacement ast.Node
		if opts.Direction { // true means AtoB
			pattern = rule.Upper
			replacement = rule.Lower
		} else {
			pattern = rule.Lower
			replacement = rule.Upper
		}

		// Extract the inner nodes from the pattern and replacement tokens
		if token, ok := pattern.(*ast.Token); ok {
			pattern = token.Wrap
		}
		if token, ok := replacement.(*ast.Token); ok {
			replacement = token.Wrap
		}

		// Apply foundry and layer overrides
		if opts.Direction { // true means AtoB
			applyFoundryAndLayerOverrides(pattern, opts.FoundryA, opts.LayerA)
			applyFoundryAndLayerOverrides(replacement, opts.FoundryB, opts.LayerB)
		} else {
			applyFoundryAndLayerOverrides(pattern, opts.FoundryB, opts.LayerB)
			applyFoundryAndLayerOverrides(replacement, opts.FoundryA, opts.LayerA)
		}

		// Create matcher and apply replacement
		m, err := matcher.NewMatcher(ast.Pattern{Root: pattern}, ast.Replacement{Root: replacement})
		if err != nil {
			return nil, fmt.Errorf("failed to create matcher: %w", err)
		}
		node = m.Replace(node)
	}

	// Wrap the result in a token if the input was a token
	var result ast.Node
	if isToken {
		result = &ast.Token{Wrap: node}
	} else {
		result = node
	}

	// Convert AST back to JSON
	resultBytes, err := parser.SerializeToJSON(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize AST to JSON: %w", err)
	}

	// Parse the JSON string back into an interface{}
	var resultData interface{}
	if err := json.Unmarshal(resultBytes, &resultData); err != nil {
		return nil, fmt.Errorf("failed to parse result JSON: %w", err)
	}

	return resultData, nil
}

// applyFoundryAndLayerOverrides recursively applies foundry and layer overrides to terms
func applyFoundryAndLayerOverrides(node ast.Node, foundry, layer string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Term:
		if foundry != "" {
			n.Foundry = foundry
		}
		if layer != "" {
			n.Layer = layer
		}
	case *ast.TermGroup:
		for _, op := range n.Operands {
			applyFoundryAndLayerOverrides(op, foundry, layer)
		}
	case *ast.Token:
		if n.Wrap != nil {
			applyFoundryAndLayerOverrides(n.Wrap, foundry, layer)
		}
	case *ast.CatchallNode:
		if n.Wrap != nil {
			applyFoundryAndLayerOverrides(n.Wrap, foundry, layer)
		}
		for _, op := range n.Operands {
			applyFoundryAndLayerOverrides(op, foundry, layer)
		}
	}
}
