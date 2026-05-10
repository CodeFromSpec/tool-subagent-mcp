---
version: 10
subject_version: 38
implements:
  - internal/write_file/write_file_test.go
---

# TEST/tech_design/internal/tools/write_file

## Context

Each test uses `t.TempDir()` as the project root and working
directory. A spec tree is created with the necessary frontmatter
containing an `Implements` list. The handler is called with
`WriteFileArgs` including the `LogicalName` of the node.

## Happy Path

### Writes file successfully

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Call the handler with
`LogicalName: "ROOT/a"`, `Path: "output/file.go"`, and
`Content: "package main"`.

Expect: success result with text `"wrote output/file.go"`.
Verify the file exists on disk with the correct content.

### Creates intermediate directories

Create a spec tree with `ROOT/a` having
`implements: ["deep/nested/dir/file.go"]`. Call the handler
with `Path: "deep/nested/dir/file.go"`.

Expect: success. Directories created automatically.

### Overwrites existing file

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file at
that path. Call the handler with new content.

Expect: success. File content replaced.

### Path with backslashes is normalized (Windows only)

Skip this test on non-Windows platforms — backslash is a
valid filename character on Linux/macOS, not a separator.

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Call the handler with
`LogicalName: "ROOT/a"`, `Path: "output\\file.go"`, and
`Content: "package main"`.

Expect: success result with text `"wrote output/file.go"`.
The backslash path matches the forward-slash implements
entry after normalization.

## Failure Cases

### Invalid logical name prefix

Call the handler with `LogicalName: "ROOT/external/something"`.

Expect: tool error.

### Nonexistent logical name

Call the handler with `LogicalName: "ROOT/nonexistent"`.
Do not create the corresponding spec file.

Expect: tool error.

### Path not in implements

Create a spec tree with `ROOT/a` having
`implements: ["allowed/file.go"]`. Call the handler with
`Path: "other/file.go"`.

Expect: tool error containing `"path not allowed"` and
listing the allowed paths.

### Path traversal attempt

Create a spec tree with `ROOT/a` having
`implements: ["../../etc/passwd"]`. Call the handler with
`Path: "../../etc/passwd"`.

Expect: tool error from `ValidatePath`.

### Empty path

Create a spec tree with `ROOT/a` having
`implements: ["some/file.go"]`. Call the handler with
`Path: ""`.

Expect: tool error containing `"path is empty"`.

### Symlink escaping project root

Create a symlink inside the temp dir pointing outside it.
Create a spec tree with the symlink path in `implements`.
Call the handler with that path.

Expect: tool error containing `"resolves outside project root"`.
