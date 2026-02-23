# Mapping File Reference

This document describes the syntax and guidelines for writing mapping files for Koral-Mapper. For general project information, installation, and API documentation, see [README.md](README.md).

## Mapping File Format

A mapping file defines a single mapping list with an ID, optional foundry/layer defaults, and a list of mapping rules:

```yaml
id: mapping-list-id
foundryA: source-foundry
layerA: source-layer
foundryB: target-foundry
layerB: target-layer
mappings:
  - "[pattern1] <> [replacement1]"
  - "[pattern2] <> [replacement2]"
```

Mapping files can also be embedded inside a main configuration file under the `lists:` key (see [README.md](README.md) for configuration file format).

Koral-Mapper supports two mapping types: **annotation** (the default) and **corpus**.

## Annotation Mapping Rules (type: annotation)

Annotation mapping rules rewrite `koral:token` / `koral:term` / `koral:termGroup` structures in query JSON and annotation spans in response snippets.

Each rule consists of two patterns separated by `<>`. The patterns can be:
- Simple terms: `[key]`, `[layer=key]`, `[foundry/*=key]`, `[foundry/layer=key]`, or `[foundry/layer=key:value]`
- Complex terms with AND/OR relations: `[term1 & term2]`, `[term1 | term2]`, or `[term1 | (term2 & term3)]`

Example mapping file:

```yaml
id: stts-upos
desc: Mapping from STTS and Universal dependency Part-of-Speech
foundryA: opennlp
layerA: p
foundryB: upos
layerB: p
mappings:
  - "[ADJA] <> [ADJ]"
  - "[ADJD] <> [ADJ & Variant=Short]"
  - "[ART] <> [DET & PronType=Art]"
  - "[PIAT] <> [DET & (PronType=Ind | PronType=Neg | PronType=Tot)]"
```

### Foundry and Layer Precedence

Koral-Mapper follows a strict precedence hierarchy when determining which foundry and layer values to use during mapping transformations:

1. **Mapping rule foundry/layer** (highest priority)
   - Explicit foundry/layer specifications in mapping rule patterns
   - Example: `[opennlp/p=DT]` has explicit foundry "opennlp" and layer "p"

2. **Passed overwrite foundry/layer** (medium priority)
   - Values provided via query parameters (`foundryA`, `foundryB`, `layerA`, `layerB`)
   - Applied only when mapping rules don't have explicit foundry/layer

3. **Mapping list foundry/layer** (lowest priority)
   - Default values from the mapping list configuration
   - Used as fallback when neither mapping rules nor query parameters specify values

## Corpus Mapping Rules (type: corpus)

Corpus mapping rules use `key=value <> key=value` syntax for rewriting `koral:doc` / `koral:docGroup` structures in the `corpus`/`collection` section of a KoralQuery request, and enriching `fields` arrays in responses.

### Rule Syntax

#### Simple fields

```yaml
mappings:
  - "textClass=novel <> genre=fiction"
```

The left side is "side A" and the right side is "side B". With `dir=atob`, the query matcher rewrites A-side matches to B-side replacements. With `dir=btoa`, the direction is reversed.

#### Match types and value types

Rules can specify match operators and value types:

```yaml
mappings:
  - "pubDate=2020:geq <> yearFrom=2020:geq"            # match type (eq, ne, geq, leq, contains, excludes)
  - "pubDate=2020-01#date <> year=2020#string"           # value type (string, regex, date)
  - "textClass=wissenschaft.*#regex <> genre=science"    # regex matching
```

When a rule specifies a match type (e.g. `:geq`), it only matches nodes with that exact match type. When no match type is specified, the rule matches any match type and preserves the original.

#### Group rules (AND / OR)

Rules can use AND (`&`) and OR (`|`) groups on either side:

```yaml
mappings:
  # Single field → AND group
  - "textClass=novel <> (genre=fiction & type=book)"
  # AND group → single field (matches AND docGroups via subset matching)
  - "genre=fiction <> (textClass=kultur & textClass=musik)"
  # OR group → single field (matches individual docs or OR docGroups)
  - "(genre=fiction | genre=novel) <> textClass=belletristik"
  # Complex: OR-of-AND on B-side
  - "Entertainment <> ((kultur & musik) | (kultur & film))"
```

Key points for groups:
- **AND patterns** match any AND group containing at least the pattern's operands (subset). Extra operands are preserved.
- **OR patterns** match a single leaf if any operand matches, or an OR group structurally (exact operand count).
- Groups on **both sides** are supported.

#### Bare values with `fieldA` / `fieldB`

When `fieldA` / `fieldB` are set in the mapping list header, values without a `key=` prefix are shorthand. The field name is filled in from the header:

```yaml
id: satek-wiki-dereko
type: corpus
fieldA: wikiCat
fieldB: textClass
mappings:
  # Equivalent to "wikiCat=Entertainment <> textClass=kultur"
  - "Entertainment <> kultur"
  # Groups work too
  - "Entertainment <> (kultur & musik)"
```

### Matching Semantics

#### Query rewriting — iterative rule application

Corpus rules are applied **iteratively**: each rule is applied to the **entire tree** in order, and subsequent rules see the **transformed result** of all previous rules. This means multiple rules can transform successive AST states, just like the annotation matcher.

For each rule, the matcher tries matching at the current node first. If no match is found and the node is a `koral:docGroup` / `koral:fieldGroup`, the rule recurses into operands.

#### OR pattern matching

OR patterns like `(a | b)` match in two ways:

- **Leaf nodes** (`koral:doc` / `koral:field`): An OR pattern matches if **any operand** matches the leaf. For example, the pattern `(Entertainment | Culture)` matches a single `koral:doc` with value `Entertainment`.
- **Group nodes** (`koral:docGroup` / `koral:fieldGroup`): Structural matching — the node must be an OR group with **exactly** the same operands (commutative, exact count).

#### AND pattern matching (subset)

AND patterns like `(a & b)` use **subset matching**: the node must be an AND `koral:docGroup` / `koral:fieldGroup` containing **at least** all pattern operands. Extra operands beyond the pattern are preserved alongside the replacement.

For example, if the rule is `genre=fiction <> (textClass=kultur & textClass=musik)` and the input is `AND(textClass=kultur, textClass=musik, pubDate=2020)`, the AND pattern matches (subset of 3 operands), and the result is `AND(genre=fiction, pubDate=2020)` — the replacement plus the preserved extra operand.

If all operands match (no extras), the group is replaced entirely by the replacement node.

#### Response enrichment

For response field enrichment, the matching rules work as follows:

- **Pattern matching**: Field patterns match directly. OR group patterns match a single response field if **any operand** matches. AND group patterns **cannot** match a single field and are skipped.
- **Replacement collection**: AND group replacements are **flattened** — all operands become individual `koral:field` entries. OR group replacements are **skipped** because response fields are flat key/value entries and OR semantics (one-of) cannot be represented.

Examples:
- `(a | b) <> (c & d)` — when field `a` is in the response, both `c` and `d` are added.
- `(a | b) <> (c | d)` — when field `a` is in the response, nothing is added (OR replacement skipped).
- `a <> (c & d)` — when field `a` is in the response, both `c` and `d` are added.
- `a <> c` — when field `a` is in the response, `c` is added.

(Supported `@type` aliases: `koral:field` for `koral:doc`, `koral:fieldGroup` for `koral:docGroup`).

### Rule Ordering Strategy

Rules should be ordered from **most specific to most general** (by total leaf count across both sides, descending). Because rules are applied iteratively, more specific rules should appear first to transform the AST before more general rules get a chance to match. Generated mapping files typically contain complementary rule types such as:

1. **Aggregated rules** with OR-of-AND groups — match exact complex structures
2. **Individual group rules** with AND patterns — match individual `koral:docGroup` nodes (subset matching)

### Iterative Application and Rule Chaining

Because rules are applied iteratively (each rule sees the result of previous rules), you can chain transformations:

```yaml
mappings:
  - "textClass=novel <> genre=fiction"
  - "genre=fiction <> category=lit"
```

With this configuration and `dir=atob`, an input `textClass=novel` is first rewritten to `genre=fiction` by rule 1, then to `category=lit` by rule 2.

This also means that for bidirectional mappings, you often need complementary rules that handle decomposed groups:

```yaml
mappings:
  # Forward: source category → OR-of-AND target categories (for AtoB)
  - "Entertainment <> ((kultur & musik) | (kultur & film))"
  # Reverse AND: multiple source categories ← AND group (for BtoA with AND input)
  - "(Entertainment | Culture) <> (kultur & film)"
```
