---
version: 2
parent_version: 22
implements:
  - internal/modes/codegen/write_file_test.go
---

# TEST/tech_design/internal/modes/codegen/tools/write_file

## Context

Each test uses `t.TempDir()` as the project root and working
directory. A `Target` is created with a known `Frontmatter`
containing an `Implements` list. The handler is called with
`WriteFileArgs`.

## Happy Path

### Writes file successfully

Create a `Target` with `Implements: ["output/file.go"]`.
Call the handler with `Path: "output/file.go"` and
`Content: "package main"`.

Expect: success result with text `"wrote output/file.go"`.
Verify the file exists on disk with the correct content.

### Creates intermediate directories

Create a `Target` with
`Implements: ["deep/nested/dir/file.go"]`.
Call the handler with `Path: "deep/nested/dir/file.go"`.

Expect: success. Directories created automatically.

### Overwrites existing file

Create a `Target` with `Implements: ["output/file.go"]`.
Write an initial file at that path. Call the handler with
new content.

Expect: success. File content replaced.

## Failure Cases

### Path not in implements

Create a `Target` with `Implements: ["allowed/file.go"]`.
Call the handler with `Path: "other/file.go"`.

Expect: tool error containing `"path not allowed"` and
listing the allowed paths.

### Path traversal attempt

Create a `Target` with `Implements: ["../../etc/passwd"]`.
Call the handler with `Path: "../../etc/passwd"`.

Expect: tool error from `ValidatePath`.

### Empty path

Call the handler with `Path: ""`.

Expect: tool error containing `"path is empty"`.

### Symlink escaping project root

Create a symlink inside the temp dir pointing outside it.
Add the symlink path to `Implements`. Call the handler with
that path.

Expect: tool error containing `"resolves outside project root"`.
