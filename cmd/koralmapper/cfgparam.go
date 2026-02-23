package main

import (
	"fmt"
	"strings"

	"github.com/KorAP/Koral-Mapper/config"
)

// CascadeEntry represents a single mapping configuration parsed from
// the cfg URL parameter. After parsing, empty override fields are
// merged with the YAML defaults from the corresponding MappingList.
type CascadeEntry struct {
	ID        string
	Direction string
	FoundryA  string
	LayerA    string
	FoundryB  string
	LayerB    string
	FieldA    string
	FieldB    string
}

// ParseCfgParam parses the compact cfg URL parameter into a slice of
// CascadeEntry structs. Empty override fields are merged with YAML
// defaults from the matching MappingList.
//
// Format: entry (";" entry)*
//
//	entry = id ":" dir [ ":" foundryA ":" layerA ":" foundryB ":" layerB ]
//	      | id ":" dir [ ":" fieldA ":" fieldB ]
//
// Annotation entries have either 2 fields (all foundry/layer use defaults)
// or 6 fields (explicit values, empty means use default).
// Corpus entries have either 2 fields (all field overrides use defaults)
// or 4 fields (explicit values, empty means use default).
func ParseCfgParam(raw string, lists []config.MappingList) ([]CascadeEntry, error) {
	if raw == "" {
		return nil, nil
	}

	listsByID := make(map[string]*config.MappingList, len(lists))
	for i := range lists {
		listsByID[lists[i].ID] = &lists[i]
	}

	parts := strings.Split(raw, ";")
	result := make([]CascadeEntry, 0, len(parts))

	for _, part := range parts {
		fields := strings.Split(part, ":")
		n := len(fields)
		if n < 2 {
			return nil, fmt.Errorf("invalid entry %q: expected at least 2 colon-separated fields, got %d", part, n)
		}

		id := fields[0]
		dir := fields[1]

		if dir != "atob" && dir != "btoa" {
			return nil, fmt.Errorf("invalid direction %q in entry %q", dir, part)
		}

		list, ok := listsByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown mapping ID %q", id)
		}
		isCorpus := list.IsCorpus()

		if isCorpus {
			if n != 2 && n != 4 {
				return nil, fmt.Errorf("invalid corpus entry %q: expected 2 or 4 colon-separated fields, got %d", part, n)
			}
		} else if n != 2 && n != 6 {
			return nil, fmt.Errorf("invalid annotation entry %q: expected 2 or 6 colon-separated fields, got %d", part, n)
		}

		ce := CascadeEntry{
			ID:        id,
			Direction: dir,
		}

		if isCorpus {
			if n == 4 {
				ce.FieldA = fields[2]
				ce.FieldB = fields[3]
			}
			if ce.FieldA == "" {
				ce.FieldA = list.FieldA
			}
			if ce.FieldB == "" {
				ce.FieldB = list.FieldB
			}
		} else {
			if n == 6 {
				ce.FoundryA = fields[2]
				ce.LayerA = fields[3]
				ce.FoundryB = fields[4]
				ce.LayerB = fields[5]
			}

			if ce.FoundryA == "" {
				ce.FoundryA = list.FoundryA
			}
			if ce.LayerA == "" {
				ce.LayerA = list.LayerA
			}
			if ce.FoundryB == "" {
				ce.FoundryB = list.FoundryB
			}
			if ce.LayerB == "" {
				ce.LayerB = list.LayerB
			}
		}

		result = append(result, ce)
	}

	return result, nil
}

// BuildCfgParam serialises a slice of CascadeEntry back to the compact
// cfg string format. Entries with all override fields empty use the
// short 2-field format (id:dir). Entries with any non-empty
// foundry/layer field use the full 6-field annotation format.
// Entries with any non-empty fieldA/fieldB use the full 4-field
// corpus format.
func BuildCfgParam(entries []CascadeEntry) string {
	if len(entries) == 0 {
		return ""
	}

	parts := make([]string, len(entries))
	for i, e := range entries {
		if e.FoundryA == "" && e.LayerA == "" && e.FoundryB == "" && e.LayerB == "" && e.FieldA == "" && e.FieldB == "" {
			parts[i] = e.ID + ":" + e.Direction
		} else if e.FoundryA == "" && e.LayerA == "" && e.FoundryB == "" && e.LayerB == "" {
			parts[i] = e.ID + ":" + e.Direction + ":" + e.FieldA + ":" + e.FieldB
		} else {
			parts[i] = e.ID + ":" + e.Direction + ":" + e.FoundryA + ":" + e.LayerA + ":" + e.FoundryB + ":" + e.LayerB
		}
	}

	return strings.Join(parts, ";")
}
