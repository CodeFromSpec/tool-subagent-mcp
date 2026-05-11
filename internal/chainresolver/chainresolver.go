// code-from-spec: ROOT/tech_design/internal/chain_resolver@v80

// Package chainresolver resolves the ordered list of files that form the
// chain for a given target logical name. The chain is separated into:
//   - Ancestors: the target's ancestor nodes (sorted alphabetically by logical name)
//   - Target: the target node itself
//   - Dependencies: cross-tree dependencies declared in frontmatter
//   - Code: existing generated source files declared in the target's Implements list
//
// The chain is assembled by:
//  1. Walking up from the target via ParentLogicalName, sorting, and splitting
//     into Ancestors + Target.
//  2. Reading the target's frontmatter DependsOn entries and resolving each to
//     a ChainItem with an optional qualifier.
//  3. Collecting existing Implements paths into Code.
//  4. Normalizing all file paths to forward slashes.
//  5. Deduplicating Ancestors and Dependencies (keeping first occurrence;
//     qualified entries are dropped when a nil-qualifier entry for the same
//     file is already present).
package chainresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
)

// ChainItem represents a single node in the chain.
//
// FilePath is the spec file path relative to the project root, always using
// forward slashes as separators.
//
// Qualifier, when non-nil, tells the caller to use only the
// "## <qualifier>" subsection within "# Public" of the file.
// When Qualifier is nil, the caller should use the entire "# Public" section.
type ChainItem struct {
	LogicalName string
	FilePath    string
	Qualifier   *string
}

// Chain holds the fully resolved chain for a target node.
//
//   - Ancestors: ancestor nodes sorted alphabetically by logical name.
//   - Target: the target node itself.
//   - Dependencies: cross-tree dependencies from the target's DependsOn
//     frontmatter field, sorted by FilePath then Qualifier.
//   - Code: paths to generated source files listed in the target's Implements
//     field that already exist on disk.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
	Code         []string
}

// ResolveChain builds the complete chain for the given target logical name.
// Returns an error if:
//   - any logical name cannot be resolved to a file path
//   - the target's frontmatter cannot be parsed
//   - a dependency file does not exist on disk
func ResolveChain(targetLogicalName string) (*Chain, error) {
	var chain Chain

	// -----------------------------------------------------------------------
	// Step 1 — Ancestors and Target
	//
	// Starting from the target, repeatedly call ParentLogicalName to walk
	// upward. Collect all logical names (including the target). Sort them
	// alphabetically — the deepest/most-specific name sorts last and becomes
	// the Target; all others become Ancestors. Qualifier is always nil for
	// these items (the full # Public section applies).
	// -----------------------------------------------------------------------
	collected := []string{targetLogicalName}
	current := targetLogicalName

	for {
		hasParent, ok := logicalnames.HasParent(current)
		if !ok {
			// logicalnames.HasParent returned ok=false — the name is invalid.
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		if !hasParent {
			// Reached the root; stop walking.
			break
		}
		parent, ok := logicalnames.ParentLogicalName(current)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		collected = append(collected, parent)
		current = parent
	}

	// Sort alphabetically so that ancestor names precede the target name.
	sort.Strings(collected)

	// Build a ChainItem for each logical name. Qualifier is nil for all
	// ancestors and the target.
	items := make([]ChainItem, 0, len(collected))
	for _, name := range collected {
		filePath, ok := logicalnames.PathFromLogicalName(name)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", name)
		}
		items = append(items, ChainItem{
			LogicalName: name,
			FilePath:    filePath,
			Qualifier:   nil,
		})
	}

	// The last item (deepest path) is the Target; the rest are Ancestors.
	chain.Target = items[len(items)-1]
	if len(items) > 1 {
		chain.Ancestors = items[:len(items)-1]
	}

	// -----------------------------------------------------------------------
	// Step 2 — Dependencies
	//
	// Read the target's frontmatter and process each DependsOn entry:
	//  1. Resolve the file path via PathFromLogicalName.
	//  2. Extract any qualifier from the logical name.
	//  3. Verify the file exists on disk.
	//  4. Append a ChainItem to Dependencies.
	//
	// After collecting all entries, sort by FilePath ascending, with nil
	// Qualifier sorting before non-nil Qualifier for the same FilePath.
	// -----------------------------------------------------------------------

	// Resolve the target's spec file path (needed for ParseFrontmatter).
	targetFilePath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	targetFM, err := frontmatter.ParseFrontmatter(targetFilePath)
	if err != nil {
		// Wrap the underlying error so callers can still inspect it.
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	for _, dep := range targetFM.DependsOn {
		// Step 2.1 — Resolve file path (qualifier is stripped internally).
		depFilePath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
		}

		// Step 2.2 — Determine qualifier.
		var qualifier *string
		hasQual, qualOk := logicalnames.HasQualifier(dep.LogicalName)
		if !qualOk {
			return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
		}
		if hasQual {
			qualName, qualNameOk := logicalnames.QualifierName(dep.LogicalName)
			if !qualNameOk {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			qualifier = &qualName
		}

		// Step 2.3 — Verify the file exists on disk.
		if _, err := os.Stat(depFilePath); err != nil {
			return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
		}

		// Step 2.4 — Append to Dependencies.
		chain.Dependencies = append(chain.Dependencies, ChainItem{
			LogicalName: dep.LogicalName,
			FilePath:    depFilePath,
			Qualifier:   qualifier,
		})
	}

	// Sort Dependencies: primary key = FilePath ascending; secondary key =
	// Qualifier where nil < non-nil (nil sorts first).
	sort.Slice(chain.Dependencies, func(i, j int) bool {
		fpI := chain.Dependencies[i].FilePath
		fpJ := chain.Dependencies[j].FilePath
		if fpI != fpJ {
			return fpI < fpJ
		}
		qI := chain.Dependencies[i].Qualifier
		qJ := chain.Dependencies[j].Qualifier
		if qI == nil && qJ != nil {
			return true // nil before non-nil
		}
		if qI != nil && qJ == nil {
			return false
		}
		if qI != nil && qJ != nil {
			return *qI < *qJ
		}
		return false // both nil — equal
	})

	// -----------------------------------------------------------------------
	// Step 3 — Code
	//
	// Collect each path from the target's Implements list that already exists
	// on disk. Paths that do not yet exist are silently skipped.
	// -----------------------------------------------------------------------
	for _, implPath := range targetFM.Implements {
		if _, err := os.Stat(implPath); err == nil {
			chain.Code = append(chain.Code, implPath)
		}
		// File does not exist — skip it per spec.
	}

	// -----------------------------------------------------------------------
	// Step 4 — Normalize file paths
	//
	// Convert every file path to forward slashes so the output is
	// OS-independent. This is required even on Unix (no-op there, but
	// ensures consistency across platforms).
	// -----------------------------------------------------------------------
	for i := range chain.Ancestors {
		chain.Ancestors[i].FilePath = filepath.ToSlash(chain.Ancestors[i].FilePath)
	}
	chain.Target.FilePath = filepath.ToSlash(chain.Target.FilePath)
	for i := range chain.Dependencies {
		chain.Dependencies[i].FilePath = filepath.ToSlash(chain.Dependencies[i].FilePath)
	}
	for i, fp := range chain.Code {
		chain.Code[i] = filepath.ToSlash(fp)
	}

	// -----------------------------------------------------------------------
	// Step 5 — Deduplicate
	//
	// Rules (applied across both Ancestors and Dependencies, keeping the
	// first occurrence):
	//
	//  a) Two entries are duplicates when they share the same FilePath AND the
	//     same Qualifier (both nil, or both equal non-nil strings).
	//
	//  b) If an entry with Qualifier=nil exists for a FilePath (meaning the
	//     entire # Public section is included), any entry with the same
	//     FilePath and a non-nil Qualifier is redundant — the full # Public
	//     already covers every subsection.
	//
	// We first build a set of "full-coverage" FilePaths (those that appear
	// with Qualifier=nil in either Ancestors or Dependencies), then sweep
	// both slices keeping only first-seen non-redundant entries.
	// -----------------------------------------------------------------------

	// Build the full-coverage set: FilePaths where Qualifier=nil appears at
	// least once in Ancestors or Dependencies.
	fullCoveragePaths := make(map[string]bool)
	for _, item := range chain.Ancestors {
		if item.Qualifier == nil {
			fullCoveragePaths[item.FilePath] = true
		}
	}
	for _, item := range chain.Dependencies {
		if item.Qualifier == nil {
			fullCoveragePaths[item.FilePath] = true
		}
	}

	// itemKey produces a unique string key for a (FilePath, Qualifier) pair,
	// used to detect exact duplicates.
	itemKey := func(filePath string, qualifier *string) string {
		if qualifier == nil {
			return filePath + "\x00nil"
		}
		return filePath + "\x00q:" + *qualifier
	}

	seenKeys := make(map[string]bool)

	// deduplicateItems applies both deduplication rules to a slice of items.
	// It relies on the shared seenKeys and fullCoveragePaths maps so that
	// entries in Ancestors suppress duplicates in Dependencies and vice versa.
	deduplicateItems := func(items []ChainItem) []ChainItem {
		result := make([]ChainItem, 0, len(items))
		for _, item := range items {
			// Rule (b): drop qualified entries whose FilePath already has a
			// nil-qualifier entry anywhere in the combined set.
			if item.Qualifier != nil && fullCoveragePaths[item.FilePath] {
				continue
			}
			// Rule (a): drop exact duplicates; keep first occurrence.
			key := itemKey(item.FilePath, item.Qualifier)
			if seenKeys[key] {
				continue
			}
			seenKeys[key] = true
			result = append(result, item)
		}
		return result
	}

	chain.Ancestors = deduplicateItems(chain.Ancestors)
	chain.Dependencies = deduplicateItems(chain.Dependencies)

	return &chain, nil
}
