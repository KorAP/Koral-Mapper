package matcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/KorAP/KoralPipe-TermMapper/parser"
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

// CheckTokenSequence checks if a sequence of tokens matches the pattern
func (sm *SnippetMatcher) CheckTokenSequence(tokens []TokenSpan) (bool, error) {
	if len(tokens) == 0 {
		return false, nil
	}

	// For token sequences, we need to check different strategies:
	// 1. Check if any individual token matches
	// 2. Check if the combined annotations of all tokens match

	// Strategy 1: Check individual tokens
	for _, token := range tokens {
		matches, err := sm.CheckToken(token)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}

	// Strategy 2: Check combined annotations
	allAnnotations := make([]string, 0)
	for _, token := range tokens {
		allAnnotations = append(allAnnotations, token.Annotations...)
	}

	// Remove duplicates from combined annotations
	annotationMap := make(map[string]bool)
	uniqueAnnotations := make([]string, 0)
	for _, annotation := range allAnnotations {
		if !annotationMap[annotation] {
			annotationMap[annotation] = true
			uniqueAnnotations = append(uniqueAnnotations, annotation)
		}
	}

	if len(uniqueAnnotations) == 0 {
		return false, nil
	}

	// Create a combined token for checking
	combinedToken := TokenSpan{
		Text:        strings.Join(getTokenTexts(tokens), " "),
		StartPos:    tokens[0].StartPos,
		EndPos:      tokens[len(tokens)-1].EndPos,
		Annotations: uniqueAnnotations,
	}

	return sm.CheckToken(combinedToken)
}

// FindMatchingTokens finds all tokens in the snippet that match the pattern
func (sm *SnippetMatcher) FindMatchingTokens(snippet string) ([]TokenSpan, error) {
	tokens, err := sm.ParseSnippet(snippet)
	if err != nil {
		return nil, err
	}

	matchingTokens := make([]TokenSpan, 0)

	for _, token := range tokens {
		matches, err := sm.CheckToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to check token '%s': %w", token.Text, err)
		}
		if matches {
			matchingTokens = append(matchingTokens, token)
		}
	}

	return matchingTokens, nil
}

// FindMatchingTokenSequences finds all token sequences that match the pattern
func (sm *SnippetMatcher) FindMatchingTokenSequences(snippet string, maxSequenceLength int) ([][]TokenSpan, error) {
	tokens, err := sm.ParseSnippet(snippet)
	if err != nil {
		return nil, err
	}

	if maxSequenceLength <= 0 {
		maxSequenceLength = len(tokens)
	}

	matchingSequences := make([][]TokenSpan, 0)

	// Check all possible token sequences up to maxSequenceLength
	for start := 0; start < len(tokens); start++ {
		for length := 1; length <= maxSequenceLength && start+length <= len(tokens); length++ {
			sequence := tokens[start : start+length]

			matches, err := sm.CheckTokenSequence(sequence)
			if err != nil {
				return nil, fmt.Errorf("failed to check token sequence: %w", err)
			}
			if matches {
				matchingSequences = append(matchingSequences, sequence)
			}
		}
	}

	return matchingSequences, nil
}

// GetReplacement returns the replacement node from the matcher
func (sm *SnippetMatcher) GetReplacement() ast.Node {
	return sm.matcher.replacement.Root
}

// Helper function to extract token texts
func getTokenTexts(tokens []TokenSpan) []string {
	texts := make([]string, len(tokens))
	for i, token := range tokens {
		texts[i] = token.Text
	}
	return texts
}
