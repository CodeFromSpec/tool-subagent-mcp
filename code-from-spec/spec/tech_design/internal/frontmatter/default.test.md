---
version: 4
parent_version: 17
implements:
  - internal/frontmatter/frontmatter_test.go
---

# TEST/tech_design/internal/frontmatter

## Context

Each test uses `t.TempDir()` to create an isolated
temporary directory. Test files are created with
controlled frontmatter content. `ParseFrontmatter` is
called with the path to each test file.

## Happy Path

### Parses complete frontmatter

Create a file with all fields:

```
---
version: 3
parent_version: 2
depends_on:
  - path: ROOT/other
    version: 1
  - path: EXTERNAL/database
    version: 5
    filter:
      - "schema/*.sql"
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---
```

Expect `DependsOn` has two entries:
- `LogicalName` = `"ROOT/other"`, `Filter` = nil
- `LogicalName` = `"EXTERNAL/database"`, `Filter` = `["schema/*.sql"]`

`Implements` = `["internal/config/config.go", "internal/config/config_test.go"]`.

### Parses frontmatter with only implements

Create a file with only `implements`:

```
---
version: 1
parent_version: 1
implements:
  - internal/config/config.go
---
```

Expect `DependsOn` = nil, `Implements` = `["internal/config/config.go"]`.

### Parses frontmatter with no relevant fields

Create a file with only `version`:

```
---
version: 5
---
```

Expect `DependsOn` = nil, `Implements` = nil. No error.

### Ignores unknown frontmatter fields

Create a file with extra fields:

```
---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---
```

Expect no error. Known fields parsed correctly.
Unknown fields ignored.

## Edge Cases

### Empty frontmatter

Create a file with:

```
---
---
```

Expect all fields zero/nil. No error.

### File with only frontmatter, nothing after

Create a file with:

```
---
version: 1
---
```

Expect no error. Body is not read.

## Failure Cases

### File does not exist

Call `ParseFrontmatter` with a non-existent path.
Expect an error containing the file path.

### No frontmatter delimiters

Create a file with no `---` at all:

```
Just some text.
```

Expect an error indicating frontmatter not found.

### Malformed YAML in frontmatter

Create a file with invalid YAML between delimiters:

```
---
version: [invalid
---
```

Expect an error indicating parse failure.
