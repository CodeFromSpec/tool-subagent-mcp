---
version: 14
parent_version: 14
depends_on:
  - path: ROOT/external/owasp-path-traversal
    version: 3
implements:
  - internal/pathvalidation/pathvalidation.go
---

# ROOT/tech_design/internal/pathvalidation

Validates that a file path is safe to write to within a project
directory. This is a security-critical package — it prevents
writing files outside the intended project boundary.

# Public

## Package

`package pathvalidation`

## Threat model

When a tool accepts a file path as input and writes to disk,
the path could attempt to escape the project directory using:

- **Relative traversal**: `../../etc/passwd`
- **Embedded traversal**: `internal/../../outside/file.go`
- **OS-specific separators**: backslash on Windows (`..\..\`)
- **Encoding tricks**: URL-encoded or Unicode sequences
- **Symlinks**: a valid relative path that resolves outside
  the project via a symlink in the directory tree

This package provides a single validation function that callers
use before any write operation.

## Interface

```go
func ValidatePath(path string, projectRoot string) error
```

Returns nil if the path is safe to write within `projectRoot`.
Returns an error describing the violation otherwise.

### Error messages

- `"path is empty"`
- `"path is absolute: <path>"`
- `"path contains directory traversal: <path>"`
- `"path resolves outside project root: <path>"`

# Private

## Implementation

1. Reject empty paths.
2. Reject absolute paths. Use `strings.HasPrefix(path, "/")` to
   catch Unix-style absolute paths (including on Windows, where
   `filepath.IsAbs` returns false for paths starting with `/`
   without a drive letter). Also reject if the path contains `:`
   (Windows drive letter, e.g. `C:\...`).
3. Call `filepath.Clean` on the path to normalize separators
   and resolve `.` segments.
4. Reject if any component is `..` after cleaning.
5. Resolve the full absolute path:
   `abs := filepath.Join(projectRoot, cleaned)`.
6. Call `filepath.EvalSymlinks` on `abs` to resolve any
   symlinks in the path. If the target does not exist yet,
   evaluate the longest existing prefix.
7. Verify that the resolved path starts with `projectRoot`.
   If not, the path escapes the project — reject it.

## Constraints

- This function must not write or create anything on disk.
  It is read-only validation.
- Never attempt to sanitize or fix an invalid path. Reject
  and report — the caller decides what to do.
- Every error must identify the offending path.
