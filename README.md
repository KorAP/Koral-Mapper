# KoralPipe-TermMapper

A KorAP service using the KoralPipe mechanism to rewrite terms in queries and responses between different annotations.

## Overview

KoralPipe-TermMapper is a tool for transforming linguistic annotations between different annotation schemes. It allows you to define mapping rules in YAML configuration files and apply these mappings to JSON-encoded linguistic annotations.

## Features

- Define mapping rules in YAML configuration files
- Support for bidirectional mappings
- Override foundry and layer values at runtime
- Handle complex term patterns with AND/OR relations

## Installation

```bash
go get github.com/KorAP/KoralPipe-TermMapper
```

## Configuration Format

Mapping rules are defined in YAML files with the following structure:

```yaml
- id: mapping-list-id
  foundryA: source-foundry
  layerA: source-layer
  foundryB: target-foundry
  layerB: target-layer
  mappings:
    - "[pattern1] <> [replacement1]"
    - "[pattern2] <> [replacement2]"
```

Each mapping rule consists of two patterns separated by `<>`. The patterns can be:
- Simple terms: `[key]` or `[foundry/layer=key]` or `[foundry/layer=key:value]`
- Complex terms with AND/OR relations: `[term1 & term2]` or `[term1 | term2]` or `[term1 | (term2 & term3)]`

## Progress

- [x] Mapping functionality
- [x] Support for rewrites
- [x] Web service
- [ ] Support for negation
- [ ] JSON script for Kalamar integration
- [ ] Response rewriting
- [ ] Integration of mapping files

## COPYRIGHT AND LICENSE

Copyright (C) 2025, [IDS Mannheim](https://www.ids-mannheim.de/)<br>
Author: [Nils Diewald](https://www.nils-diewald.de/)

Disclaimer: This software was developed (as an experiment) with major assistance by AI (mainly Claude 3.5-sonnet and Claude 4-sonnet).
