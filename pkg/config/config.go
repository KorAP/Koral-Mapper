package config

import (
	"fmt"
	"os"

	"github.com/KorAP/KoralPipe-TermMapper2/pkg/ast"
	"github.com/KorAP/KoralPipe-TermMapper2/pkg/parser"
	"gopkg.in/yaml.v3"
)

// MappingRule represents a single mapping rule in the configuration
type MappingRule string

// MappingList represents a list of mapping rules with metadata
type MappingList struct {
	ID       string        `yaml:"id"`
	FoundryA string        `yaml:"foundryA,omitempty"`
	LayerA   string        `yaml:"layerA,omitempty"`
	FoundryB string        `yaml:"foundryB,omitempty"`
	LayerB   string        `yaml:"layerB,omitempty"`
	Mappings []MappingRule `yaml:"mappings"`
}

// Config represents the root configuration containing multiple mapping lists
type Config struct {
	Lists []MappingList
}

// LoadConfig loads a YAML configuration file and returns a Config object
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var lists []MappingList
	if err := yaml.Unmarshal(data, &lists); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate the configuration
	for i, list := range lists {
		if list.ID == "" {
			return nil, fmt.Errorf("mapping list at index %d is missing an ID", i)
		}
		if len(list.Mappings) == 0 {
			return nil, fmt.Errorf("mapping list '%s' has no mapping rules", list.ID)
		}

		// Validate each mapping rule
		for j, rule := range list.Mappings {
			if rule == "" {
				return nil, fmt.Errorf("mapping list '%s' rule at index %d is empty", list.ID, j)
			}
		}
	}

	return &Config{Lists: lists}, nil
}

// ParseMappings parses all mapping rules in a list and returns a slice of parsed rules
func (list *MappingList) ParseMappings() ([]*parser.MappingResult, error) {
	// Create a grammar parser with the list's default foundries and layers
	grammarParser, err := parser.NewGrammarParser("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to create grammar parser: %w", err)
	}

	results := make([]*parser.MappingResult, len(list.Mappings))
	for i, rule := range list.Mappings {
		// Parse the mapping rule
		result, err := grammarParser.ParseMapping(string(rule))
		if err != nil {
			return nil, fmt.Errorf("failed to parse mapping rule %d in list '%s': %w", i, list.ID, err)
		}

		// Apply default foundries and layers if not specified in the rule
		if list.FoundryA != "" {
			applyDefaultFoundryAndLayer(result.Upper.Wrap, list.FoundryA, list.LayerA)
		}
		if list.FoundryB != "" {
			applyDefaultFoundryAndLayer(result.Lower.Wrap, list.FoundryB, list.LayerB)
		}

		results[i] = result
	}

	return results, nil
}

// applyDefaultFoundryAndLayer recursively applies default foundry and layer to terms that don't have them specified
func applyDefaultFoundryAndLayer(node ast.Node, defaultFoundry, defaultLayer string) {
	switch n := node.(type) {
	case *ast.Term:
		if n.Foundry == "" {
			n.Foundry = defaultFoundry
		}
		if n.Layer == "" {
			n.Layer = defaultLayer
		}
	case *ast.TermGroup:
		for _, op := range n.Operands {
			applyDefaultFoundryAndLayer(op, defaultFoundry, defaultLayer)
		}
	}
}
