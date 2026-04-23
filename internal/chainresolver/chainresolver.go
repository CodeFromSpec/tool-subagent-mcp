// spec: ROOT/tech_design/internal/chain_resolver@v65

package chainresolver

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
)

// ChainItem represents a single node in the chain, identified by its
// logical name and the file paths that contain its content.
type ChainItem struct {
	LogicalName string
	FilePaths   []string
}

// Chain holds the fully resolved chain for a target node, separated
// into ancestors, the target itself, dependencies, and existing
// generated code files.
type Chain struct {
	Ancestors    []ChainItem
	Target       ChainItem
	Dependencies []ChainItem
	Code         []string
}

// ResolveChain builds the complete chain for the given target logical name.
// It returns the chain separated into ancestors, target, dependencies, and
// code. Ancestors and Dependencies are sorted alphabetically by logical name.
func ResolveChain(targetLogicalName string) (*Chain, error) {
	var chain Chain

	// ---------------------------------------------------------------
	// Step 1 — Ancestors and Target
	// ---------------------------------------------------------------
	// Walk upward from the target, collecting every logical name
	// (including the target itself) into a flat list.
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

	// Sort alphabetically. The last item after sorting is the Target
	// (because the target is the most specific / deepest name).
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

	// Last item is the Target; the rest are Ancestors.
	chain.Target = items[len(items)-1]
	if len(items) > 1 {
		chain.Ancestors = items[:len(items)-1]
	}

	// ---------------------------------------------------------------
	// Step 2 — Dependencies
	// ---------------------------------------------------------------
	// Parse the target node's frontmatter to get DependsOn entries.
	targetFilePath, ok := logicalnames.PathFromLogicalName(targetLogicalName)
	if !ok {
		return nil, fmt.Errorf("cannot resolve logical name: %s", targetLogicalName)
	}

	targetFM, err := frontmatter.ParseFrontmatter(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Collect all DependsOn entries. If the target is a TEST/ node,
	// also include the parent leaf node's DependsOn.
	allDeps := make([]frontmatter.DependsOn, 0, len(targetFM.DependsOn))
	allDeps = append(allDeps, targetFM.DependsOn...)

	if strings.HasPrefix(targetLogicalName, "TEST/") || targetLogicalName == "TEST" {
		// The parent of a TEST/ node is a ROOT/ leaf node.
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

	// Index for quick lookup of existing dependency items by logical name.
	depIndex := make(map[string]int)

	for _, dep := range allDeps {
		if strings.HasPrefix(dep.LogicalName, "ROOT/") || dep.LogicalName == "ROOT" {
			// ROOT/ dependency: single file path.
			depFilePath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			// Verify file exists on disk.
			if _, err := os.Stat(depFilePath); err != nil {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}
			// Skip if already present.
			if _, exists := depIndex[dep.LogicalName]; exists {
				continue
			}
			chain.Dependencies = append(chain.Dependencies, ChainItem{
				LogicalName: dep.LogicalName,
				FilePaths:   []string{depFilePath},
			})
			depIndex[dep.LogicalName] = len(chain.Dependencies) - 1

		} else if strings.HasPrefix(dep.LogicalName, "EXTERNAL/") {
			// EXTERNAL/ dependency: walk the folder, optionally filtering.
			externalPath, ok := logicalnames.PathFromLogicalName(dep.LogicalName)
			if !ok {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			// The dependency folder is the directory containing _external.md.
			depFolder := filepath.Dir(externalPath)

			// Collect files by walking the dependency folder recursively.
			var filePaths []string
			err := filepath.WalkDir(depFolder, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// Skip directories — only collect files.
				if d.IsDir() {
					return nil
				}
				filePaths = append(filePaths, p)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("cannot resolve logical name: %s", dep.LogicalName)
			}

			// Apply filter if present.
			if len(dep.Filter) > 0 {
				var filtered []string
				for _, fp := range filePaths {
					// Get path relative to the dependency folder.
					relPath, err := filepath.Rel(depFolder, fp)
					if err != nil {
						continue
					}
					// Use forward slashes for matching with path.Match.
					relPath = filepath.ToSlash(relPath)

					// Always include _external.md.
					if relPath == "_external.md" {
						filtered = append(filtered, fp)
						continue
					}

					// Check against each filter pattern using path.Match.
					for _, pattern := range dep.Filter {
						matched, matchErr := path.Match(pattern, relPath)
						if matchErr != nil {
							return nil, fmt.Errorf("error evaluating filter %s for %s: %s", pattern, dep.LogicalName, matchErr.Error())
						}
						if matched {
							filtered = append(filtered, fp)
							break
						}
					}
				}
				filePaths = filtered
			}

			// Sort file paths.
			sort.Strings(filePaths)

			// If a ChainItem with the same logical name exists, merge paths.
			if idx, exists := depIndex[dep.LogicalName]; exists {
				chain.Dependencies[idx].FilePaths = append(chain.Dependencies[idx].FilePaths, filePaths...)
			} else {
				chain.Dependencies = append(chain.Dependencies, ChainItem{
					LogicalName: dep.LogicalName,
					FilePaths:   filePaths,
				})
				depIndex[dep.LogicalName] = len(chain.Dependencies) - 1
			}
		}
	}

	// Sort Dependencies by logical name alphabetically.
	sort.Slice(chain.Dependencies, func(i, j int) bool {
		return chain.Dependencies[i].LogicalName < chain.Dependencies[j].LogicalName
	})

	// ---------------------------------------------------------------
	// Step 3 — Code
	// ---------------------------------------------------------------
	// Extract Implements from the target frontmatter and keep only
	// files that exist on disk.
	for _, implPath := range targetFM.Implements {
		if _, err := os.Stat(implPath); err == nil {
			chain.Code = append(chain.Code, implPath)
		}
	}

	// ---------------------------------------------------------------
	// Step 4 — Normalize file paths
	// ---------------------------------------------------------------
	// Convert all file paths to forward slashes.
	normalizeItem := func(item *ChainItem) {
		for i, fp := range item.FilePaths {
			item.FilePaths[i] = filepath.ToSlash(fp)
		}
	}

	for i := range chain.Ancestors {
		normalizeItem(&chain.Ancestors[i])
	}
	normalizeItem(&chain.Target)
	for i := range chain.Dependencies {
		normalizeItem(&chain.Dependencies[i])
	}
	for i, fp := range chain.Code {
		chain.Code[i] = filepath.ToSlash(fp)
	}

	// ---------------------------------------------------------------
	// Step 5 — Deduplicate file paths
	// ---------------------------------------------------------------
	// Each file path must appear only once across Ancestors and
	// Dependencies. Keep the first occurrence, discard subsequent ones.
	seen := make(map[string]bool)

	// Process Ancestors first (they come before Dependencies).
	for i := range chain.Ancestors {
		var deduped []string
		for _, fp := range chain.Ancestors[i].FilePaths {
			if !seen[fp] {
				seen[fp] = true
				deduped = append(deduped, fp)
			}
		}
		chain.Ancestors[i].FilePaths = deduped
	}

	// Then process Dependencies.
	for i := range chain.Dependencies {
		var deduped []string
		for _, fp := range chain.Dependencies[i].FilePaths {
			if !seen[fp] {
				seen[fp] = true
				deduped = append(deduped, fp)
			}
		}
		chain.Dependencies[i].FilePaths = deduped
	}

	return &chain, nil
}
