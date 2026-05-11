---
version: 12
subject_version: 14
implements:
  - internal/pathvalidation/pathvalidation_test.go
---

# TEST/tech_design/internal/pathvalidation

## Context

Each test uses `t.TempDir()` as the project root. When testing
symlinks, create them inside the temp directory pointing to
targets outside it.

## Happy Path

### Simple relative path

Input: `"internal/config/config.go"`, project root = temp dir.
Expect: no error.

### Nested path

Input: `"cmd/subagent-mcp/main.go"`, project root = temp dir.
Expect: no error.

### Single filename

Input: `"main.go"`, project root = temp dir.
Expect: no error.

### Path with dot segment

Input: `"internal/./config/config.go"`, project root = temp dir.
Expect: no error (cleaned to `internal/config/config.go`).

## Edge Cases

### Path with trailing slash

Input: `"internal/config/"`, project root = temp dir.
Expect: no error.

### Path with duplicate separators

Input: `"internal//config//config.go"`, project root = temp dir.
Expect: no error (cleaned by `filepath.Clean`).

## Failure Cases

### Empty path

Input: `""`, project root = temp dir.
Expect: error containing `"path is empty"`.

### Absolute path with leading slash

Input: `"/etc/passwd"`, project root = temp dir.
Expect: error containing `"path is absolute"`.

### Absolute path with drive letter (Windows-style)

Input: `"C:\\Windows\\system32"`, project root = temp dir.
Expect: error containing `"path is absolute"`.

### Simple traversal

Input: `"../../etc/passwd"`, project root = temp dir.
Expect: error containing `"directory traversal"`.

### Embedded traversal

Input: `"internal/../../outside/file.go"`, project root = temp dir.
Expect: error containing `"directory traversal"`.

### Symlink escaping project root

Create a symlink inside the temp dir pointing to a directory
outside it. Input: `"<symlink>/file.go"`, project root = temp dir.
Expect: error containing `"resolves outside project root"`.

### Traversal disguised with dot segments

Input: `"a/../../outside"`, project root = temp dir.
Expect: error containing `"directory traversal"`.
