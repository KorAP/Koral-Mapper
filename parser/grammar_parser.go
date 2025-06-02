package parser

import (
	"fmt"
	"strings"

	"github.com/KorAP/KoralPipe-TermMapper/ast"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// GrammarParser parses a simple grammar into AST nodes
type GrammarParser struct {
	defaultFoundry string
	defaultLayer   string
	tokenParser    *participle.Parser[TokenGrammar]
	mappingParser  *participle.Parser[MappingGrammar]
}

// TokenGrammar represents a single token expression
type TokenGrammar struct {
	Token *TokenExpr `parser:"@@"`
}

// MappingGrammar represents a mapping rule
type MappingGrammar struct {
	Mapping *MappingRule `parser:"@@"`
}

// MappingRule represents a mapping between two token expressions
type MappingRule struct {
	Upper *TokenExpr `parser:"@@"`
	Lower *TokenExpr `parser:"'<>' @@"`
}

// TokenExpr represents a token expression in square brackets
type TokenExpr struct {
	Expr *Expr `parser:"'[' @@ ']'"`
}

// Expr represents a sequence of terms and operators
type Expr struct {
	First *Term `parser:"@@"`
	Rest  []*Op `parser:"@@*"`
}

type Op struct {
	Operator string `parser:"@('&' | '|')"`
	Term     *Term  `parser:"@@"`
}

// Term represents either a simple term or a parenthesized expression
type Term struct {
	Simple *SimpleTerm `parser:"@@"`
	Paren  *ParenExpr  `parser:"| @@"`
}

type ParenExpr struct {
	Expr *Expr `parser:"'(' @@ ')'"`
}

// SimpleTerm represents any valid term form
type SimpleTerm struct {
	WithFoundryLayer *FoundryLayerTerm `parser:"@@"`
	WithFoundryKey   *FoundryKeyTerm   `parser:"| @@"`
	WithLayer        *LayerTerm        `parser:"| @@"`
	SimpleKey        *KeyTerm          `parser:"| @@"`
}

// FoundryLayerTerm represents foundry/layer=key:value
type FoundryLayerTerm struct {
	Foundry string `parser:"@Ident '/'"`
	Layer   string `parser:"@Ident '='"`
	Key     string `parser:"@Ident"`
	Value   string `parser:"(':' @Ident)?"`
}

// FoundryKeyTerm represents foundry/key
type FoundryKeyTerm struct {
	Foundry string `parser:"@Ident '/'"`
	Key     string `parser:"@Ident"`
}

// LayerTerm represents layer=key:value
type LayerTerm struct {
	Layer string `parser:"@Ident '='"`
	Key   string `parser:"@Ident"`
	Value string `parser:"(':' @Ident)?"`
}

// KeyTerm represents key:value
type KeyTerm struct {
	Key   string `parser:"@Ident"`
	Value string `parser:"(':' @Ident)?"`
}

// NewGrammarParser creates a new grammar parser with optional default foundry and layer
func NewGrammarParser(defaultFoundry, defaultLayer string) (*GrammarParser, error) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Ident", Pattern: `(?:[a-zA-Z$]|\\.)(?:[a-zA-Z0-9_$]|\\.)*`},
		{Name: "Punct", Pattern: `[\[\]()&\|=:/]|<>`},
		{Name: "Whitespace", Pattern: `\s+`},
	})

	tokenParser, err := participle.Build[TokenGrammar](
		participle.Lexer(lex),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build token parser: %w", err)
	}

	mappingParser, err := participle.Build[MappingGrammar](
		participle.Lexer(lex),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build mapping parser: %w", err)
	}

	return &GrammarParser{
		defaultFoundry: defaultFoundry,
		defaultLayer:   defaultLayer,
		tokenParser:    tokenParser,
		mappingParser:  mappingParser,
	}, nil
}

// Parse parses a grammar string into an AST node (for backward compatibility)
func (p *GrammarParser) Parse(input string) (ast.Node, error) {
	// Remove extra spaces around operators to help the parser
	input = strings.ReplaceAll(input, " & ", "&")
	input = strings.ReplaceAll(input, " | ", "|")

	// Add spaces around parentheses to help the parser
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")

	// Remove any extra spaces
	input = strings.TrimSpace(input)

	grammar, err := p.tokenParser.ParseString("", input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grammar: %w", err)
	}

	if grammar.Token == nil {
		return nil, fmt.Errorf("expected token expression, got mapping rule")
	}

	wrap, err := p.parseExpr(grammar.Token.Expr)
	if err != nil {
		return nil, err
	}
	return &ast.Token{Wrap: wrap}, nil
}

// ParseMapping parses a mapping rule string into a MappingResult
func (p *GrammarParser) ParseMapping(input string) (*MappingResult, error) {
	// Remove extra spaces around operators to help the parser
	input = strings.ReplaceAll(input, " & ", "&")
	input = strings.ReplaceAll(input, " | ", "|")
	input = strings.ReplaceAll(input, " <> ", "<>")

	// Add spaces around parentheses to help the parser
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")

	// Remove any extra spaces
	input = strings.TrimSpace(input)

	grammar, err := p.mappingParser.ParseString("", input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grammar: %w", err)
	}

	if grammar.Mapping == nil {
		return nil, fmt.Errorf("expected mapping rule, got token expression")
	}

	upper, err := p.parseExpr(grammar.Mapping.Upper.Expr)
	if err != nil {
		return nil, err
	}

	lower, err := p.parseExpr(grammar.Mapping.Lower.Expr)
	if err != nil {
		return nil, err
	}

	return &MappingResult{
		Upper: &ast.Token{Wrap: upper},
		Lower: &ast.Token{Wrap: lower},
	}, nil
}

// MappingResult represents the parsed mapping rule
type MappingResult struct {
	Upper *ast.Token
	Lower *ast.Token
}

// parseExpr builds the AST from the parsed Expr
func (p *GrammarParser) parseExpr(expr *Expr) (ast.Node, error) {
	var operands []ast.Node
	var operators []string

	// Parse the first term
	first, err := p.parseTerm(expr.First)
	if err != nil {
		return nil, err
	}
	operands = append(operands, first)

	// Parse the rest
	for _, op := range expr.Rest {
		node, err := p.parseTerm(op.Term)
		if err != nil {
			return nil, err
		}
		operands = append(operands, node)
		operators = append(operators, op.Operator)
	}

	// If only one operand, return it
	if len(operands) == 1 {
		return operands[0], nil
	}

	// Group operands by operator precedence (left-to-right, no precedence between & and |)
	// We'll group by runs of the same operator
	var groupOperands []ast.Node
	var currentOp string
	var currentGroup []ast.Node
	for i, op := range operators {
		if i == 0 {
			currentOp = op
			currentGroup = append(currentGroup, operands[i])
		}
		if op == currentOp {
			currentGroup = append(currentGroup, operands[i+1])
		} else {
			groupOperands = append(groupOperands, &ast.TermGroup{
				Operands: append([]ast.Node{}, currentGroup...),
				Relation: toRelation(currentOp),
			})
			currentOp = op
			currentGroup = []ast.Node{operands[i+1]}
		}
	}
	if len(currentGroup) > 0 {
		groupOperands = append(groupOperands, &ast.TermGroup{
			Operands: append([]ast.Node{}, currentGroup...),
			Relation: toRelation(currentOp),
		})
	}
	if len(groupOperands) == 1 {
		return groupOperands[0], nil
	}
	// If mixed operators, nest them left-to-right
	result := groupOperands[0]
	for i := 1; i < len(groupOperands); i++ {
		result = &ast.TermGroup{
			Operands: []ast.Node{result, groupOperands[i]},
			Relation: toRelation(operators[0]),
		}
	}
	return result, nil
}

// parseTerm converts a Term into an AST node
func (p *GrammarParser) parseTerm(term *Term) (ast.Node, error) {
	if term.Simple != nil {
		return p.parseSimpleTerm(term.Simple)
	}
	if term.Paren != nil {
		return p.parseExpr(term.Paren.Expr)
	}
	return nil, fmt.Errorf("invalid term: neither simple nor parenthesized")
}

func toRelation(op string) ast.RelationType {
	if op == "|" {
		return ast.OrRelation
	}
	return ast.AndRelation
}

// unescapeString handles unescaping of backslash-escaped characters
func unescapeString(s string) string {
	if s == "" {
		return s
	}

	result := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			// Escape sequence found, add the escaped character
			result = append(result, s[i+1])
			i += 2
		} else {
			// Regular character
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// parseSimpleTerm converts a SimpleTerm into an AST Term node
func (p *GrammarParser) parseSimpleTerm(term *SimpleTerm) (ast.Node, error) {
	var foundry, layer, key, value string

	switch {
	case term.WithFoundryLayer != nil:
		foundry = unescapeString(term.WithFoundryLayer.Foundry)
		layer = unescapeString(term.WithFoundryLayer.Layer)
		key = unescapeString(term.WithFoundryLayer.Key)
		value = unescapeString(term.WithFoundryLayer.Value)
	case term.WithFoundryKey != nil:
		foundry = unescapeString(term.WithFoundryKey.Foundry)
		key = unescapeString(term.WithFoundryKey.Key)
	case term.WithLayer != nil:
		layer = unescapeString(term.WithLayer.Layer)
		key = unescapeString(term.WithLayer.Key)
		value = unescapeString(term.WithLayer.Value)
	case term.SimpleKey != nil:
		key = unescapeString(term.SimpleKey.Key)
		value = unescapeString(term.SimpleKey.Value)
	default:
		return nil, fmt.Errorf("invalid term: no valid form found")
	}

	if foundry == "" {
		foundry = p.defaultFoundry
	}
	if layer == "" {
		layer = p.defaultLayer
	}

	return &ast.Term{
		Foundry: foundry,
		Key:     key,
		Layer:   layer,
		Match:   ast.MatchEqual,
		Value:   value,
	}, nil
}
