// spec: ROOT/tech_design/internal/chain_resolver@v53

// Package chainresolver resolves the ordered list of files that form
// the chain for a given target logical name. The chain consists of
// ancestors (root to parent), the target itself, and any dependencies
// declared via depends_on in the target's frontmatter.
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

// ChainItem represents a single node in the chain, identified by its
// logical name and the file path(s) that contain its content.
type ChainItem struct {
	LogicalName string
	FilePaths   []string
}

// Chain holds the resolved chain separated into three groups:
// Ancestors (root to parent), Target, and Dependencies.
// Ancestors and Dependencies are sorted by logical name alphabetically.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
}

// ResolveChain builds the complete chain for the given target logical name.
// It walks upward to collect ancestors, resolves the target, and processes
// all dependencies declared in the target's frontmatter (and the parent
// leaf's frontmatter for TEST/ nodes).
func ResolveChain(targetLogicalName string) (*Chain, error) {
	// Step 1 — Ancestors and Target
	//
	// Walk upward from the target, collecting all logical names in the
	// ancestor chain (including the target itself). Sort alphabetically,
	// then split: last item is the target, everything else is ancestors.

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

	// Sort all collected logical names alphabetically.
	sort.Strings(collected)

	// Build ChainItems for each collected logical name.
	items := make([]ChainItem, 0, len(collected))
	for _, name := range collected {
		filePath, ok := logicalnames.PathFromLogicalName(name)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", name)
		}
		items = append(items, ChainItem{
			LogicalName: name,
			FilePaths:   []string{filePath},
		})
	}

	// The last item in the sorted list is the Target; the rest are Ancestors.
	chain := &Chain{
		Target: items[len(items)-1],
	}
	if len(items) > 1 {
		chain.Ancestors = items[:len(items)-1]
	}

	// Step 2 — Dependencies
	//
	// Read the target node's frontmatter to get depends_on entries.
	// For TEST/ nodes, also read the parent leaf node's frontmatter
	// and combine all depends_on entries.

	targetFilePath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	targetFM, err := frontmatter.ParseFrontmatter(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Collect all DependsOn entries to process.
	allDeps := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	allDeps = append(allDeps, targetFM.DependsOn...)

	// If this is a TEST/ node, also include the parent leaf node's dependencies.
	if strings.HasPrefix(targetLogicalName, "TEST/") || targetLogicalName == "TEST" {
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
			return nil, fmt.Errorf("error parsing frontmatter: %w", err)
		}
		allDeps = append(allDeps, parentFM.DependsOn...)
	}

	// Process each dependency entry.
	var dependencies []ChainItem
	for _, dep := range allDeps {
		if strings.HasPrefix(dep.LogicalName, "ROOT/") || dep.LogicalName == "ROOT" {
			// ROOT dependency: resolve to single file path and verify existence.
			item, err := resolveRootDependency(dep)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, item)
		} else if strings.HasPrefix(dep.LogicalName, "EXTERNAL/") {
			// EXTERNAL dependency: resolve _external.md plus folder contents.
			item, err := resolveExternalDependency(dep)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, item)
		}
	}

	// Sort dependencies by logical name alphabetically.
	sort.Slice(dependencies, func(i, j int) bool {
		return dependencies[i].LogicalName < dependencies[j].LogicalName
	})

	chain.Dependencies = dependencies
	return chain, nil
}

// resolveRootDependency resolves a ROOT/ dependency to a ChainItem.
// It verifies that the resolved file exists on disk.
func resolveRootDependency(dep frontmatter.DependsOn) (ChainItem, error) {
	filePath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
	if !ok {
		return ChainItem{}, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
	}

	// Verify the file exists on disk.
	if _, err := os.Stat(filePath); err != nil {
		return ChainItem{}, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
	}

	return ChainItem{
		LogicalName: dep.LogicalName,
		FilePaths:   []string{filePath},
	}, nil
}

// resolveExternalDependency resolves an EXTERNAL/ dependency to a ChainItem.
// It always includes _external.md, plus additional files from the dependency
// folder (filtered by glob patterns if a filter is specified, or all files
// if no filter is present).
func resolveExternalDependency(dep frontmatter.DependsOn) (ChainItem, error) {
	// Get the _external.md path.
	externalMdPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
	if !ok {
		return ChainItem{}, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
	}

	// The dependency folder is the directory containing _external.md.
	depFolder := filepath.Dir(externalMdPath)

	// Collect all files in the dependency folder and subfolders.
	var allFiles []string
	err := filepath.WalkDir(depFolder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Normalize to forward slashes for consistent path handling.
		allFiles = append(allFiles, filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		return ChainItem{}, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
	}

	var filePaths []string
	externalMdSlash := filepath.ToSlash(externalMdPath)
	depFolderSlash := filepath.ToSlash(depFolder)

	if len(dep.Filter) == 0 {
		// No filter: include all files in the dependency folder.
		filePaths = allFiles
	} else {
		// With filter: always include _external.md, plus files matching
		// any of the glob patterns. Patterns are relative to the
		// dependency folder.
		filePaths = append(filePaths, externalMdSlash)

		for _, file := range allFiles {
			// Skip _external.md since we already added it.
			if file == externalMdSlash {
				continue
			}

			// Get the path relative to the dependency folder for matching.
			relPath := strings.TrimPrefix(file, depFolderSlash+"/")

			// Check if this file matches any filter pattern.
			matched := false
			for _, pattern := range dep.Filter {
				m, err := filepath.Match(pattern, relPath)
				if err != nil {
					return ChainItem{}, fmt.Errorf(
						"error evaluating filter %s for %s: %s",
						pattern, dep.LogicalName, err,
					)
				}
				if m {
					matched = true
					break
				}
			}
			if matched {
				filePaths = append(filePaths, file)
			}
		}
	}

	// Sort all file paths for deterministic output.
	sort.Strings(filePaths)

	return ChainItem{
		LogicalName: dep.LogicalName,
		FilePaths:   filePaths,
	}, nil
}
