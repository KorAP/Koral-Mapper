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
termmapper -c config.yaml -m extra-mapper1.yaml -m extra-mapper2.yaml
```

Command Line Options

- `--config` or `-c`: YAML configuration file containing mapping directives and global settings (optional)
- `--mappings` or `-m`: Individual YAML mapping files to load (can be used multiple times, optional)
- `--port` or `-p`: Port to listen on (overrides config file, defaults to 3000 if not specified)
- `--log-level` or `-l`: Log level (debug, info, warn, error) (overrides config file, defaults to warn if not specified)
- `--help` or `-h`: Show help message

**Note**: At least one mapping source must be provided

## Configuration

KoralPipe-TermMapper supports loading configuration from multiple sources:

1. **Main Configuration File** (`-c`): Contains global settings (SDK, server endpoints, port, log level) and optional mapping lists
2. **Individual Mapping Files** (`-m`): Contains single mapping lists, can be specified multiple times

The main configuration provides global settings, and all mapping lists from both sources are combined. Duplicate mapping IDs across all sources will result in an error.

### Configuration File Format

Configurations can contain global settings and mapping lists (used with the `-c` flag):

```yaml
# Optional: Custom SDK endpoint for Kalamar plugin integration
sdk: "https://custom.example.com/js/korap-plugin.js"

# Optional: Custom server endpoint for Kalamar plugin integration  
server: "https://custom.example.com/"

# Optional: Port to listen on (default: 5725)
port: 8080

# Optional: Log level - debug, info, warn, error (default: warn)
loglevel: info

# Optional: ServiceURL for the termmapper
serviceURL: "https://korap.ids-mannheim.de/plugin/termmapper"

# Optional: Mapping lists (same format as individual mapping files)
lists:
  - id: mapping-list-id
    foundryA: source-foundry
    layerA: source-layer
    foundryB: target-foundry
    layerB: target-layer
    mappings:
      - "[pattern1] <> [replacement1]"
      - "[pattern2] <> [replacement2]"
```

Map files contain a single mapping list (used with the `-m` flag):

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

Command line arguments take precedence over configuration file values:

The `sdk`, `server`, `port`, and `loglevel` fields in the main configuration file are optional and override the following default values:

- **`sdk`**: Custom SDK JavaScript file URL (default: `https://korap.ids-mannheim.de/js/korap-plugin-latest.js`)
- **`server`**: Custom server endpoint URL (default: `https://korap.ids-mannheim.de/`)
- **`port`**: Server port (default: `5725`)
- **`loglevel`**: Log level (default: `warn`)
- **`serviceURL`**: Service URL of the TermMapper (default: `https://korap.ids-mannheim.de/plugin/termmapper`)

These values are applied during configuration parsing. When using only individual mapping files (`-m` flags), default values are used unless overridden by command line arguments.

### Mapping Rules

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

### POST /:map/response

Transform JSON response objects using the specified mapping list. This endpoint processes response snippets by applying term mappings to annotations within HTML snippet markup.

Parameters:

- `:map`: ID of the mapping list to use
- `dir` (query): Direction of transformation (atob or `btoa`, default: `atob`)
- `foundryA` (query): Override default foundryA from mapping list
- `foundryB` (query): Override default foundryB from mapping list
- `layerA` (query): Override default layerA from mapping list
- `layerB` (query): Override default layerB from mapping list

Request body: JSON object containing a `snippet` field with HTML markup

Example request:

```http
POST /opennlp-mapper/response?dir=atob&foundryB=custom HTTP/1.1
Content-Type: application/json

{
  "snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
}
```

Example response:

```json
{
  "snippet": "<span title=\"marmot/m:gender:masc\"><span title=\"custom/p:M\" class=\"notinindex\"><span title=\"custom/m:M\" class=\"notinindex\">Der</span></span></span>"
}
```

### GET /

Serves the Kalamar plugin integration page. This HTML page includes:

- Plugin information and available mapping lists
- JavaScript integration code for Kalamar
- SDK and server endpoints configured via `sdk` and `server` configuration fields

The SDK script and server data-attribute in the HTML are determined by the configuration file's `sdk` and `server` values, with fallback to default endpoints if not specified.

## Supported mappings

### `mappings/stts-upos.yaml`

Mapping between STTS and UD part-of-spech tags.

## Progress

- [x] Mapping functionality
- [x] Support for rewrites
- [x] Web service
- [x] JSON script for Kalamar integration
- [x] Integration of multiple mapping files
- [ ] Support for negation
- [ ] Support multiple mappings (by having a check list)
- [ ] Response rewriting
- [ ] Support corpus mappings
- [ ] Support chaining of mappings

## COPYRIGHT AND LICENSE

Copyright (C) 2025, [IDS Mannheim](https://www.ids-mannheim.de/)<br>
Author: [Nils Diewald](https://www.nils-diewald.de/)

TermMapper is free software published under the
[BSD-2 License](https://opensource.org/licenses/BSD-2-Clause).

*Disclaimer*: This software was developed (as an experiment) with major assistance by AI (mainly Claude 3.5-sonnet and Claude 4-sonnet).
