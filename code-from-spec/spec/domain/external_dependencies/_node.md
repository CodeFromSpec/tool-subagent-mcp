---
version: 1
parent_version: 1
---

# ROOT/domain/external_dependencies

## Intent

Defines the structure of external dependencies as relevant to this
server: where they live, how they are identified, and how files
within them are selected.

## Contracts

### Location

Each external dependency is a folder under `code-from-spec/external/`.
The `external/` directory itself is optional — a project may have
no external dependencies.

### Logical names

The logical name of an external dependency is `EXTERNAL/` followed
by the folder name — e.g., folder `celcoin-api/` becomes
`EXTERNAL/celcoin-api`.

### Entry point

Every dependency folder must contain an `_external.md` at its root.
This is always included in the chain when the dependency is
referenced.

Frontmatter contains at least `version`:

```yaml
---
version: 1
---
```

Other frontmatter fields may exist and must be ignored.

### Additional file selection

A `depends_on` entry referencing an `EXTERNAL/` path may include a
`filter` field with glob patterns. When present, files within the
external dependency folder that match any of the patterns are also
included in the chain, in addition to `_external.md`.

Patterns are relative to the external dependency folder (not the
project root). Example:

```yaml
depends_on:
  - path: EXTERNAL/celcoin-api
    version: 5
    filter:
      - "endpoints/*.md"
      - "types.md"
```

Files in the folder that do not match any pattern are not included.
The tool must ignore subfolders and files it does not need — only
`_external.md` plus filtered files are read.
