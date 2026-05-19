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
	FieldA      string
	FieldB      string
	Direction   Direction
	AddRewrites bool
}

// validateEffectiveOptions checks that the resolved source and target
// identifiers are not identical, which would cause an infinite mapping loop.
// For annotation mappings it compares the effective foundry+layer pair;
// for corpus mappings it compares the effective field names.
// The effective value is: query-parameter override if non-empty, otherwise
// the YAML list default.
func (m *Mapper) validateEffectiveOptions(mappingID string, opts MappingOptions) error {
	list, exists := m.mappingLists[mappingID]
	if !exists {
		return nil // will be caught later
	}

	if list.IsCorpus() {
		effFieldA := opts.FieldA
		if effFieldA == "" {
			effFieldA = list.FieldA
		}
		effFieldB := opts.FieldB
		if effFieldB == "" {
			effFieldB = list.FieldB
		}
		if effFieldA != "" && effFieldA == effFieldB {
			return fmt.Errorf("identical source and target field (fieldA == fieldB == %q) in mapping list '%s': this would cause an infinite mapping loop", effFieldA, mappingID)
		}
		return nil
	}

	effFoundryA := opts.FoundryA
	if effFoundryA == "" {
		effFoundryA = list.FoundryA
	}
	effLayerA := opts.LayerA
	if effLayerA == "" {
		effLayerA = list.LayerA
	}
	effFoundryB := opts.FoundryB
	if effFoundryB == "" {
		effFoundryB = list.FoundryB
	}
	effLayerB := opts.LayerB
	if effLayerB == "" {
		effLayerB = list.LayerB
	}

	if effFoundryA != "" && effFoundryA == effFoundryB && effLayerA == effLayerB {
		return fmt.Errorf("identical source and target (foundryA/layerA == foundryB/layerB == %q/%q) in mapping list '%s': this would cause an infinite mapping loop", effFoundryA, effLayerA, mappingID)
	}

	return nil
}

// CascadeQueryMappings applies multiple mapping lists sequentially,
// feeding the output of each into the next. orderedIDs and
// perMappingOpts must have the same length. An empty list returns
// jsonData unchanged.
func (m *Mapper) CascadeQueryMappings(orderedIDs []string, perMappingOpts []MappingOptions, jsonData any) (any, error) {
	if len(orderedIDs) != len(perMappingOpts) {
		return nil, fmt.Errorf("orderedIDs length (%d) must match perMappingOpts length (%d)", len(orderedIDs), len(perMappingOpts))
	}

	result := jsonData
	for i, id := range orderedIDs {
		var err error
		result, err = m.ApplyQueryMappings(id, perMappingOpts[i], result)
		if err != nil {
			return nil, fmt.Errorf("cascade step %d (mapping %q): %w", i, id, err)
		}
	}
	return result, nil
}

// CascadeResponseMappings applies multiple mapping lists sequentially
// to a response object, feeding the output of each into the next.
// orderedIDs and perMappingOpts must have the same length. An empty
// list returns jsonData unchanged.
func (m *Mapper) CascadeResponseMappings(orderedIDs []string, perMappingOpts []MappingOptions, jsonData any) (any, error) {
	if len(orderedIDs) != len(perMappingOpts) {
		return nil, fmt.Errorf("orderedIDs length (%d) must match perMappingOpts length (%d)", len(orderedIDs), len(perMappingOpts))
	}

	result := jsonData
	for i, id := range orderedIDs {
		var err error
		result, err = m.ApplyResponseMappings(id, perMappingOpts[i], result)
		if err != nil {
			return nil, fmt.Errorf("cascade step %d (mapping %q): %w", i, id, err)
		}
	}
	return result, nil
}
