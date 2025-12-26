# Sprint-10 — CLI Data Dictionary

## CLI Invocation Context

### subcommand

* Type: string
* Required: yes
* Description: Selected CLI action

### flags

* Type: object
* Required: yes
* Description: Parsed CLI flags

## Execution Configuration

### strict

* Type: boolean
* Required: no
* Default: false
* Description: Fail validation on warnings

### mode

* Type: string (enum: `clean`, `incremental`)
* Required: no
* Default: `clean`
* Description: Execution strategy

### trace

* Type: boolean
* Required: no
* Default: false
* Description: Enable verbose execution tracing

## Resume Context

### previous_run_id

* Type: string
* Required: yes (resume only)
* Description: Identifier of prior run

### graph

* Type: string (path)
* Required: yes (resume, run, validate)
* Description: Path to graph definition (must match hash for resume)

### retry-failed-only

* Type: boolean
* Required: no
* Default: false
* Description: Only re-execute failed nodes from prior run

## Plugin Selection

### plugins

* Type: array (string)
* Required: no
* Default: none (empty list)
* Description: Explicit allowlist of plugin IDs

## Determinism Notes

- CLI flag order does not affect behavior
- Defaults are explicit
- CLI does not influence graph hashing

## Hash Inclusion Rules

Included:
- None

Excluded:
- CLI flags
- Plugin configuration
