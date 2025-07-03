package parser

import (
	"fmt"
	"regexp"

	"github.com/KorAP/Koral-Mapper/ast"
)

// TitleAttribute represents a parsed title attribute from an HTML span
type TitleAttribute struct {
	Foundry string
	Layer   string
	Key     string
	Value   string
}

// TitleAttributeParser parses title attributes from HTML span elements
type TitleAttributeParser struct {
	regex *regexp.Regexp
}

// NewTitleAttributeParser creates a new title attribute parser
func NewTitleAttributeParser() *TitleAttributeParser {
	// Single regex that captures: foundry/layer:key or foundry/layer:key[:=]value
	// Groups: 1=foundry, 2=layer, 3=key, 4=value (optional)
	regex := regexp.MustCompile(`^([^/]+)/([^:]+):([^:]+)(?::(.+))?$`)
	return &TitleAttributeParser{
		regex: regex,
	}
}

// parseTitleAttribute parses a single title attribute string
// Expects format: "foundry/layer:key" or "foundry/layer:key[:=]value"
func (p *TitleAttributeParser) parseTitleAttribute(title string) (*TitleAttribute, error) {
	if title == "" {
		return nil, fmt.Errorf("empty title attribute")
	}

	matches := p.regex.FindStringSubmatch(title)
	if matches == nil {
		return nil, fmt.Errorf("invalid title format: '%s'", title)
	}

	foundry := matches[1]
	layer := matches[2]
	key := matches[3]
	value := ""
	if len(matches) > 4 && matches[4] != "" {
		value = matches[4]
	}

	return &TitleAttribute{
		Foundry: foundry,
		Layer:   layer,
		Key:     key,
		Value:   value,
	}, nil
}

// ParseTitleAttributesToTerms converts title attributes to AST Term nodes
func (p *TitleAttributeParser) ParseTitleAttributesToTerms(titles []string) ([]ast.Node, error) {
	terms := make([]ast.Node, 0) // Initialize as empty slice instead of nil

	for _, title := range titles {
		attr, err := p.parseTitleAttribute(title)
		if err != nil {
			return nil, fmt.Errorf("failed to parse title '%s': %w", title, err)
		}

		term := &ast.Term{
			Foundry: attr.Foundry,
			Layer:   attr.Layer,
			Key:     attr.Key,
			Value:   attr.Value,
			Match:   ast.MatchEqual,
		}

		terms = append(terms, term)
	}

	return terms, nil
}
