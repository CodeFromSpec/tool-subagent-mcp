---
version: 7
subject_version: 32
implements:
  - internal/parsenode/parsenode_test.go
---

# TEST/tech_design/internal/parsenode

## Context

Each test creates files in `t.TempDir()` under the directory
structure expected by `logicalnames.PathFromLogicalName`. For
example, a test for logical name `ROOT/x/y` creates the file
at `<tmpdir>/code-from-spec/x/y/_node.md`. The test must
change the working directory to `<tmpdir>` before calling
`ParseNode`, and restore it after.

Helper function `testWriteNode(t, dir, logicalName, content)`
creates the file at the correct path relative to `dir`.

## Happy path

### Minimal node — name section only

Logical name: `ROOT/x`

```
---
version: 3
parent_version: 1
---
# ROOT/x

This node has only a name section.
```

Expect:
- `NameSection.Heading` = `"root/x"` (normalized)
- `NameSection.Content` = `"This node has only a name section."`
- `NameSection.Subsections` = nil
- `Public` = nil
- `Private` = nil

### Full node — name, public, private sections

Logical name: `ROOT/payments/fees`

```
---
version: 5
parent_version: 2
depends_on:
  - path: ROOT/architecture/backend
    version: 3
implements:
  - internal/fees/fees.go
---
# ROOT/payments/fees

Calculates transaction fees.

# Public

## Interface

Fee calculation types and functions.

## Constraints

Maximum fee is 5%.

# Implementation

Step-by-step logic for fee calculation.

# Decisions

Chose percentage-based over flat fees.
```

Expect:
- `NameSection.Heading` = `"root/payments/fees"`
- `NameSection.Content` = `"Calculates transaction fees."`
- `Public` not nil:
  - `Public.Heading` = `"public"`
  - `Public.Content` = `""` (no content before first `##`)
  - `Public.Subsections` has 2 entries:
    - `Heading` = `"interface"`, `Content` =
      `"Fee calculation types and functions."`
    - `Heading` = `"constraints"`, `Content` =
      `"Maximum fee is 5%."`
- `Private` has 2 entries:
  - `Heading` = `"implementation"`, `Content` =
    `"Step-by-step logic for fee calculation."`
  - `Heading` = `"decisions"`, `Content` =
    `"Chose percentage-based over flat fees."`

### Test node body

Logical name: `TEST/x`

```
---
version: 2
subject_version: 5
implements:
  - internal/x/x_test.go
---
# TEST/x

Test cases for x.

## Happy path

### Case one

Check basic behavior.
```

Expect:
- `NameSection.Heading` = `"test/x"`
- `NameSection.Content` = `"Test cases for x."`
- `NameSection.Subsections` has 1 entry:
  - `Heading` = `"happy path"`, subsection content includes
    the `### Case one` heading and its text as raw markdown

### Node with no public section

Logical name: `ROOT/decisions`

```
---
version: 1
---
# ROOT/decisions

Architecture decisions.

# Rationale

Why we chose this approach.
```

Expect:
- `Public` = nil
- `NameSection.Heading` = `"root/decisions"`
- `Private` has 1 entry with `Heading` = `"rationale"`

### Public section with content before first subsection

Logical name: `ROOT/a`

```
---
version: 1
---
# ROOT/a

Intent.

# Public

This is direct content of the public section.

## Interface

Types and functions.
```

Expect:
- `Public.Content` = `"This is direct content of the public section."`
- `Public.Subsections` has 1 entry with `Heading` = `"interface"`

## Heading normalization

### Case insensitive public detection

Logical name: `ROOT/c`

```
---
version: 1
---
# ROOT/c

Intent.

# PUBLIC

## Interface

Content.
```

Expect `Public` not nil, `Public.Heading` = `"public"`.

### Public with mixed case and extra whitespace

Logical name: `ROOT/d`

```
---
version: 1
---
# ROOT/d

Intent.

#   PuBLiC

## Interface

Content.
```

Expect `Public` not nil, `Public.Heading` = `"public"`.

### Node name with varied whitespace

Logical name: `ROOT/e`

```
---
version: 1
---
#    ROOT/e

Intent.
```

Expect `NameSection.Heading` = `"root/e"`.

### Subsection headings are normalized

Logical name: `ROOT/f`

```
---
version: 1
---
# ROOT/f

Intent.

# Public

##   Interface

Types.

## CONSTRAINTS

Rules.
```

Expect `Public.Subsections`:
- `Heading` = `"interface"`
- `Heading` = `"constraints"`

### Tab characters in heading whitespace

Logical name: `ROOT/g`

```
---
version: 1
---
# ROOT/g

Intent.

# Public

## 	Interface	

Content.
```

(The `##` line contains tab characters around "Interface".)

Expect subsection `Heading` = `"interface"`.

## Content extraction

### Level-3 and deeper headings are content

Logical name: `ROOT/h`

```
---
version: 1
---
# ROOT/h

Intent.

# Public

## Interface

### Types

Type definitions here.

#### Nested detail

Even deeper content.

## Constraints

### Rule one

Details.
```

Expect:
- `Public.Subsections[0]` (`"interface"`) — content includes
  `### Types`, `Type definitions here.`, `#### Nested detail`,
  and `Even deeper content.` as raw markdown.
- `Public.Subsections[1]` (`"constraints"`) — content includes
  `### Rule one` and `Details.` as raw markdown.

### Fenced code blocks with heading-like content

Logical name: `ROOT/i`

```
---
version: 1
---
# ROOT/i

Intent.

# Public

## Interface

~~~go
// # This is not a heading
// ## Neither is this
func Foo() {}
~~~

After the code block.
```

Expect:
- `Public.Subsections[0]` (`"interface"`) — content includes
  the entire fenced code block and `"After the code block."`.
  The `#` and `##` inside the code block are not treated as
  structural headings.

### Content between sections is trimmed

Logical name: `ROOT/j`

```
---
version: 1
---
# ROOT/j

Intent.

# Public



Content with surrounding blank lines.



## Interface

Also surrounded.


```

Expect:
- `Public.Content` = `"Content with surrounding blank lines."`
  (leading and trailing blank lines trimmed)
- Subsection content = `"Also surrounded."` (trimmed)

## Validation errors

### File does not exist

Call `ParseNode` with `ROOT/nonexistent`.
Expect `errors.Is(err, ErrRead)`.

### No frontmatter delimiters

Logical name: `ROOT/m`

File content (no `---`):

```
# ROOT/m

Just text.
```

Expect `errors.Is(err, ErrFrontmatterMissing)`.

### Content before first heading

Logical name: `ROOT/o`

```
---
version: 1
---
Some text before any heading.

# ROOT/o

Intent.
```

Expect `errors.Is(err, ErrUnexpectedContent)`.

### Level-2 heading before any level-1 heading

Logical name: `ROOT/p`

```
---
version: 1
---
## Orphan subsection

# ROOT/p

Intent.
```

Expect `errors.Is(err, ErrUnexpectedContent)`.

### Node name does not match logical name

Logical name: `ROOT/q`

```
---
version: 1
---
# ROOT/wrong

Intent.
```

Expect `errors.Is(err, ErrInvalidNodeName)`.

### Node name case mismatch is not an error

Logical name: `ROOT/q`

```
---
version: 1
---
# root/Q

Intent.
```

Expect no error. Normalization makes them equal.

### Duplicate public section — same case

Logical name: `ROOT/r`

```
---
version: 1
---
# ROOT/r

Intent.

# Public

First public.

# Public

Second public.
```

Expect `errors.Is(err, ErrDuplicatePublic)`.

### Duplicate public section — different case

Logical name: `ROOT/s`

```
---
version: 1
---
# ROOT/s

Intent.

# Public

First.

# PUBLIC

Second.
```

Expect `errors.Is(err, ErrDuplicatePublic)`.

### Duplicate subsection in public — same case

Logical name: `ROOT/t`

```
---
version: 1
---
# ROOT/t

Intent.

# Public

## Interface

First interface.

## Interface

Second interface.
```

Expect `errors.Is(err, ErrDuplicateSubsection)`.

### Duplicate subsection in public — different case

Logical name: `ROOT/u`

```
---
version: 1
---
# ROOT/u

Intent.

# Public

## Interface

First.

## INTERFACE

Second.
```

Expect `errors.Is(err, ErrDuplicateSubsection)`.

### Duplicate subsection in public — whitespace variation

Logical name: `ROOT/v`

```
---
version: 1
---
# ROOT/v

Intent.

# Public

## Interface

First.

##   Interface

Second.
```

Expect `errors.Is(err, ErrDuplicateSubsection)`.

### First element is a paragraph — missing node name

Logical name: `ROOT/w`

```
---
version: 1
---
This is a paragraph, not a heading.
```

Expect `errors.Is(err, ErrUnexpectedContent)`.
