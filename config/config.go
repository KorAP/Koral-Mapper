package config

import (
	"fmt"
	"os"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/KorAP/KoralPipe-TermMapper/parser"
	"gopkg.in/yaml.v3"
)

const (
	defaultServer = "https://korap.ids-mannheim.de/"
	defaultSDK    = "https://korap.ids-mannheim.de/js/korap-plugin-latest.js"
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

// MappingConfig represents the root configuration containing multiple mapping lists
type MappingConfig struct {
	SDK    string        `yaml:"sdk,omitempty"`
	Server string        `yaml:"server,omitempty"`
	Lists  []MappingList `yaml:"lists,omitempty"`
}

// LoadConfig loads a YAML configuration file and returns a Config object
func LoadConfig(filename string) (*MappingConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check for empty file
	if len(data) == 0 {
		return nil, fmt.Errorf("EOF: config file is empty")
	}

	// Try to unmarshal as new format first (object with optional sdk/server and lists)
	var config MappingConfig
	if err := yaml.Unmarshal(data, &config); err == nil && len(config.Lists) > 0 {
		// Successfully parsed as new format with lists field
		if err := validateMappingLists(config.Lists); err != nil {
			return nil, err
		}
		// Apply defaults if not specified
		applyDefaults(&config)
		return &config, nil
	}

	// Fall back to old format (direct list)
	var lists []MappingList
	if err := yaml.Unmarshal(data, &lists); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := validateMappingLists(lists); err != nil {
		return nil, err
	}

	config = MappingConfig{Lists: lists}
	// Apply defaults if not specified
	applyDefaults(&config)
	return &config, nil
}

// applyDefaults sets default values for SDK and Server if they are empty
func applyDefaults(config *MappingConfig) {
	if config.SDK == "" {
		config.SDK = defaultSDK
	}
	if config.Server == "" {
		config.Server = defaultServer
	}
}

// validateMappingLists validates a slice of mapping lists
func validateMappingLists(lists []MappingList) error {
	// Validate the configuration
	seenIDs := make(map[string]bool)
	for i, list := range lists {
		if list.ID == "" {
			return fmt.Errorf("mapping list at index %d is missing an ID", i)
		}

		// Check for duplicate IDs
		if seenIDs[list.ID] {
			return fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
		}
		seenIDs[list.ID] = true

		if len(list.Mappings) == 0 {
			return fmt.Errorf("mapping list '%s' has no mapping rules", list.ID)
		}

		// Validate each mapping rule
		for j, rule := range list.Mappings {
			if rule == "" {
				return fmt.Errorf("mapping list '%s' rule at index %d is empty", list.ID, j)
			}
		}
	}
	return nil
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
		// Check for empty rules first
		if rule == "" {
			return nil, fmt.Errorf("empty mapping rule at index %d in list '%s'", i, list.ID)
		}

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
