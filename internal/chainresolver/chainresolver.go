// code-from-spec: ROOT/tech_design/internal/chain_resolver@v72

// Package chainresolver resolves the ordered list of files that form the
// chain for a given target logical name. The chain is separated into:
//   - Ancestors: the target's ancestor nodes (sorted alphabetically by logical name)
//   - Target: the target node itself
//   - Dependencies: cross-tree dependencies declared in frontmatter
//   - Code: existing generated source files declared in the target's Implements list
package chainresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
)

// ChainItem represents a single node in the chain. FilePath is the path to
// the spec file. Qualifier, when non-nil, restricts which section the caller
// should use: only the "## <qualifier>" subsection within "# Public". When
// Qualifier is nil, the caller should use the entire "# Public" section.
type ChainItem struct {
	LogicalName string
	FilePath    string
	Qualifier   *string
}

// Chain holds the fully resolved chain for a target node, separated into
// ancestors, the target itself, dependencies, and existing generated code files.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
	Code         []string
}

// ResolveChain builds the complete chain for the given target logical name.
// It returns the chain separated into Ancestors, Target, Dependencies, and Code.
// Returns an error if the chain cannot be built (e.g., unresolvable logical
// names, missing files, or frontmatter parse errors).
func ResolveChain(targetLogicalName string) (*Chain, error) {
	var chain Chain

	// -------------------------------------------------------------------
	// Step 1 — Ancestors and Target
	//
	// Walk upward from the target using ParentLogicalName, collecting every
	// logical name (including the target itself). Sort alphabetically. The
	// last item after sorting is the Target; the rest are Ancestors.
	// -------------------------------------------------------------------
	collected := []string{targetLogicalName}
	current := targetLogicalName

	for {
		hasParent, ok := logicalnames.HasParent(current)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		if !hasParent {
			break
		}
		parent, ok := logicalnames.ParentLogicalName(current)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		collected = append(collected, parent)
		current = parent
	}

	// Sort alphabetically. The deepest/most-specific name sorts last.
	sort.Strings(collected)

	// Build a ChainItem for each collected logical name. Qualifier is always
	// nil for ancestors and target (they use their full # Public section).
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

	// The last item is the Target; everything before it is Ancestors.
	chain.Target = items[len(items)-1]
	if len(items) > 1 {
		chain.Ancestors = items[:len(items)-1]
	}

	// -------------------------------------------------------------------
	// Step 2 — Dependencies
	//
	// Read the target's frontmatter. If the target is a TEST/ node, also
	// read the subject node's frontmatter. Collect all DependsOn entries
	// from both and process them together.
	// -------------------------------------------------------------------
	targetFilePath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	targetFM, err := frontmatter.ParseFrontmatter(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Accumulate all DependsOn entries to process.
	allDeps := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	allDeps = append(allDeps, targetFM.DependsOn...)

	// If the target is a TEST/ node, also include the subject (ROOT/) node's
	// DependsOn entries.
	if targetLogicalName == "TEST" || strings.HasPrefix(targetLogicalName, "TEST/") {
		subjectName, ok := logicalnames.ParentLogicalName(targetLogicalName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
		}
		subjectPath, ok := logicalnames.PathFromLogicalName(subjectName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", subjectName)
		}
		subjectFM, err := frontmatter.ParseFrontmatter(subjectPath)
		if err != nil {
			return nil, fmt.Errorf("error parsing frontmatter: %w", err)
		}
		allDeps = append(allDeps, subjectFM.DependsOn...)
	}

	// Process each DependsOn entry into a ChainItem.
	for _, dep := range allDeps {
		// Step 2.1: Resolve the file path.
		depFilePath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
		}

		// Step 2.2: Determine the qualifier from the logical name.
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

		// Step 2.3: Verify the file exists on disk.
		if _, err := os.Stat(depFilePath); err != nil {
			return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
		}

		// Step 2.4: Add the ChainItem to Dependencies.
		chain.Dependencies = append(chain.Dependencies, ChainItem{
			LogicalName: dep.LogicalName,
			FilePath:    depFilePath,
			Qualifier:   qualifier,
		})
	}

	// Sort Dependencies alphabetically by FilePath, then by Qualifier
	// (nil sorts before non-nil).
	sort.Slice(chain.Dependencies, func(i, j int) bool {
		if chain.Dependencies[i].FilePath != chain.Dependencies[j].FilePath {
			return chain.Dependencies[i].FilePath < chain.Dependencies[j].FilePath
		}
		// nil Qualifier sorts before non-nil.
		qi := chain.Dependencies[i].Qualifier
		qj := chain.Dependencies[j].Qualifier
		if qi == nil && qj != nil {
			return true
		}
		if qi != nil && qj == nil {
			return false
		}
		if qi != nil && qj != nil {
			return *qi < *qj
		}
		return false // both nil: equal
	})

	// -------------------------------------------------------------------
	// Step 3 — Code
	//
	// Extract the Implements list from the target's frontmatter and keep
	// only paths that already exist on disk.
	// -------------------------------------------------------------------
	for _, implPath := range targetFM.Implements {
		if _, err := os.Stat(implPath); err == nil {
			chain.Code = append(chain.Code, implPath)
		}
		// If the file does not exist, skip it silently (per spec).
	}

	// -------------------------------------------------------------------
	// Step 4 — Normalize file paths
	//
	// Convert all file paths to forward slashes regardless of OS.
	// -------------------------------------------------------------------
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

	// -------------------------------------------------------------------
	// Step 5 — Deduplicate
	//
	// Two entries are duplicates when they share the same FilePath and
	// Qualifier. Additionally, if an entry exists with Qualifier=nil for a
	// given FilePath, all entries with that same FilePath and a non-nil
	// Qualifier are redundant (the full # Public already covers every
	// subsection) and must be removed.
	//
	// Deduplication applies across Ancestors and Dependencies combined.
	// Keep the first occurrence.
	// -------------------------------------------------------------------

	// Build a set of FilePaths that appear with Qualifier=nil across both
	// Ancestors and Dependencies. These "full-coverage" paths make any
	// qualified entry with the same path redundant.
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

	// Helper: returns a string key for a (FilePath, Qualifier) pair.
	itemKey := func(filePath string, qualifier *string) string {
		if qualifier == nil {
			return filePath + "\x00nil"
		}
		return filePath + "\x00q:" + *qualifier
	}

	seenKeys := make(map[string]bool)

	deduplicateItems := func(items []ChainItem) []ChainItem {
		result := make([]ChainItem, 0, len(items))
		for _, item := range items {
			// If the full # Public for this path is already present (Qualifier=nil),
			// any qualified variant is redundant — skip it.
			if item.Qualifier != nil && fullCoveragePaths[item.FilePath] {
				continue
			}
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
