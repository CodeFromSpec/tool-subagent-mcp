---
version: 19
subject_version: 33
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
Expect: `"code-from-spec/_node.md"`, `true`.

### ROOT with path

Input: `"ROOT/payments/processor"`
Expect: `"code-from-spec/payments/processor/_node.md"`, `true`.

### ROOT with qualifier

Input: `"ROOT/payments/processor(interface)"`
Expect: `"code-from-spec/payments/processor/_node.md"`, `true`.

### ROOT with qualifier — strips qualifier from path

Input: `"ROOT/x(y)"`
Expect: `"code-from-spec/x/_node.md"`, `true`.

### TEST without path

Input: `"TEST"`
Expect: `"code-from-spec/default.test.md"`, `true`.

### TEST canonical

Input: `"TEST/domain/config"`
Expect: `"code-from-spec/domain/config/default.test.md"`, `true`.

### TEST named

Input: `"TEST/domain/config(edge_cases)"`
Expect: `"code-from-spec/domain/config/edge_cases.test.md"`, `true`.

### Unrecognized prefix

Input: `"UNKNOWN/something"`
Expect: `""`, `false`.

### Empty string

Input: `""`
Expect: `""`, `false`.

## HasParent

### ROOT

Input: `"ROOT"`
Expect: `false`, `true`.

### ROOT with path

Input: `"ROOT/domain/config"`
Expect: `true`, `true`.

### ROOT with qualifier

Input: `"ROOT/domain/config(interface)"`
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

### ROOT/x/y(z) — parent is ROOT/x

Input: `"ROOT/domain/config(interface)"`
Expect: `"ROOT/domain"`, `true`.

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

### Invalid input

Input: `""`
Expect: `""`, `false`.

## HasQualifier

### ROOT without qualifier

Input: `"ROOT/x"`
Expect: `false`, `true`.

### ROOT with qualifier

Input: `"ROOT/x(y)"`
Expect: `true`, `true`.

### ROOT with nested path and qualifier

Input: `"ROOT/x/y/z(w)"`
Expect: `true`, `true`.

### ROOT alone

Input: `"ROOT"`
Expect: `false`, `true`.

### TEST without qualifier

Input: `"TEST/x"`
Expect: `false`, `true`.

### TEST with qualifier

Input: `"TEST/x(edge_cases)"`
Expect: `true`, `true`.

### Empty string

Input: `""`
Expect: `false`, `false`.

### Unrecognized prefix

Input: `"UNKNOWN/x(y)"`
Expect: `false`, `false`.

## QualifierName

### ROOT with qualifier

Input: `"ROOT/x(y)"`
Expect: `"y"`, `true`.

### ROOT with nested path and qualifier

Input: `"ROOT/x/y(interface)"`
Expect: `"interface"`, `true`.

### TEST with qualifier

Input: `"TEST/x(edge_cases)"`
Expect: `"edge_cases"`, `true`.

### ROOT without qualifier

Input: `"ROOT/x"`
Expect: `""`, `false`.

### ROOT alone

Input: `"ROOT"`
Expect: `""`, `false`.

### Empty string

Input: `""`
Expect: `""`, `false`.
