# Koral-Mapper

[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.20340398.svg)](https://doi.org/10.5281/zenodo.20340398)

A KorAP service using the KoralPipe mechanism to rewrite terms in queries and responses between different annotations.

![Koral-Mapper Screenshot](https://raw.githubusercontent.com/KorAP/Koral-Mapper/main/misc/koral-mapper.png)

## Overview

Koral-Mapper is a tool for transforming linguistic annotations between different annotation schemes. It allows you to define mapping rules in YAML configuration files and apply these mappings to JSON-encoded linguistic annotations.

## Installation

The easiest way to install Koral-Mapper for deployment is using [Docker](https://hub.docker.com/r/korap/koral-mapper).

For development purposes or non-docker environments, use

```bash
go get github.com/KorAP/Koral-Mapper
```

## Usage

```bash
koralmapper -c config.yaml -m extra-mapper1.yaml -m extra-mapper2.yaml
```

Command Line Options

- `--config` or `-c`: YAML configuration file containing mapping directives and global settings (optional)
- `--mappings` or `-m`: Individual YAML mapping files to load (can be used multiple times, optional)
- `--port` or `-p`: Port to listen on (overrides config file, defaults to 3000 if not specified)
- `--log-level` or `-l`: Log level (debug, info, warn, error) (overrides config file, defaults to warn if not specified)
- `--help` or `-h`: Show help message

**Note**: At least one mapping source must be provided

## Configuration

Koral-Mapper supports loading configuration from multiple sources:

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

# Optional: Custom Kalamar stylesheet URL for the configuration page
stylesheet: "https://korap.ids-mannheim.de/css/kalamar-plugin-latest.css"

# Optional: Port to listen on (default: 5725)
port: 8080

# Optional: Log level - debug, info, warn, error (default: warn)
loglevel: info

# Optional: ServiceURL for the koralmapper
serviceURL: "https://korap.ids-mannheim.de/plugin/koralmapper"

# Optional: Maximum requests per minute per IP for rate limiting (default: 100)
rateLimit: 100

# Optional: List of allowed CORS origins.
# Defaults to the server value (trailing slash stripped).
# Required when the service is called cross-origin (e.g. as a Kalamar plugin in an iframe).
# Use "*" to allow all origins (not recommended for production).
allowOrigins:
  - "https://korap.ids-mannheim.de"

# Optional: Base path for file loading confinement (default: current working directory).
# All config and mapping file paths must resolve within this directory or /tmp.
# Set to "/" to allow loading from anywhere on the filesystem.
basePath: "/opt/koralmapper"

# Optional: Global default for koral:rewrite annotations (default: false).
# Can be overridden per mapping list and per request via query parameter.
rewrites: false

# Optional: Mapping lists (same format as individual mapping files)
lists:
  - id: mapping-list-id
    foundryA: source-foundry
    layerA: source-layer
    foundryB: target-foundry
    layerB: target-layer
    rewrites: false  # Optional: attach koral:rewrite annotations (default: false)
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
rewrites: false  # Optional: attach koral:rewrite annotations (default: false)
mappings:
  - "[pattern1] <> [replacement1]"
  - "[pattern2] <> [replacement2]"
```

Command line arguments take precedence over configuration file values:

The `sdk`, `stylesheet`, `server`, `port`, `loglevel`, `rewrites`, and `basePath` fields in the main configuration file are optional and override the following default values:

- **`sdk`**: Custom SDK JavaScript file URL (default: `https://korap.ids-mannheim.de/js/korap-plugin-latest.js`)
- **`stylesheet`**: Kalamar stylesheet URL for the config page (default: `https://korap.ids-mannheim.de/css/kalamar-plugin-latest.css`)
- **`server`**: Custom server endpoint URL (default: `https://korap.ids-mannheim.de/`)
- **`port`**: Server port (default: `5725`)
- **`loglevel`**: Log level (default: `warn`)
- **`serviceURL`**: Service URL of the KoralMapper (default: `https://korap.ids-mannheim.de/plugin/koralmapper`)
- **`rateLimit`**: Maximum number of requests per minute per IP address (default: `100`). When the limit is exceeded, the server responds with HTTP 429 (Too Many Requests).
- **`allowOrigins`**: List of origins allowed for CORS (default: derived from `server` with trailing slash removed, e.g. `["https://korap.ids-mannheim.de"]`). Must be specified as a YAML list. The service is designed to be called cross-origin as a Kalamar plugin loaded in iframes. This setting controls which origins may make cross-origin API requests. Allowed methods are `GET` and `POST`. The `Content-Type` header is permitted. Use `["*"]` to allow all origins (not recommended for production).
- **`rewrites`**: Global default for attaching `koral:rewrite` annotations (default: `false`). When `true`, all mapping lists will attach rewrite annotations unless individually overridden. See [Rewrites Resolution](#rewrites-resolution) for the full precedence chain.
- **`basePath`**: Directory tree for file loading confinement (default: current working directory). Configuration and mapping files must resolve within this path or the system temp directory. Set to `"/"` to disable confinement. This prevents path traversal attacks (CWE-22).

These values are applied during configuration parsing. When using only individual mapping files (`-m` flags), default values are used unless overridden by command line arguments.

### Environment Variable Overrides

In addition to YAML config, global settings can be overridden with environment variables.
All variables are optional and use the `KORAL_MAPPER_` prefix:

- `KORAL_MAPPER_SERVER`: Overrides `server`
- `KORAL_MAPPER_SDK`: Overrides `sdk`
- `KORAL_MAPPER_STYLESHEET`: Overrides `stylesheet`
- `KORAL_MAPPER_SERVICE_URL`: Overrides `serviceURL`
- `KORAL_MAPPER_COOKIE_NAME`: Overrides `cookieName`
- `KORAL_MAPPER_LOG_LEVEL`: Overrides `loglevel`
- `KORAL_MAPPER_PORT`: Overrides `port` (integer)
- `KORAL_MAPPER_RATE_LIMIT`: Overrides `rateLimit` (integer, requests per minute per IP)
- `KORAL_MAPPER_ALLOW_ORIGINS`: Overrides `allowOrigins` (comma-separated string of allowed CORS origins, e.g. `https://a.com,https://b.com`)
- `KORAL_MAPPER_REWRITES`: Overrides `rewrites` (`true` or `false`, global default for koral:rewrite annotations)
- `KORAL_MAPPER_BASE_PATH`: Overrides `basePath` (directory path for file loading confinement)

Environment variable values take precedence over values from the configuration file.

### Mapping Rules

Koral-Mapper supports two types of mapping rules:

- **Annotation mappings** (default): Rewrite `koral:token` / `koral:term` structures in queries and annotation spans in responses
- **Corpus mappings** (`type: corpus`): Rewrite `koral:doc` / `koral:docGroup` structures in corpus/collection queries and enrich response fields

For detailed mapping rule syntax, examples, and guidelines on writing mapping files, see [MAPPING.md](MAPPING.md).

## API Endpoints

### POST /query/:cfg

Apply a cascade of query mappings to a JSON object. The `:cfg` path parameter specifies which mapping lists to apply and in what order, using a compact serialization format.

**cfg format:** `id:dir[:foundryA:layerA:foundryB:layerB]` entries separated by `;`

- `id`: ID of the mapping list
- `dir`: Direction (`atob` or `btoa`)
- Optional foundry/layer overrides (annotation mappings use 6 fields, corpus mappings use 4 fields with `fieldA:fieldB`)

When override fields are omitted, defaults from the YAML mapping list are used.

Request body: JSON object to transform

Example request:

```http
POST /query/stts-upos:atob;other-mapper:btoa HTTP/1.1
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

### POST /response/:cfg

Apply a cascade of response mappings to a JSON object. The `:cfg` path parameter uses the same format as `/query/:cfg`.

This endpoint processes response snippets by applying term mappings to annotations within HTML snippet markup, and enriches corpus fields for corpus mappings.

Request body: JSON object (with `snippet` field for annotation mappings, or `fields` for corpus mappings)

Example request:

```http
POST /response/stts-upos:btoa HTTP/1.1
Content-Type: application/json

{
  "snippet": "<span title=\"marmot/m:gender:masc\">Der</span>"
}
```

### POST /:map/query

Transform a JSON object using a single mapping list.

Parameters:

- `:map`: ID of the mapping list to use
- `dir` (query): Direction of transformation (`atob` or `btoa`, default: `atob`)
- `foundryA` (query): Override default foundryA from mapping list
- `foundryB` (query): Override default foundryB from mapping list
- `layerA` (query): Override default layerA from mapping list
- `layerB` (query): Override default layerB from mapping list
- `rewrites` (query): Override the mapping list's `rewrites` setting (`true` or `false`)

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

Transform JSON response objects using a single mapping list. This endpoint processes response snippets by applying term mappings to annotations within HTML snippet markup.

Parameters:

- `:map`: ID of the mapping list to use
- `dir` (query): Direction of transformation (`atob` or `btoa`, default: `atob`)
- `foundryA` (query): Override default foundryA from mapping list
- `foundryB` (query): Override default foundryB from mapping list
- `layerA` (query): Override default layerA from mapping list
- `layerB` (query): Override default layerB from mapping list
- `rewrites` (query): Override the mapping list's `rewrites` setting (`true` or `false`)

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

Serves the configuration page for the Kalamar plugin integration. This HTML page allows selecting mapping lists and configuring their parameters. The JavaScript registers KorAP pipes using the path-based `/query/:cfg` and `/response/:cfg` endpoints.

### GET /:map

Serves the Kalamar plugin integration page for a single mapping list. This HTML page includes:

- Plugin information and available mapping lists
- JavaScript integration code for Kalamar
- SDK and server endpoints configured via `sdk` and `server` configuration fields

The SDK script and server data-attribute in the HTML are determined by the configuration file's `sdk` and `server` values, with fallback to default endpoints if not specified.

### GET /health

Health check endpoint. Returns `OK` with HTTP 200.

## Kalamar Plugin Registration

To register Koral-Mapper as a Kalamar plugin, a JSON manifest must be provided to the Kalamar plugin system. The manifest specifies how the plugin is embedded and what permissions it requires. For example:

```json
{
  "name": "Koral-Mapper",
  "desc": "Rewrites annotation terms in queries and responses",
  "embed": [{
    "panel": "query",
    "title": "Term Mapper Configuration",
    "icon": "\uf074",
    "classes": ["button-icon", "plugin"],
    "onClick": {
      "action": "addWidget",
      "template": "https://korap.ids-mannheim.de/plugin/koralmapper/",
      "permissions": ["scripts", "forms", "same-origin"]
    }
  }]
}
```

Adjust the `template` URL to match your deployment's `serviceURL`.

## Supported Mappings

### `mappings/stts-upos.yaml`

Mapping between [STTS and UD part-of-speech tags](https://universaldependencies.org/tagset-conversion/de-stts-uposf.html).

### `mappings/wiki-dereko.yaml`

Corpus mapping between wiki categories and DeReKo text classes.

## Progress

- [x] Mapping functionality
- [x] Support for rewrites
- [x] Web service
- [x] JSON script for Kalamar integration
- [x] Integration of multiple mapping files
- [x] Response rewriting
- [x] Support corpus mappings
- [x] Support chaining of mappings
- [ ] Support for negation

## COPYRIGHT AND LICENSE

*Disclaimer*: This software was developed as an experiment with major assistance by AI
(mainly Claude 3.5-sonnet and Claude 4-sonnet, starting with 0.1.1: Claude 4.6 Opus and GPT 5.3 Codex).
The code should not be used as an example on how to create services as Kalamar plugins.

Copyright (C) 2025-2026, [IDS Mannheim](https://www.ids-mannheim.de/)<br>
Author: [Nils Diewald](https://www.nils-diewald.de/)

Koral-Mapper is free software published under the
[BSD-2 License](https://opensource.org/licenses/BSD-2-Clause).
