package mapper

import (
	"fmt"

	"github.com/KorAP/KoralPipe-TermMapper/config"
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
	FoundryA    string
	LayerA      string
	FoundryB    string
	LayerB      string
	Direction   Direction
	AddRewrites bool
}
