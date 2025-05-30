# KoralPipe-TermMapper2

A web service for transforming JSON objects using term mapping rules.

## Overview

This service provides a REST API for transforming JSON objects according to mapping rules defined in a YAML configuration file. The mappings can be applied in both directions (A to B or B to A) and support foundry and layer overrides.

## Installation

```bash
go get github.com/KorAP/KoralPipe-TermMapper2
```

## Usage

### Starting the Server

```bash
termmapper -c config.yaml -p 8080 -l info
```

Command line options:
- `--config` or `-c`: YAML configuration file containing mapping directives (required)
- `--port` or `-p`: Port to listen on (default: 8080)
- `--log-level` or `-l`: Log level (debug, info, warn, error) (default: info)
- `--help` or `-h`: Show help message

### Configuration File Format

The configuration file should be in YAML format and contain a list of mapping definitions:

```yaml
- id: opennlp-mapper
  foundryA: opennlp
  layerA: p
  foundryB: upos
  layerB: p
  mappings:
    - "[PIDAT] <> [opennlp/p=PIDAT & opennlp/p=AdjType:Pdt]"
    - "[DET] <> [opennlp/p=DET]"

- id: simple-mapper
  mappings:
    - "[A] <> [B]"
```

Each mapping list has:
- `id`: Unique identifier for the mapping list
- `foundryA`, `layerA`: Default foundry and layer for the left side of mappings
- `foundryB`, `layerB`: Default foundry and layer for the right side of mappings
- `mappings`: List of mapping rules in the format `[pattern] <> [replacement]`

### API Endpoints

#### Transform JSON Object

```http
POST /:map/query
```

Transform a JSON object using the specified mapping list.

Parameters:
- `:map`: ID of the mapping list to use
- `dir` (query): Direction of transformation (`atob` or `btoa`, default: `atob`)
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

#### Health Check

```http
GET /health
```

Returns "OK" if the service is running.

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o termmapper ./cmd/termmapper
``` 