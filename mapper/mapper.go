package mapper

import (
	"fmt"

	"github.com/KorAP/Koral-Mapper/config"
	"github.com/KorAP/Koral-Mapper/parser"
)

// Direction represents the mapping direction (A to B or B to A)
type Direction bool

const (
	AtoB Direction = true
	BtoA Direction = false

	RewriteEditor = "Koral-Mapper"
)

// newRewriteEntry creates a koral:rewrite annotation entry.
func newRewriteEntry(scope string, original any) map[string]any {
	r := map[string]any{
		"@type":  "koral:rewrite",
		"editor": RewriteEditor,
	}
	if scope != "" {
		r["scope"] = scope
	}
	if original != nil {
		r["original"] = original
	}
	return r
}

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
	mappingLists      map[string]*config.MappingList
	parsedQueryRules  map[string][]*parser.MappingResult
	parsedCorpusRules map[string][]*parser.CorpusMappingResult
}

// NewMapper creates a new Mapper instance from a list of MappingLists
func NewMapper(lists []config.MappingList) (*Mapper, error) {
	m := &Mapper{
		mappingLists:      make(map[string]*config.MappingList),
		parsedQueryRules:  make(map[string][]*parser.MappingResult),
		parsedCorpusRules: make(map[string][]*parser.CorpusMappingResult),
	}

	// Store mapping lists by ID
	for _, list := range lists {
		if _, exists := m.mappingLists[list.ID]; exists {
			return nil, fmt.Errorf("duplicate mapping list ID found: %s", list.ID)
		}

		listCopy := list
		m.mappingLists[list.ID] = &listCopy

		if list.IsCorpus() {
			corpusRules, err := list.ParseCorpusMappings()
			if err != nil {
				return nil, fmt.Errorf("failed to parse corpus mappings for list %s: %w", list.ID, err)
			}
			m.parsedCorpusRules[list.ID] = corpusRules
		} else {
			queryRules, err := list.ParseMappings()
			if err != nil {
				return nil, fmt.Errorf("failed to parse mappings for list %s: %w", list.ID, err)
			}
			m.parsedQueryRules[list.ID] = queryRules
		}
	}

	return m, nil
}

// MappingOptions contains the options for applying mappings
type MappingOptions struct {
	FoundryA    string
	LayerA      string
	FoundryB    string
	LayerB      string
	Direction   Direction
	AddRewrites bool
}
