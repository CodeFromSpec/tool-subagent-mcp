// Package frontmatter reads and parses YAML frontmatter from spec nodes,
// test nodes, and external dependency files.
//
// spec: ROOT/tech_design/internal/frontmatter@v27
package frontmatter

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// DependsOn represents a single entry in the depends_on list.
type DependsOn struct {
	LogicalName string
	Filter      []string
}

// Frontmatter holds the parsed frontmatter fields.
// All fields are optional at the parsing level.
type Frontmatter struct {
	DependsOn  []DependsOn
	Implements []string
}

// rawDependsOn mirrors the YAML structure of a depends_on entry.
type rawDependsOn struct {
	Path   string   `yaml:"path"`
	Filter []string `yaml:"filter"`
}

// rawFrontmatter mirrors the YAML frontmatter block for unmarshalling.
// Unknown fields are silently ignored by goccy/go-yaml.
type rawFrontmatter struct {
	DependsOn  []rawDependsOn `yaml:"depends_on"`
	Implements []string       `yaml:"implements"`
}

// ParseFrontmatter reads the file at filePath, extracts the YAML frontmatter
// block (between the first and second "---" lines), parses it, and returns
// the result. It reads line by line and stops as soon as the closing "---"
// is found — the file body is never read.
func ParseFrontmatter(filePath string) (*Frontmatter, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Find the opening "---".
	foundOpen := false
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "---" {
			foundOpen = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	if !foundOpen {
		return nil, fmt.Errorf("frontmatter not found in %s", filePath)
	}

	// Collect lines until the closing "---".
	var lines []string
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClose = true
			break
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	if !foundClose {
		return nil, fmt.Errorf("frontmatter not found in %s", filePath)
	}

	// Parse the YAML content.
	yamlContent := strings.Join(lines, "\n")
	var raw rawFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter in %s: %w", filePath, err)
	}

	// Convert raw structs to the public types.
	result := &Frontmatter{
		Implements: raw.Implements,
	}
	for _, dep := range raw.DependsOn {
		result.DependsOn = append(result.DependsOn, DependsOn{
			LogicalName: dep.Path,
			Filter:      dep.Filter,
		})
	}

	return result, nil
}
