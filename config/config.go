package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/parser"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	defaultServer     = "https://korap.ids-mannheim.de/"
	defaultSDK        = "https://korap.ids-mannheim.de/js/korap-plugin-latest.js"
	defaultStylesheet = "https://korap.ids-mannheim.de/css/kalamar-plugin-latest.css"
	defaultServiceURL = "https://korap.ids-mannheim.de/plugin/koralmapper"
	defaultCookieName = "km-config"
	defaultPort       = 5725
	defaultLogLevel     = "warn"
	defaultRateLimit    = 100
)

// MappingRule represents a single mapping rule in the configuration
type MappingRule string

// MappingList represents a list of mapping rules with metadata
type MappingList struct {
	ID          string        `yaml:"id"`
	Type        string        `yaml:"type,omitempty"` // "annotation" (default) or "corpus"
	Description string        `yaml:"desc,omitempty"`
	FoundryA    string        `yaml:"foundryA,omitempty"`
	LayerA      string        `yaml:"layerA,omitempty"`
	FoundryB    string        `yaml:"foundryB,omitempty"`
	LayerB      string        `yaml:"layerB,omitempty"`
	FieldA      string        `yaml:"fieldA,omitempty"`
	FieldB      string        `yaml:"fieldB,omitempty"`
	Rewrites    bool          `yaml:"rewrites,omitempty"`
	Mappings    []MappingRule `yaml:"mappings"`
}

// IsCorpus returns true if the mapping list type is "corpus".
func (list *MappingList) IsCorpus() bool {
	return list.Type == "corpus"
}

// ParseCorpusMappings parses all mapping rules as corpus rules.
// Bare values (without key=) are always allowed and receive the default
// field name from the mapping list header (FieldA/FieldB) when set.
func (list *MappingList) ParseCorpusMappings() ([]*parser.CorpusMappingResult, error) {
	corpusParser := parser.NewCorpusParser()
	corpusParser.AllowBareValues = true

	results := make([]*parser.CorpusMappingResult, len(list.Mappings))
	for i, rule := range list.Mappings {
		if rule == "" {
			return nil, fmt.Errorf("empty corpus mapping rule at index %d in list '%s'", i, list.ID)
		}
		result, err := corpusParser.ParseMapping(string(rule))
		if err != nil {
			return nil, fmt.Errorf("failed to parse corpus mapping rule %d in list '%s': %w", i, list.ID, err)
		}

		if list.FieldA != "" {
			applyDefaultCorpusKey(result.Upper, list.FieldA)
		}
		if list.FieldB != "" {
			applyDefaultCorpusKey(result.Lower, list.FieldB)
		}

		results[i] = result
	}
	return results, nil
}

// applyDefaultCorpusKey recursively fills in empty keys on CorpusField nodes.
func applyDefaultCorpusKey(node parser.CorpusNode, defaultKey string) {
	switch n := node.(type) {
	case *parser.CorpusField:
		if n.Key == "" {
			n.Key = defaultKey
		}
	case *parser.CorpusGroup:
		for _, op := range n.Operands {
			applyDefaultCorpusKey(op, defaultKey)
		}
	}
}

// MappingConfig represents the root configuration containing multiple mapping lists
type MappingConfig struct {
	SDK          string        `yaml:"sdk,omitempty"`
	Stylesheet   string        `yaml:"stylesheet,omitempty"`
	Server       string        `yaml:"server,omitempty"`
	ServiceURL   string        `yaml:"serviceURL,omitempty"`
	CookieName   string        `yaml:"cookieName,omitempty"`
	BasePath     string        `yaml:"basePath,omitempty"`     // restricts config file loading to this directory tree
	AllowOrigins string        `yaml:"allowOrigins,omitempty"` // comma-separated list of allowed CORS origins
	Port         int           `yaml:"port,omitempty"`
	LogLevel     string        `yaml:"loglevel,omitempty"`
	RateLimit    int           `yaml:"rateLimit,omitempty"` // max requests per minute per IP (0 = use default 100)
	Lists        []MappingList `yaml:"lists,omitempty"`
}

// AllowedBasePath restricts file loading to a specific directory tree.
// When set, all file paths must resolve to a location at or below this
// directory (or under the system temp directory). Defaults to the CWD at
// application startup; can be overridden via the "basePath" YAML config
// field or the KORAL_MAPPER_BASE_PATH environment variable. In Docker
// (WORKDIR /), the default "/" naturally allows all paths.
var AllowedBasePath string

// isWithinDir checks whether absPath is at or below the given directory.
// Uses a trailing-separator comparison to avoid prefix false positives
// (e.g. /home/user must not match /home/username).
func isWithinDir(absPath, dir string) bool {
	if dir == "/" {
		return true
	}
	return absPath == dir || strings.HasPrefix(absPath, dir+string(filepath.Separator))
}

// sanitizeFilePath cleans a file path, resolves it to an absolute path, and
// (when AllowedBasePath is set) verifies it resides at or below the allowed
// base directory or the system temp directory. This prevents path
// traversal attacks by ensuring os.ReadFile never receives
// unsanitized user input and cannot access files outside the application's
// working tree.
func sanitizeFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty file path")
	}

	// Clean the path to remove redundant separators and resolve "." and ".."
	cleaned := filepath.Clean(path)

	// Convert to absolute path so all traversal is resolved against the CWD
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for '%s': %w", path, err)
	}

	// If a base path is configured, confine access to that tree or temp dir
	if AllowedBasePath != "" {
		base := filepath.Clean(AllowedBasePath)
		tmpDir := filepath.Clean(os.TempDir())

		if !isWithinDir(absPath, base) && !isWithinDir(absPath, tmpDir) {
			return "", fmt.Errorf(
				"path traversal detected: '%s' resolves to '%s' which is outside the allowed base '%s'",
				path, absPath, base)
		}
	}

	return absPath, nil
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
		safePath, err := sanitizeFilePath(configFile)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(safePath) // #nosec G304 -- path sanitized above
		if err != nil {
			return nil, fmt.Errorf("failed to read config file '%s': %w", configFile, err)
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("EOF: config file '%s' is empty", configFile)
		}

		// Try to unmarshal as new format first (object with optional sdk/server and lists)
		if err := yaml.Unmarshal(data, &globalConfig); err == nil {
			// Successfully parsed as new format - accept it regardless of whether it has lists
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
		safePath, err := sanitizeFilePath(file)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(safePath) // #nosec G304 -- path sanitized above
		if err != nil {
			log.Error().Err(err).Str("file", file).Msg("Failed to read mapping file")
			continue
		}

		if len(data) == 0 {
			log.Error().Err(err).Str("file", file).Msg("EOF: mapping file is empty")
			continue
		}

		var list MappingList
		if err := yaml.Unmarshal(data, &list); err != nil {
			log.Error().Err(err).Str("file", file).Msg("Failed to parse YAML mapping file")
			continue
		}

		if seenIDs[list.ID] {
			log.Error().Err(err).Str("file", file).Str("list-id", list.ID).Msg("Duplicate mapping list ID found")
			continue
		}
		seenIDs[list.ID] = true
		allLists = append(allLists, list)
	}

	// Ensure we have at least some configuration
	if len(allLists) == 0 {
		return nil, fmt.Errorf("no mapping lists found: provide either a config file (-c) with lists or mapping files (-m)")
	}

	// Validate all mapping lists (skip duplicate ID check since we already did it)
	if err := validateMappingLists(allLists); err != nil {
		return nil, err
	}

	// Create final configuration
	result := &MappingConfig{
		SDK:          globalConfig.SDK,
		Stylesheet:   globalConfig.Stylesheet,
		Server:       globalConfig.Server,
		ServiceURL:   globalConfig.ServiceURL,
		BasePath:     globalConfig.BasePath,
		AllowOrigins: globalConfig.AllowOrigins,
		Port:         globalConfig.Port,
		LogLevel:     globalConfig.LogLevel,
		RateLimit:    globalConfig.RateLimit,
		Lists:        allLists,
	}

	// Apply environment variable overrides (ENV > config file)
	ApplyEnvOverrides(result)

	// Apply defaults if not specified
	ApplyDefaults(result)

	return result, nil
}

// ApplyDefaults sets default values for configuration fields if they are empty
func ApplyDefaults(config *MappingConfig) {
	defaults := map[*string]string{
		&config.SDK:        defaultSDK,
		&config.Stylesheet: defaultStylesheet,
		&config.Server:     defaultServer,
		&config.ServiceURL: defaultServiceURL,
		&config.CookieName: defaultCookieName,
		&config.LogLevel:   defaultLogLevel,
	}

	for field, defaultValue := range defaults {
		if *field == "" {
			*field = defaultValue
		}
	}

	// AllowOrigins defaults to the Server value (with trailing slash
	// stripped to form a proper origin). This avoids duplicating the
	// server URL string and keeps CORS in sync with the deployment.
	if config.AllowOrigins == "" {
		config.AllowOrigins = strings.TrimRight(config.Server, "/")
	}

	if config.Port == 0 {
		config.Port = defaultPort
	}
	if config.RateLimit == 0 {
		config.RateLimit = defaultRateLimit
	}
}

// ApplyEnvOverrides overrides configuration fields from environment variables.
// All environment variables are uppercase and prefixed with KORAL_MAPPER_.
// Non-empty environment values override any previously loaded config values.
func ApplyEnvOverrides(config *MappingConfig) {
	envMappings := map[string]*string{
		"KORAL_MAPPER_SERVER":        &config.Server,
		"KORAL_MAPPER_SDK":           &config.SDK,
		"KORAL_MAPPER_STYLESHEET":    &config.Stylesheet,
		"KORAL_MAPPER_SERVICE_URL":   &config.ServiceURL,
		"KORAL_MAPPER_COOKIE_NAME":   &config.CookieName,
		"KORAL_MAPPER_LOG_LEVEL":     &config.LogLevel,
		"KORAL_MAPPER_BASE_PATH":     &config.BasePath,
		"KORAL_MAPPER_ALLOW_ORIGINS": &config.AllowOrigins,
	}

	for envKey, field := range envMappings {
		if val := os.Getenv(envKey); val != "" {
			*field = val
		}
	}

	if val := os.Getenv("KORAL_MAPPER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}

	if val := os.Getenv("KORAL_MAPPER_RATE_LIMIT"); val != "" {
		if rl, err := strconv.Atoi(val); err == nil {
			config.RateLimit = rl
		}
	}
}

// validateMappingLists validates a slice of mapping lists (without duplicate ID checking)
func validateMappingLists(lists []MappingList) error {
	for i, list := range lists {
		if list.ID == "" {
			return fmt.Errorf("mapping list at index %d is missing an ID", i)
		}

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
		if n.Foundry == "" && defaultFoundry != "" {
			n.Foundry = defaultFoundry
		}
		if n.Layer == "" && defaultLayer != "" {
			n.Layer = defaultLayer
		}
	case *ast.TermGroup:
		for _, op := range n.Operands {
			applyDefaultFoundryAndLayer(op, defaultFoundry, defaultLayer)
		}
	}
}
