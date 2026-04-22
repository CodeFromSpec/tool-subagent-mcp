# google/uuid — API used

## Import

```go
import "github.com/google/uuid"
```

## New

```go
func New() UUID
```

Generates a new random UUID (version 4). Panics if the
random source fails — safe to call without error checking
in normal conditions.

```go
id := uuid.New()
fmt.Println(id.String()) // e.g. "550e8400-e29b-41d4-a716-446655440000"
```

## String

```go
func (uuid UUID) String() string
```

Returns the UUID in standard hyphenated format:
`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`.
