---
version: 2
subject_version: 2
implements:
  - internal/find_replace/find_replace_test.go
---

# TEST/tech_design/internal/tools/find_replace

## Happy Path

### Replaces a string successfully

Create a spec file with `implements: [internal/find_replace/test.go]`.
Create the target file with content `hello world`.
Call the handler with `OldString: "hello"`, `NewString: "goodbye"`.

Expect: success result with text `"edited internal/find_replace/test.go"`.
Verify: file content is `goodbye world`.

### Replaces multi-line old_string

Create a target file with multi-line content. Call the handler
with an `OldString` that spans multiple lines.

Expect: success, file content has the multi-line block replaced.

### Replaces with empty new_string

Create a target file. Call the handler with a non-empty
`OldString` and an empty `NewString`.

Expect: success, the matched text is deleted from the file.

### Backslash path is normalized

Windows only — skip on other platforms.

Create a spec file with `implements: [internal/find_replace/test.go]`.
Create the target file. Call the handler with
`Path: "internal\\find_replace\\test.go"`.

Expect: success, path is normalized to forward slashes.

## Failure Cases

### Invalid logical name prefix

Call the handler with `LogicalName: "INVALID/something"`.

Expect: tool error, result contains the invalid name.

### Nonexistent logical name

Call the handler with a logical name that does not resolve to
an existing spec file.

Expect: tool error.

### Path not in implements

Call the handler with a valid logical name but a path not
listed in `implements`.

Expect: tool error containing `"path not allowed"` and the
list of allowed paths.

### Path traversal attempt

Create a spec file with `implements` containing a path with
`../../`. Call `ValidatePath` should catch the traversal.

Expect: tool error.

### Empty path

Call the handler with an empty `Path`.

Expect: tool error containing `"path is empty"`.

### Symlink escaping project root

Create a symlink inside the project that points outside it.
Create a spec file with `implements` listing a path through
the symlink.

Expect: tool error containing `"resolves outside project root"`.
Skip if symlink creation fails.

### File does not exist

Create a spec file with `implements` listing a file that does
not exist on disk. Call the handler.

Expect: tool error containing `"file does not exist"`.

### old_string not found

Create a target file. Call the handler with an `OldString`
that does not appear in the file.

Expect: tool error containing `"old_string not found"`.

### old_string matches multiple locations

Create a target file with repeated content. Call the handler
with an `OldString` that appears more than once.

Expect: tool error containing `"old_string matches multiple locations"`.
