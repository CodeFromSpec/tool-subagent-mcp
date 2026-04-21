// spec: ROOT/tech_design/internal/chain_resolver@v45

// Package chainresolver resolves the ordered list of files that form the
// context chain for a given target logical name. The chain is divided into
// three groups: ancestor spec nodes (root → parent), the target node itself,
// and cross-tree / external dependencies declared in the target's frontmatter.
package chainresolver

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
)

// ChainItem pairs a logical name with the set of file paths it contributes to
// the chain. Most items have a single path; EXTERNAL/ dependencies may have
// many (the _external.md plus supporting files).
type ChainItem struct {
	LogicalName string
	FilePaths   []string
}

// Chain is the complete, ordered context for a code generation subagent. It
// mirrors the structure described in the Code from Spec methodology:
//   - Ancestors: all spec nodes from ROOT down to (but not including) Target,
//     sorted alphabetically by logical name.
//   - Target: the leaf node (or test node) being implemented.
//   - Dependencies: cross-tree (ROOT/) and external (EXTERNAL/) depends_on
//     entries declared by the target, sorted alphabetically by logical name.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
}

// ResolveChain builds the complete chain for the given target logical name.
//
// Step 1 – Ancestors and Target:
//   Walk up the tree via ParentLogicalName, collecting every logical name from
//   the target to ROOT. Sort alphabetically — because logical names use path
//   notation (ROOT/a/b) alphabetical order is top-down. The last item in the
//   sorted slice is the Target; the remaining items are Ancestors.
//
// Step 2 – Dependencies:
//   Read the target node's frontmatter. If the target is a TEST/ node, also
//   read the parent leaf node's frontmatter and merge their DependsOn lists.
//   Resolve each entry to a ChainItem and sort by logical name.
//
// Returns an error if any logical name cannot be resolved, frontmatter cannot
// be parsed, or a glob pattern fails to evaluate.
func ResolveChain(targetLogicalName string) (*Chain, error) {
	// -------------------------------------------------------------------------
	// Step 1 — Walk from target up to ROOT, collecting all logical names.
	// -------------------------------------------------------------------------

	// allNames collects the target plus all its ancestors.
	var allNames []string
	current := targetLogicalName

	for {
		allNames = append(allNames, current)

		hasParent, ok := logicalnames.HasParent(current)
		if !ok {
			// Not a valid logical name — bail out early.
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		if !hasParent {
			// Reached a root-level node (ROOT or EXTERNAL/<name>); stop.
			break
		}

		parent, ok := logicalnames.ParentLogicalName(current)
		if !ok {
			// HasParent said true but ParentLogicalName failed — defensive guard.
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		current = parent
	}

	// Sort alphabetically. For ROOT/ names this yields top-down order because
	// shorter names (ROOT, ROOT/a) sort before deeper names (ROOT/a/b).
	sort.Strings(allNames)

	// Convert each logical name to a ChainItem (single file path).
	items := make([]ChainItem, 0, len(allNames))
	for _, name := range allNames {
		path, ok := logicalnames.PathFromLogicalName(name)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", name)
		}
		items = append(items, ChainItem{
			LogicalName: name,
			FilePaths:   []string{path},
		})
	}

	// The last item after sorting is the deepest / target node.
	target := items[len(items)-1]
	ancestors := items[:len(items)-1]

	// -------------------------------------------------------------------------
	// Step 2 — Resolve Dependencies from frontmatter.
	// -------------------------------------------------------------------------

	// Determine the file path of the target node for frontmatter parsing.
	targetPath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	// Parse the target node's frontmatter.
	targetFM, err := frontmatter.ParseFrontmatter(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolving chain for %s: %w", targetLogicalName, err)
	}

	// Collect all DependsOn entries to process.
	dependsOnEntries := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	dependsOnEntries = append(dependsOnEntries, targetFM.DependsOn...)

	// If the target is a TEST/ node, also pull in the parent leaf's DependsOn.
	// The spec says the leaf node's depends_on is implicitly included in the
	// test chain — the test node "depends fundamentally on its test subject."
	if strings.HasPrefix(targetLogicalName, "TEST/") || targetLogicalName == "TEST" {
		// ParentLogicalName for a TEST/ node yields the ROOT/ leaf node.
		parentName, ok := logicalnames.ParentLogicalName(targetLogicalName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
		}
		parentPath, ok := logicalnames.PathFromLogicalName(parentName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", parentName)
		}
		parentFM, err := frontmatter.ParseFrontmatter(parentPath)
		if err != nil {
			return nil, fmt.Errorf("resolving chain for %s (parent leaf): %w", targetLogicalName, err)
		}
		dependsOnEntries = append(dependsOnEntries, parentFM.DependsOn...)
	}

	// Process each DependsOn entry into a ChainItem.
	var dependencies []ChainItem
	for _, dep := range dependsOnEntries {
		if strings.HasPrefix(dep.LogicalName, "ROOT/") {
			// Cross-tree spec dependency — single _node.md file.
			depPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			dependencies = append(dependencies, ChainItem{
				LogicalName: dep.LogicalName,
				FilePaths:   []string{depPath},
			})

		} else if strings.HasPrefix(dep.LogicalName, "EXTERNAL/") {
			// External dependency — _external.md plus supporting files.
			item, err := resolveExternalDep(dep)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, item)
		}
		// Other prefixes (e.g., bare ROOT, TEST/) are not valid depends_on
		// targets per spec — silently skip them.
	}

	// Sort dependencies alphabetically by logical name.
	sort.Slice(dependencies, func(i, j int) bool {
		return dependencies[i].LogicalName < dependencies[j].LogicalName
	})

	return &Chain{
		Ancestors:    ancestors,
		Target:       target,
		Dependencies: dependencies,
	}, nil
}

// resolveExternalDep builds a ChainItem for an EXTERNAL/ dependency entry.
//
// The _external.md is always included. When a Filter is present, only files in
// the dependency folder matching any pattern are added (additive / OR logic).
// When no Filter is present, all files in the folder and subfolders are added.
// File paths are sorted and relative to the project root.
func resolveExternalDep(dep frontmatter.DependsOn) (ChainItem, error) {
	// PathFromLogicalName for EXTERNAL/x → code-from-spec/external/x/_external.md
	externalMDPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
	if !ok {
		return ChainItem{}, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
	}

	// The dependency folder is the directory containing _external.md.
	depFolder := filepath.Dir(externalMDPath)

	var filePaths []string

	// Always include _external.md.
	filePaths = append(filePaths, filepath.ToSlash(externalMDPath))

	if len(dep.Filter) == 0 {
		// No filter — include every file in the dependency folder recursively.
		err := filepath.WalkDir(depFolder, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil // descend into subdirectories
			}
			// Normalize separators to forward slashes for consistency.
			normalized := filepath.ToSlash(path)
			// Skip _external.md — already added above.
			if normalized == filepath.ToSlash(externalMDPath) {
				return nil
			}
			filePaths = append(filePaths, normalized)
			return nil
		})
		if err != nil {
			return ChainItem{}, fmt.Errorf("error walking external dependency %s: %w", dep.LogicalName, err)
		}
	} else {
		// Filter mode — evaluate each glob pattern relative to the dep folder.
		// A file matching ANY pattern is included (additive / OR logic).
		matched := make(map[string]struct{})
		for _, pattern := range dep.Filter {
			// Construct an absolute-ish glob by joining depFolder with pattern.
			fullPattern := filepath.Join(depFolder, pattern)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return ChainItem{}, fmt.Errorf("error evaluating filter %s for %s: %w",
					pattern, dep.LogicalName, err)
			}
			for _, m := range matches {
				normalized := filepath.ToSlash(m)
				// Skip _external.md — already added.
				if normalized == filepath.ToSlash(externalMDPath) {
					continue
				}
				matched[normalized] = struct{}{}
			}
		}
		for p := range matched {
			filePaths = append(filePaths, p)
		}
	}

	// Sort file paths so the output is deterministic and consistent.
	sort.Strings(filePaths)

	return ChainItem{
		LogicalName: dep.LogicalName,
		FilePaths:   filePaths,
	}, nil
}
