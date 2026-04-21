# Path Traversal — OWASP Guidance

## What is path traversal

A path traversal attack aims to access files or directories
outside the intended scope by manipulating file path inputs.
The attacker supplies input like `../../../etc/passwd` to
escape the expected base directory.

## Common bypass techniques

Simple validation (e.g., rejecting `../`) consistently fails
against these techniques:

- **URL encoding**: `%2e%2e%2f` decodes to `../`
- **Double encoding**: `%252e%252e%252f` decodes in two passes
- **Unicode/UTF-8**: `..%c0%af` represents `../` in some parsers
- **Null byte injection**: `file.txt%00.pdf` — the application
  sees `.pdf` but the OS reads `file.txt`
- **OS-specific separators**: both `/` and `\` work on Windows;
  `\` is often overlooked in validation
- **Extra characters**: Windows allows trailing `.`, `\`, or `/`
  in file paths that are silently stripped

## Prevention recommendations

### Prefer whitelisting over blacklisting

Blacklist approaches (filtering `../`) consistently fail.
Accept only explicitly approved file paths or indices.

### Normalize before validating

Apply path canonicalization (e.g., Go's `filepath.Clean`)
before any validation check. This resolves `.`, `..`,
duplicate separators, and OS-specific quirks into a canonical
form.

### Resolve and verify containment

After normalization, resolve the full absolute path and verify
it starts with the expected base directory. This catches any
traversal that survived normalization.

### Resolve symlinks

A path that appears to be within the base directory may resolve
outside it via symlinks. Use symlink resolution (e.g., Go's
`filepath.EvalSymlinks`) before the containment check.

### Do not sanitize — reject

Never attempt to clean or fix a suspicious path. If validation
fails, reject the input entirely. Sanitization introduces
complexity and new bypass opportunities.

### Validate at every layer

Validate at configuration time (when paths are first declared)
and again at write time (before each I/O operation). Defense
in depth prevents a single bypass from compromising the system.
