// spec: ROOT/tech_design/internal/chain_resolver@v53

// Package chainresolver resolves the ordered list of files that form the chain
// for a given target logical name.
//
// Spec ref: ROOT/tech_design/internal/chain_resolver § "Intent"
//
// The chain is split into three parts:
//   - Ancestors: ancestor nodes from ROOT down to (but not including) the target.
//   - Target: the target node itself.
//   - Dependencies: nodes/files referenced via depends_on in the target's frontmatter.
package chainresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
)

// ChainItem represents one node in the chain, identified by its logical name
// and associated file paths.
//
// Spec ref: ROOT/tech_design/internal/chain_resolver § "Types"
type ChainItem struct {
	LogicalName string
	FilePaths   []string
}

// Chain holds the resolved chain for a target logical name, separated into
// ancestors, the target itself, and its dependencies.
//
// Spec ref: ROOT/tech_design/internal/chain_resolver § "Types"
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
}

// ResolveChain builds the full chain for the given targetLogicalName.
//
// The algorithm is:
//  1. Walk upward via ParentLogicalName to collect all ancestor logical names
//     plus the target itself, then sort alphabetically. The last entry is the
//     Target; the rest are Ancestors.
//  2. Read the target's frontmatter (and, for TEST/ nodes, also the parent leaf
//     node's frontmatter) to collect depends_on entries, then build Dependencies.
//
// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm"
func ResolveChain(targetLogicalName string) (*Chain, error) {
	// -------------------------------------------------------------------------
	// Step 1 — Ancestors and Target
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 1"
	// -------------------------------------------------------------------------

	// Collect the target and all its ancestor logical names by walking upward.
	var allNames []string
	current := targetLogicalName
	for {
		allNames = append(allNames, current)

		hasParent, ok := logicalnames.HasParent(current)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		if !hasParent {
			break
		}

		parent, ok := logicalnames.ParentLogicalName(current)
		if !ok {
			// HasParent returned true but ParentLogicalName failed — should not happen.
			return nil, fmt.Errorf("cannot resolve logical name: %s", current)
		}
		current = parent
	}

	// Sort alphabetically so that ROOT sorts before ROOT/x, which sorts before ROOT/x/y.
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 1"
	sort.Strings(allNames)

	// Build ChainItems for each name in sorted order.
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

	// The last item (alphabetically last) is the Target; the rest are Ancestors.
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 1"
	target := items[len(items)-1]
	ancestors := items[:len(items)-1]

	// -------------------------------------------------------------------------
	// Step 2 — Dependencies
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2"
	// -------------------------------------------------------------------------

	// Resolve the file path for the target so we can parse its frontmatter.
	targetFilePath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	targetFM, err := frontmatter.ParseFrontmatter(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("error resolving dependencies: %w", err)
	}

	// For TEST/ nodes, also read the parent leaf node's frontmatter and merge
	// its depends_on entries with the target's.
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2"
	allDependsOn := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	allDependsOn = append(allDependsOn, targetFM.DependsOn...)

	if strings.HasPrefix(targetLogicalName, "TEST/") || targetLogicalName == "TEST" {
		parentName, ok := logicalnames.ParentLogicalName(targetLogicalName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
		}
		parentFilePath, ok := logicalnames.PathFromLogicalName(parentName)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", parentName)
		}
		parentFM, err := frontmatter.ParseFrontmatter(parentFilePath)
		if err != nil {
			return nil, fmt.Errorf("error resolving dependencies: %w", err)
		}
		allDependsOn = append(allDependsOn, parentFM.DependsOn...)
	}

	// Process each depends_on entry and build the Dependencies list.
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2"
	var deps []ChainItem
	for _, dep := range allDependsOn {
		switch {
		case strings.HasPrefix(dep.LogicalName, "ROOT/"):
			// Internal spec dependency — resolve path and verify it exists on disk.
			// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2 / ROOT"
			depPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			if _, err := os.Stat(depPath); err != nil {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			deps = append(deps, ChainItem{
				LogicalName: dep.LogicalName,
				FilePaths:   []string{depPath},
			})

		case strings.HasPrefix(dep.LogicalName, "EXTERNAL/"):
			// External dependency — always include _external.md plus matching files.
			// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2 / EXTERNAL"
			externalMDPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			// The dependency folder is the directory containing _external.md.
			depFolder := filepath.Dir(externalMDPath)

			var collectedPaths []string

			if len(dep.Filter) > 0 {
				// With filters: include _external.md plus files matching any pattern.
				// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2 / EXTERNAL / filter"
				collectedPaths = append(collectedPaths, externalMDPath)
				seen := map[string]struct{}{externalMDPath: {}}
				for _, pattern := range dep.Filter {
					// Patterns are relative to the dependency folder.
					fullPattern := filepath.Join(depFolder, pattern)
					matches, err := filepath.Glob(fullPattern)
					if err != nil {
						return nil, fmt.Errorf(
							"error evaluating filter %s for %s: %s",
							pattern, dep.LogicalName, err,
						)
					}
					for _, m := range matches {
						// Normalize to forward slashes for consistency.
						normalized := filepath.ToSlash(m)
						if _, alreadySeen := seen[normalized]; !alreadySeen {
							seen[normalized] = struct{}{}
							collectedPaths = append(collectedPaths, normalized)
						}
					}
				}
			} else {
				// No filter: include all files in the dependency folder and subfolders.
				// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2 / EXTERNAL / no filter"
				err := filepath.Walk(depFolder, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						collectedPaths = append(collectedPaths, filepath.ToSlash(path))
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
				}
			}

			// Sort file paths alphabetically (relative to project root).
			// Spec ref: ROOT/tech_design/internal/chain_resolver § "Algorithm / Step 2 / EXTERNAL"
			sort.Strings(collectedPaths)

			deps = append(deps, ChainItem{
				LogicalName: dep.LogicalName,
				FilePaths:   collectedPaths,
			})
		}
		// Entries with other prefixes are silently ignored.
	}

	// Sort Dependencies by logical name alphabetically.
	// Spec ref: ROOT/tech_design/internal/chain_resolver § "Types / ResolveChain"
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].LogicalName < deps[j].LogicalName
	})

	return &Chain{
		Ancestors:    ancestors,
		Target:       target,
		Dependencies: deps,
	}, nil
}
