---
version: 27
parent_version: 11
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
  - path: EXTERNAL/goccy-go-yaml
    version: 1
  - path: EXTERNAL/yuin-goldmark
    version: 1
  - path: EXTERNAL/golang-x-text
    version: 1
  - path: ROOT/tech_design/internal/logical_names
    version: 26
implements:
  - internal/parsenode/parsenode.go
---

# ROOT/tech_design/internal/parsenode

## Intent

Reads and parses a spec node file, returning a structured
representation of all sections.

## Context

### Package

`package parsenode`

### Dependencies

- `github.com/goccy/go-yaml` — YAML parsing of the frontmatter
  block.
- `github.com/yuin/goldmark` — CommonMark parsing of the body.
  The body is parsed into an AST; only level-1 and level-2
  headings are used as structural delimiters.
- `golang.org/x/text/cases` — Unicode simple case folding for
  heading normalization.

## Contracts

### Interface

```go
type DependsOn struct {
	LogicalName string
	Version     int
}

type Frontmatter struct {
	Version        int
	ParentVersion  *int
	SubjectVersion *int
	DependsOn      []DependsOn
	Implements     []string
}

type Subsection struct {
	Heading string
	Content string
}

type Section struct {
	Heading     string
	Content     string
	Subsections []Subsection
}

type Node struct {
	Frontmatter Frontmatter
	NameSection Section
	Public      *Section
	Private     []Section
}

var (
	ErrRead               = errors.New("error reading file")
	ErrFrontmatterParse   = errors.New("error parsing frontmatter")
	ErrFrontmatterMissing = errors.New("frontmatter not found")
	ErrUnexpectedContent  = errors.New("unexpected content before first heading")
	ErrInvalidNodeName    = errors.New("node name section does not match file path")
	ErrDuplicatePublic       = errors.New("duplicate public section")
	ErrDuplicateSubsection  = errors.New("duplicate subsection in public")
)

func ParseNode(logicalName string) (*Node, error)
```

`Public` is nil when no `# Public` section exists in the file.

Errors returned by `ParseNode` wrap the sentinel with context
(file path, underlying error) using `fmt.Errorf`, so callers
can match with `errors.Is()`.

### Heading normalization (internal)

```go
func normalizeHeading(raw string) string
```

Internal helper. Applies the framework normalization rules to
a raw heading text:

1. Trim leading and trailing whitespace.
2. Collapse each sequence of one or more whitespace characters
   to a single space (`U+0020`).
3. Apply Unicode simple case folding using `cases.Fold()` from
   `golang.org/x/text/cases`.

Whitespace characters are space (`U+0020`) and horizontal tab
(`U+0009`).

### Parsing algorithm

#### Step 1 — Resolve logical name

Resolve the logical name to a file path using
`logicalnames.PathFromLogicalName`.

#### Step 2 — Read file and extract frontmatter

The frontmatter is the YAML block between the first `---` and
the second `---` at the top of the file. Extract and parse it
with go-yaml. Unknown fields are ignored.

Frontmatter fields:

| Field | Type | Description |
|---|---|---|
| `version` | int | Node version. |
| `parent_version` | *int | Parent version. Nil if absent. |
| `subject_version` | *int | Subject version (test nodes). Nil if absent. |
| `depends_on` | []DependsOn | Cross-tree dependencies. |
| `implements` | []string | Output files. |

Each `depends_on` entry:

| YAML key | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Logical name of the dependency. |
| `version` | int | yes | Known version of the dependency. |

#### Step 3 — Parse body as CommonMark

The body is everything after the closing `---` of the
frontmatter. Parse it with goldmark to produce an AST.

#### Step 4 — Validate: first child is a level-1 heading

The first direct child of the document root node must be a
level-1 heading. If it is not, it is an error.

#### Step 5 — Validate: node name section

Extract the inline text content of the first level-1 heading
(see "Extracting heading text" in `EXTERNAL/yuin-goldmark`).
Apply `normalizeHeading` to it and to the logical name received
as argument. If the results do not match, it is an error.

#### Step 6 — Validate: no duplicate public section

For each level-1 heading, extract its inline text content
(see "Extracting heading text" in `EXTERNAL/yuin-goldmark`)
and apply `normalizeHeading`. If more than one result equals
`public`, it is an error.

#### Step 7 — Validate: no duplicate public subsections

For each level-2 heading within the public section, extract
its inline text content (see "Extracting heading text" in
`EXTERNAL/yuin-goldmark`) and apply `normalizeHeading`. If any
two results are equal, it is an error.

#### Step 8 — Extract sections

Iterate the direct children of the document root. Each level-1
heading starts a new section. All AST nodes between one level-1
heading and the next (or end of document) are the content of
that section.

For each section, extract:
- **Heading** — extract the inline text content of the level-1
  heading (see "Extracting heading text" in
  `EXTERNAL/yuin-goldmark`) and apply `normalizeHeading`.
- **Content** — the raw source bytes between the end of the
  level-1 heading and the start of the first level-2 heading
  within the section (or the start of the next level-1 heading
  / end of document if there are no level-2 headings).
- **Subsections** — each level-2 heading within the section
  starts a subsection. A subsection's heading is obtained by
  extracting the inline text content of the level-2 heading
  (see "Extracting heading text" in `EXTERNAL/yuin-goldmark`)
  and applying `normalizeHeading`. A
  subsection's content is the raw source bytes between the end
  of the level-2 heading and the start of the next level-2
  heading, the next level-1 heading, or the end of document.

Leading and trailing blank lines in content are trimmed.

#### Step 9 — Classify sections

1. The first section is the node name section.
2. A section whose `normalizeHeading` result equals `public`
   is the public section.
3. All other sections are private.

### Invariants

- The first element after the frontmatter is always a level-1
  heading. If it is not, the file is invalid
  (`ErrUnexpectedContent`).
- Every subsection (`##`) is contained within a section (`#`).
  There are no orphan subsections.
- There is exactly one node name section (the first `#`),
  and its normalized heading matches the logical name received
  as argument.
- There is at most one public section (`# Public`). Duplicates
  are rejected.
- All `##` subsections within `# Public` have unique normalized
  headings. Duplicates are rejected.
- Headings of level 3 and deeper are content, not structural
  delimiters. They appear inside `Section.Content` or
  `Subsection.Content` as raw markdown.

### Error handling

All errors wrap a sentinel so callers can use `errors.Is()`:

| Sentinel | Returned when |
|---|---|
| `ErrRead` | The file cannot be read. |
| `ErrFrontmatterParse` | The YAML frontmatter is malformed. |
| `ErrFrontmatterMissing` | No `---` delimiters found at the top of the file. |
| `ErrUnexpectedContent` | Non-heading content appears before the first level-1 heading. |
| `ErrInvalidNodeName` | The first level-1 heading does not match the logical name. |
| `ErrDuplicatePublic` | More than one level-1 heading normalizes to `public`. |
| `ErrDuplicateSubsection` | Two or more level-2 headings within `# Public` have the same normalized text. |
