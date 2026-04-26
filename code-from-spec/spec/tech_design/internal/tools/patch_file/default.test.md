---
version: 3
parent_version: 3
implements:
  - internal/patch_file/patch_file_test.go
---

# TEST/tech_design/internal/tools/patch_file

## Context

Each test uses `t.TempDir()` as the project root and working
directory. A spec tree is created with the necessary frontmatter
containing an `Implements` list. The handler is called with
`PatchFileArgs` including the `LogicalName` of the node.

Unified diffs used in tests follow the standard format:

```
--- a/path/to/file
+++ b/path/to/file
@@ -start,count +start,count @@
 context line
-removed line
+added line
```

## Happy Path

### Applies a simple diff

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file at
`output/file.go` with content:

```
package main

func hello() string {
	return "hello"
}
```

Call the handler with `LogicalName: "ROOT/a"`,
`Path: "output/file.go"`, and a diff that changes `"hello"`
to `"world"`:

```
--- a/output/file.go
+++ b/output/file.go
@@ -3,3 +3,3 @@
 func hello() string {
-	return "hello"
+	return "world"
 }
```

Expect: success result with text `"patched output/file.go"`.
Verify the file on disk has `"world"` instead of `"hello"`.

### Applies a multi-hunk diff

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file with
multiple functions. Call the handler with a diff containing
two hunks that modify different parts of the file.

Expect: success. Both hunks applied correctly.

### Applies a diff that adds lines

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with a diff that adds new lines without
removing any.

Expect: success. New lines present in the file.

### Applies a diff that removes lines

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with a diff that removes lines without
adding any.

Expect: success. Lines removed from the file.

### Path with backslashes is normalized (Windows only)

Skip this test on non-Windows platforms.

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with `Path: "output\\file.go"` and a valid
diff.

Expect: success result with text `"patched output/file.go"`.

## Failure Cases

### Invalid logical name prefix

Call the handler with `LogicalName: "EXTERNAL/something"`.

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

### File does not exist

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Do not create the file.
Call the handler with a valid diff.

Expect: tool error containing
`"file does not exist: output/file.go"`.

### Malformed diff

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with `Diff: "this is not a valid diff"`.

Expect: tool error containing `"failed to parse diff"`.

### Diff with zero file entries

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with an empty diff (only whitespace or
empty string).

Expect: tool error containing
`"diff must contain exactly one file"`.

### Diff with multiple file entries

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with a diff that modifies two different
files.

Expect: tool error containing
`"diff must contain exactly one file"`.

### Diff context does not match file

Create a spec tree with `ROOT/a` having
`implements: ["output/file.go"]`. Write an initial file.
Call the handler with a diff whose context lines do not
match the file's actual content.

Expect: tool error containing
`"failed to apply diff to output/file.go"`.
