package config

import (
	"fmt"
	"os"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/KorAP/KoralPipe-TermMapper/parser"
	"gopkg.in/yaml.v3"
)

const (
	defaultServer   = "https://korap.ids-mannheim.de/"
	defaultSDK      = "https://korap.ids-mannheim.de/js/korap-plugin-latest.js"
	defaultPort     = 3000
	defaultLogLevel = "warn"
)

// MappingRule represents a single mapping rule in the configuration
type MappingRule string

// MappingList represents a list of mapping rules with metadata
type MappingList struct {
	ID          string        `yaml:"id"`
	Description string        `yaml:"desc,omitempty"`
	FoundryA    string        `yaml:"foundryA,omitempty"`
	LayerA      string        `yaml:"layerA,omitempty"`
	FoundryB    string        `yaml:"foundryB,omitempty"`
	LayerB      string        `yaml:"layerB,omitempty"`
	Mappings    []MappingRule `yaml:"mappings"`
}

// MappingConfig represents the root configuration containing multiple mapping lists
type MappingConfig struct {
	SDK      string        `yaml:"sdk,omitempty"`
	Server   string        `yaml:"server,omitempty"`
	Port     int           `yaml:"port,omitempty"`
	LogLevel string        `yaml:"loglevel,omitempty"`
	Lists    []MappingList `yaml:"lists,omitempty"`
}

// LoadFromSources loads configuration from multiple sources and merges them:
// - A main configuration file (optional) containing global settings and lists
// - Individual mapping files (optional) containing single mapping lists each
// At least one source must be provided
func LoadFromSources(configFile string, mappingFiles []string) (*MappingConfig, error) {
	var allLists []MappingList
	var globalConfig MappingConfig

	// Track seen IDs across all sources to detect duplicates
	seenIDs := make(map[string]bool)

	// Load main configuration file if provided
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file '%s': %w", configFile, err)
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("EOF: config file '%s' is empty", configFile)
		}

		// Try to unmarshal as new format first (object with optional sdk/server and lists)
		if err := yaml.Unmarshal(data, &globalConfig); err == nil && len(globalConfig.Lists) > 0 {
			// Successfully parsed as new format with lists field
			for _, list := range globalConfig.Lists {
				if seenIDs[list.ID] {
					return nil, fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
				}
				seenIDs[list.ID] = true
			}
			allLists = append(allLists, globalConfig.Lists...)
		} else {
			// Fall back to old format (direct list)
			var lists []MappingList
			if err := yaml.Unmarshal(data, &lists); err != nil {
				return nil, fmt.Errorf("failed to parse YAML config file '%s': %w", configFile, err)
			}

			for _, list := range lists {
				if seenIDs[list.ID] {
					return nil, fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
				}
				seenIDs[list.ID] = true
			}
			allLists = append(allLists, lists...)
			// Clear the lists from globalConfig since we got them from the old format
			globalConfig.Lists = nil
		}
	}

	// Load individual mapping files
	for _, file := range mappingFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read mapping file '%s': %w", file, err)
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("EOF: mapping file '%s' is empty", file)
		}

		var list MappingList
		if err := yaml.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("failed to parse YAML mapping file '%s': %w", file, err)
		}

		if seenIDs[list.ID] {
			return nil, fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
		}
		seenIDs[list.ID] = true
		allLists = append(allLists, list)
	}

	// Ensure we have at least some configuration
	if len(allLists) == 0 {
		return nil, fmt.Errorf("no mapping lists found: provide either a config file (-c) with lists or mapping files (-m)")
	}

	// Validate all mapping lists
	if err := validateMappingLists(allLists); err != nil {
		return nil, err
	}

	// Create final configuration
	result := &MappingConfig{
		SDK:    globalConfig.SDK,
		Server: globalConfig.Server,
		Lists:  allLists,
	}

	// Apply defaults if not specified
	applyDefaults(result)

	return result, nil
}

// LoadConfig loads a YAML configuration file and returns a Config object
// Deprecated: Use LoadFromSources for new code
func LoadConfig(filename string) (*MappingConfig, error) {
	return LoadFromSources(filename, nil)
}

// applyDefaults sets default values for SDK and Server if they are empty
func applyDefaults(config *MappingConfig) {
	if config.SDK == "" {
		config.SDK = defaultSDK
	}
	if config.Server == "" {
		config.Server = defaultServer
	}
	if config.Port == 0 {
		config.Port = defaultPort
	}
	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
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
