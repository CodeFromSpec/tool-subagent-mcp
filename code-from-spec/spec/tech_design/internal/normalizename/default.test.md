---
version: 2
parent_version: 2
implements:
  - internal/normalizename/normalizename_test.go
---

# TEST/tech_design/internal/normalizename

## Context

Pure function tests — no filesystem or temp directories needed.

## Identity

### Already normalized

Input: `"public"`
Expect: `"public"`.

### Single word

Input: `"Interface"`
Expect: `"interface"`.

## Trim

### Leading and trailing spaces

Input: `"  Interface  "`
Expect: `"interface"`.

### Leading and trailing tabs

Input: `"\tInterface\t"`
Expect: `"interface"`.

### Mixed leading whitespace

Input: `" \t Interface \t "`
Expect: `"interface"`.

## Collapse

### Multiple spaces between words

Input: `"Testes   de   aceitação"`
Expect: `"testes de aceitação"`.

### Tabs between words

Input: `"Testes\tde\taceitação"`
Expect: `"testes de aceitação"`.

### Mixed whitespace between words

Input: `"Testes \t de \t aceitação"`
Expect: `"testes de aceitação"`.

## Case folding

### All uppercase

Input: `"PUBLIC"`
Expect: `"public"`.

### Mixed case

Input: `"PuBLiC"`
Expect: `"public"`.

### Unicode case folding

Input: `"TESTES DE ACEITAÇÃO"`
Expect: `"testes de aceitação"`.

### German sharp s

Input: `"Straße"`
Expect: `"strasse"` (Unicode simple case folding maps `ß` to `ss`).

## Combined

### Trim, collapse, and case fold together

Input: `"  TESTES   DE   ACEITAÇÃO  "`
Expect: `"testes de aceitação"`.

### Logical name qualifier style

Input: `"testes de ACEITAÇÃO"`
Expect: `"testes de aceitação"`.

### Tabs and mixed case

Input: `"\tROOT/payments/fees\t"`
Expect: `"root/payments/fees"`.

## Edge cases

### Empty string

Input: `""`
Expect: `""`.

### Only whitespace

Input: `"   \t  "`
Expect: `""`.

### Non-breaking space is not whitespace

Input: `"hello world"`
Expect: `"hello world"` (U+00A0 is treated as text,
not collapsed).

### Single character

Input: `"X"`
Expect: `"x"`.
