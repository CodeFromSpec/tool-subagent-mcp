// spec: ROOT/tech_design/internal/chain_resolver@v62

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

// ChainItem represents a single node in the chain with its logical name
// and the file paths that contribute its content.
type ChainItem struct {
	LogicalName string
	FilePaths   []string
}

// Chain holds the fully resolved chain for a target node, separated into
// ancestors (root to parent), the target itself, its dependencies, and
// any existing generated source files.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
	Code         []string
}

// ResolveChain builds the complete chain for the given target logical name.
// It returns the chain separated into ancestors, target, dependencies, and
// existing code files. Ancestors and Dependencies are sorted alphabetically
// by logical name.
func ResolveChain(targetLogicalName string) (*Chain, error) {
	// ---------------------------------------------------------------
	// Step 1 — Ancestors and Target
	// ---------------------------------------------------------------
	// Walk upward from the target, collecting all logical names in the
	// ancestry path (including the target itself).
	allNames := []string{targetLogicalName}

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
		allNames = append(allNames, parent)
		current = parent
	}

	// Sort alphabetically — the target will be the last item because
	// it is the most deeply nested (longest) name.
	sort.Strings(allNames)

	// Build ChainItems for each name in the sorted list.
	allItems := make([]ChainItem, 0, len(allNames))
	for _, name := range allNames {
		filePath, ok := logicalnames.PathFromLogicalName(name)
		if !ok {
			return nil, fmt.Errorf("cannot resolve logical name: %s", name)
		}
		allItems = append(allItems, ChainItem{
			LogicalName: name,
			FilePaths:   []string{filePath},
		})
	}

	// The last item is the target; the rest are ancestors.
	chain := &Chain{
		Target: allItems[len(allItems)-1],
	}
	if len(allItems) > 1 {
		chain.Ancestors = allItems[:len(allItems)-1]
	}

	// ---------------------------------------------------------------
	// Step 2 — Dependencies
	// ---------------------------------------------------------------
	// Parse the target's frontmatter to get depends_on entries.
	targetPath, _ := logicalnames.PathFromLogicalName(targetLogicalName)
	targetFM, err := frontmatter.ParseFrontmatter(targetPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Collect all DependsOn entries. If the target is a TEST/ node,
	// also include the parent leaf node's depends_on.
	allDeps := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	allDeps = append(allDeps, targetFM.DependsOn...)

	if strings.HasPrefix(targetLogicalName, "TEST/") || targetLogicalName == "TEST" {
		// The parent of a TEST node is a ROOT node (the leaf it tests).
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
	// Use a map to track existing dependency items by logical name for
	// deduplication and merging.
	depMap := make(map[string]*ChainItem)
	var depOrder []string // Track insertion order for later sorting.

	for _, dep := range allDeps {
		if strings.HasPrefix(dep.LogicalName, "ROOT/") || dep.LogicalName == "ROOT" {
			// ---- ROOT dependency ----
			// Skip if we already have this logical name.
			if _, exists := depMap[dep.LogicalName]; exists {
				continue
			}

			filePath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			// Verify the file exists on disk.
			if _, err := os.Stat(filePath); err != nil {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			depMap[dep.LogicalName] = &ChainItem{
				LogicalName: dep.LogicalName,
				FilePaths:   []string{filePath},
			}
			depOrder = append(depOrder, dep.LogicalName)

		} else if strings.HasPrefix(dep.LogicalName, "EXTERNAL/") {
			// ---- EXTERNAL dependency ----
			externalMdPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			// The dependency folder is the directory containing _external.md.
			depFolder := filepath.Dir(externalMdPath)

			// Collect file paths: always include _external.md first.
			var filePaths []string
			filePaths = append(filePaths, externalMdPath)

			if len(dep.Filter) > 0 {
				// Include _external.md plus files matching any filter pattern.
				for _, pattern := range dep.Filter {
					// Pattern is relative to the dependency folder.
					fullPattern := filepath.Join(depFolder, pattern)
					matches, err := filepath.Glob(fullPattern)
					if err != nil {
						return nil, fmt.Errorf("error evaluating filter %s for %s: %s", pattern, dep.LogicalName, err.Error())
					}
					for _, m := range matches {
						filePaths = append(filePaths, m)
					}
				}
			} else {
				// No filter — include all files in the dependency folder
				// and subfolders.
				err := filepath.Walk(depFolder, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						filePaths = append(filePaths, path)
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("error evaluating filter * for %s: %s", dep.LogicalName, err.Error())
				}
			}

			// Sort and deduplicate file paths within this dependency.
			sort.Strings(filePaths)
			filePaths = deduplicateStrings(filePaths)

			// Merge into existing item or create new one.
			if existing, exists := depMap[dep.LogicalName]; exists {
				existing.FilePaths = mergeAndSortPaths(existing.FilePaths, filePaths)
			} else {
				depMap[dep.LogicalName] = &ChainItem{
					LogicalName: dep.LogicalName,
					FilePaths:   filePaths,
				}
				depOrder = append(depOrder, dep.LogicalName)
			}
		}
	}

	// Sort dependencies alphabetically by logical name.
	sort.Strings(depOrder)
	chain.Dependencies = make([]ChainItem, 0, len(depOrder))
	for _, name := range depOrder {
		chain.Dependencies = append(chain.Dependencies, *depMap[name])
	}

	// ---------------------------------------------------------------
	// Step 3 — Code
	// ---------------------------------------------------------------
	// Extract implements list from the target's frontmatter and keep
	// only files that exist on disk.
	for _, implPath := range targetFM.Implements {
		if _, err := os.Stat(implPath); err == nil {
			chain.Code = append(chain.Code, implPath)
		}
	}

	// ---------------------------------------------------------------
	// Step 4 — Normalize file paths
	// ---------------------------------------------------------------
	// Convert all paths to forward slashes regardless of OS.
	for i := range chain.Ancestors {
		for j := range chain.Ancestors[i].FilePaths {
			chain.Ancestors[i].FilePaths[j] = filepath.ToSlash(chain.Ancestors[i].FilePaths[j])
		}
	}
	for j := range chain.Target.FilePaths {
		chain.Target.FilePaths[j] = filepath.ToSlash(chain.Target.FilePaths[j])
	}
	for i := range chain.Dependencies {
		for j := range chain.Dependencies[i].FilePaths {
			chain.Dependencies[i].FilePaths[j] = filepath.ToSlash(chain.Dependencies[i].FilePaths[j])
		}
	}
	for i := range chain.Code {
		chain.Code[i] = filepath.ToSlash(chain.Code[i])
	}

	// ---------------------------------------------------------------
	// Step 5 — Deduplicate file paths across Ancestors and Dependencies
	// ---------------------------------------------------------------
	// Each file path must appear only once across the entire chain.
	// Keep the first occurrence and discard subsequent ones.
	seen := make(map[string]bool)

	// Process Ancestors first (they come before Dependencies).
	for i := range chain.Ancestors {
		chain.Ancestors[i].FilePaths = filterSeen(chain.Ancestors[i].FilePaths, seen)
	}

	// Then Dependencies.
	for i := range chain.Dependencies {
		chain.Dependencies[i].FilePaths = filterSeen(chain.Dependencies[i].FilePaths, seen)
	}

	return chain, nil
}

// filterSeen removes paths that have already been seen, updating the seen
// map with paths that are kept. Returns the filtered slice.
func filterSeen(paths []string, seen map[string]bool) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}

// deduplicateStrings removes consecutive duplicate strings from a sorted slice.
func deduplicateStrings(s []string) []string {
	if len(s) <= 1 {
		return s
	}
	result := []string{s[0]}
	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1] {
			result = append(result, s[i])
		}
	}
	return result
}

// mergeAndSortPaths merges two sorted path slices, sorts the result, and
// removes duplicates.
func mergeAndSortPaths(a, b []string) []string {
	merged := make([]string, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)
	sort.Strings(merged)
	return deduplicateStrings(merged)
}
