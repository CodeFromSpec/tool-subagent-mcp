---
version: 2
parent_version: 1
---

# ROOT/external/goccy-go-yaml

YAML parsing library for Go: `github.com/goccy/go-yaml`.

Drop-in replacement for the archived `gopkg.in/yaml.v3`.

# Public

## Import

```go
import "github.com/goccy/go-yaml"
```

## Unmarshal

```go
func Unmarshal(data []byte, v interface{}) error
```

Parses YAML bytes into a Go struct. Struct fields use
`yaml:"name"` tags to map YAML keys.

```go
type Config struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

var cfg Config
if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
    // handle error
}
```

Unknown fields are silently ignored by default.
