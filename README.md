# KoralPipe-TermMapper

A KorAP service using the KoralPipe mechanism to rewrite terms in queries and responses between different annotations.

## Overview

KoralPipe-TermMapper is a tool for transforming linguistic annotations between different annotation schemes. It allows you to define mapping rules in YAML configuration files and apply these mappings to JSON-encoded linguistic annotations.

## Installation

```bash
go get github.com/KorAP/KoralPipe-TermMapper
```

## Usage

```bash
termmapper -c config.yaml -p 8080 -l info
```
Command line options:
- `--config` or `-c`: YAML configuration file containing mapping directives (required)
- `--port` or `-p`: Port to listen on (default: 8080)
- `--log-level` or `-l`: Log level (debug, info, warn, error) (default: info)
- `--help` or `-h`: Show help message

## Configuration File Format

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

## API Endpoints

### POST /:map/query

Transform a JSON object using the specified mapping list.

Parameters:

- `:map`: ID of the mapping list to use
- `dir` (query): Direction of transformation (atob or `btoa`, default: `atob`)
- `foundryA` (query): Override default foundryA from mapping list
- `foundryB` (query): Override default foundryB from mapping list
- `layerA` (query): Override default layerA from mapping list
- `layerB` (query): Override default layerB from mapping list

Request body: JSON object to transform

Example request:

```http
POST /opennlp-mapper/query?dir=atob&foundryB=custom HTTP/1.1
Content-Type: application/json

{
  "@type": "koral:token",
  "wrap": {
    "@type": "koral:term",
    "foundry": "opennlp",
    "key": "PIDAT",
    "layer": "p",
    "match": "match:eq"
  }
}
```

Example response:

```json
{
  "@type": "koral:token",
  "wrap": {
    "@type": "koral:termGroup",
    "operands": [
      {
        "@type": "koral:term",
        "foundry": "custom",
        "key": "PIDAT",
        "layer": "p",
        "match": "match:eq"
      },
      {
        "@type": "koral:term",
        "foundry": "custom",
        "key": "AdjType",
        "layer": "p",
        "match": "match:eq",
        "value": "Pdt"
      }
    ],
    "relation": "relation:and"
  }
}
```

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

TermMapper is free software published under the
[BSD-2 License](https://opensource.org/licenses/BSD-2-Clause).

*Disclaimer*: This software was developed (as an experiment) with major assistance by AI (mainly Claude 3.5-sonnet and Claude 4-sonnet).
