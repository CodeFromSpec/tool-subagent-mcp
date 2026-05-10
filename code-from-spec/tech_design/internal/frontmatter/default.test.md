---
version: 14
subject_version: 33
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
  - path: ROOT/architecture/backend
    version: 5
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---
```

Expect:
- `Version` = 3
- `ParentVersion` = pointer to 2
- `SubjectVersion` = nil
- `DependsOn` has two entries:
  - `LogicalName` = `"ROOT/other"`, `Version` = 1
  - `LogicalName` = `"ROOT/architecture/backend"`, `Version` = 5
- `Implements` = `["internal/config/config.go", "internal/config/config_test.go"]`

### Parses test node frontmatter

Create a file with `subject_version`:

```
---
version: 2
subject_version: 5
implements:
  - internal/config/config_test.go
---
```

Expect:
- `Version` = 2
- `ParentVersion` = nil
- `SubjectVersion` = pointer to 5
- `DependsOn` = nil
- `Implements` = `["internal/config/config_test.go"]`

### Parses frontmatter with only version

Create a file with only `version`:

```
---
version: 5
---
```

Expect:
- `Version` = 5
- `ParentVersion` = nil
- `SubjectVersion` = nil
- `DependsOn` = nil
- `Implements` = nil
- No error.

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

Expect `errors.Is(err, ErrMissingVersion)`.

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
Expect `errors.Is(err, ErrRead)`.

### No frontmatter delimiters

Create a file with no `---` at all:

```
Just some text.
```

Expect `errors.Is(err, ErrFrontmatterMissing)`.

### Malformed YAML in frontmatter

Create a file with invalid YAML between delimiters:

```
---
version: [invalid
---
```

Expect `errors.Is(err, ErrFrontmatterParse)`.

### Missing version field

Create a file with:

```
---
parent_version: 1
implements:
  - internal/config/config.go
---
```

Expect `errors.Is(err, ErrMissingVersion)`.
