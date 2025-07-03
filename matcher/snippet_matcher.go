package matcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KorAP/Koral-Mapper/ast"
	"github.com/KorAP/Koral-Mapper/parser"
	"github.com/orisano/gosax"
)

// TokenSpan represents a token and its position in the snippet
type TokenSpan struct {
	Text        string   // The actual token text
	StartPos    int      // Character position where the token starts
	EndPos      int      // Character position where the token ends
	Annotations []string // All title attributes that annotate this token
}

// SnippetMatcher extends the basic matcher to work with HTML/XML snippets
type SnippetMatcher struct {
	matcher     *Matcher
	titleParser *parser.TitleAttributeParser
}

// NewSnippetMatcher creates a new snippet matcher
func NewSnippetMatcher(pattern ast.Pattern, replacement ast.Replacement) (*SnippetMatcher, error) {
	matcher, err := NewMatcher(pattern, replacement)
	if err != nil {
		return nil, fmt.Errorf("failed to create base matcher: %w", err)
	}

	return &SnippetMatcher{
		matcher:     matcher,
		titleParser: parser.NewTitleAttributeParser(),
	}, nil
}

// ParseSnippet parses an HTML/XML snippet and extracts tokens with their annotations
func (sm *SnippetMatcher) ParseSnippet(snippet string) ([]TokenSpan, error) {
	tokens := make([]TokenSpan, 0)

	// Stack to track nested spans and their annotations
	type spanInfo struct {
		title string
		level int
	}
	spanStack := make([]spanInfo, 0)

	// Current position tracking
	var currentPos int

	reader := strings.NewReader(snippet)
	r := gosax.NewReader(reader)

	for {
		e, err := r.Event()
		if err != nil {
			return nil, fmt.Errorf("failed to parse snippet: %w", err)
		}

		if e.Type() == 8 { // gosax.EventEOF
			break
		}

		switch e.Type() {
		case 1: // gosax.EventStart
			// Parse start element
			startElem, err := gosax.StartElement(e.Bytes)
			if err != nil {
				continue // Skip invalid elements
			}

			if startElem.Name.Local == "span" {
				// Look for title attribute
				var title string
				for _, attr := range startElem.Attr {
					if attr.Name.Local == "title" {
						title = attr.Value
						break
					}
				}
				spanStack = append(spanStack, spanInfo{title: title, level: len(spanStack)})
			}

		case 2: // gosax.EventEnd
			// Parse end element
			endElem := gosax.EndElement(e.Bytes)
			if endElem.Name.Local == "span" && len(spanStack) > 0 {
				spanStack = spanStack[:len(spanStack)-1]
			}

		case 3: // gosax.EventText
			// Process character data
			charData, err := gosax.CharData(e.Bytes)
			if err != nil {
				continue
			}

			text := string(charData)
			trimmed := strings.TrimSpace(text)
			if trimmed != "" && len(spanStack) > 0 {
				// Only create tokens if we're inside at least one span
				// Collect all annotations from the current span stack
				annotations := make([]string, 0)
				for _, span := range spanStack {
					if span.title != "" {
						annotations = append(annotations, span.title)
					}
				}

				// Create token span
				token := TokenSpan{
					Text:        trimmed,
					StartPos:    currentPos,
					EndPos:      currentPos + len(trimmed),
					Annotations: annotations,
				}
				tokens = append(tokens, token)
			}
			currentPos += len(text)
		}
	}

	// Sort tokens by start position to ensure proper order
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].StartPos < tokens[j].StartPos
	})

	return tokens, nil
}

// CheckToken checks if a token's annotations match the pattern
func (sm *SnippetMatcher) CheckToken(token TokenSpan) (bool, error) {
	if len(token.Annotations) == 0 {
		return false, nil
	}

	// Parse all annotations into AST terms
	terms, err := sm.titleParser.ParseTitleAttributesToTerms(token.Annotations)
	if err != nil {
		return false, fmt.Errorf("failed to parse token annotations: %w", err)
	}

	if len(terms) == 0 {
		return false, nil
	}

	// Create a TermGroup with AND relation for all annotations
	var nodeToMatch ast.Node
	if len(terms) == 1 {
		nodeToMatch = terms[0]
	} else {
		nodeToMatch = &ast.TermGroup{
			Operands: terms,
			Relation: ast.AndRelation,
		}
	}

	// Check if the constructed node matches our pattern
	return sm.matcher.Match(nodeToMatch), nil
}

// FindMatchingTokens finds all tokens in the snippet that match the pattern
func (sm *SnippetMatcher) FindMatchingTokens(snippet string) ([]TokenSpan, error) {
	tokens, err := sm.ParseSnippet(snippet)
	if err != nil {
		return nil, err
	}

	matchingTokens := make([]TokenSpan, 0)
	for _, token := range tokens {
		if matches, err := sm.CheckToken(token); err != nil {
			return nil, fmt.Errorf("failed to check token '%s': %w", token.Text, err)
		} else if matches {
			matchingTokens = append(matchingTokens, token)
		}
	}

	return matchingTokens, nil
}
