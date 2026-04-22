---
version: 12
parent_version: 24
implements:
  - internal/logicalnames/logicalnames_test.go
---

# TEST/tech_design/internal/logical_names

## Context

Pure function tests — no filesystem or temp directories
needed. Each test calls the function with a string input
and asserts the output.

## PathFromLogicalName

### ROOT

Input: `"ROOT"`
Expect: `"code-from-spec/spec/_node.md"`, `true`.

### ROOT with path

Input: `"ROOT/payments/processor"`
Expect: `"code-from-spec/spec/payments/processor/_node.md"`, `true`.

### TEST without path

Input: `"TEST"`
Expect: `"code-from-spec/spec/default.test.md"`, `true`.

### TEST canonical

Input: `"TEST/domain/config"`
Expect: `"code-from-spec/spec/domain/config/default.test.md"`, `true`.

### TEST named

Input: `"TEST/domain/config(edge_cases)"`
Expect: `"code-from-spec/spec/domain/config/edge_cases.test.md"`, `true`.

### EXTERNAL

Input: `"EXTERNAL/codefromspec"`
Expect: `"code-from-spec/external/codefromspec/_external.md"`, `true`.

### Unrecognized prefix

Input: `"UNKNOWN/something"`
Expect: `""`, `false`.

### Empty string

Input: `""`
Expect: `""`, `false`.

### EXTERNAL without name

Input: `"EXTERNAL"`
Expect: `""`, `false`.

## HasParent

### ROOT

Input: `"ROOT"`
Expect: `false`, `true`.

### ROOT with path

Input: `"ROOT/domain/config"`
Expect: `true`, `true`.

### TEST without path

Input: `"TEST"`
Expect: `true`, `true`.

### TEST with path

Input: `"TEST/domain/config"`
Expect: `true`, `true`.

### TEST named

Input: `"TEST/domain/config(edge_cases)"`
Expect: `true`, `true`.

### EXTERNAL

Input: `"EXTERNAL/codefromspec"`
Expect: `false`, `true`.

### EXTERNAL without name

Input: `"EXTERNAL"`
Expect: `false`, `false`.

### Empty string

Input: `""`
Expect: `false`, `false`.

### Unrecognized prefix

Input: `"UNKNOWN/something"`
Expect: `false`, `false`.

## ParentLogicalName

### ROOT/x — parent is ROOT

Input: `"ROOT/domain"`
Expect: `"ROOT"`, `true`.

### ROOT/x/y — parent is ROOT/x

Input: `"ROOT/domain/config"`
Expect: `"ROOT/domain"`, `true`.

### ROOT/x/y/z — parent is ROOT/x/y

Input: `"ROOT/tech_design/logical_names"`
Expect: `"ROOT/tech_design"`, `true`.

### TEST without path — parent is ROOT

Input: `"TEST"`
Expect: `"ROOT"`, `true`.

### TEST/x — parent is ROOT/x

Input: `"TEST/domain/config"`
Expect: `"ROOT/domain/config"`, `true`.

### TEST/x(name) — parent is ROOT/x

Input: `"TEST/domain/config(edge_cases)"`
Expect: `"ROOT/domain/config"`, `true`.

### ROOT has no parent

Input: `"ROOT"`
Expect: `""`, `false`.

### EXTERNAL has no parent

Input: `"EXTERNAL/codefromspec"`
Expect: `""`, `false`.

### Invalid input

Input: `""`
Expect: `""`, `false`.
