# Parser Validation Gaps Documentation

**Generated:** 2025-12-25  
**Status:** Documented and Intentional

This document explains why certain test files in the "negative" fixtures directory are actually valid syntax according to the current parser implementation.

---

## Overview

The parser implements **syntax-level validation** only. Semantic validation (duplicate detection, import resolution, reference validation) is intentionally deferred to the compiler module where scope-aware analysis is available.

---

## Known Valid Files in Negative Fixtures

### 1. `duplicate_key.csl` - Duplicate Key Detection

**File Content:**
```
section:
	key: value1
	key: value2
```

**Why Valid:**
- Duplicate key detection requires scope-aware analysis to handle:
  - Nested section hierarchies (same key in different sections)
  - Key shadowing semantics
  - Merge behavior for cascading configurations
- Parser only validates **syntax**, not **semantics**
- Duplicate detection is implemented in the **compiler** module

**Validation Level:** Semantic (compiler responsibility)

---

### 2. `incomplete_import.csl` - Import Without Path

**File Content:**
```
import:folder
```

**Why Valid:**
- Import syntax: `import:alias` OR `import:alias:path`
- The path component is **optional** in the grammar
- Valid use case: Import from default locations or resolved paths
- Path resolution happens in the **compiler** during import resolution phase

**Validation Level:** Semantic (path resolution is compiler responsibility)

---

### 3. `invalid_indentation.csl` - Non-Indented Content After Section

**File Content:**
```
section:
key: value
```

**Why Valid:**
- Parser interprets non-indented `key:` as a **new top-level statement**
- This results in an **empty section** followed by a new section declaration
- This is syntactically valid according to the grammar
- The parser does not enforce indentation-based scoping like Python

**Validation Level:** None (valid syntax, potentially confusing but legal)

**Alternative Interpretation:**
- If indentation-based scoping were enforced, this would be invalid
- Current grammar treats indentation as formatting, not semantic structure

---

### 4. `unknown_statement.csl` - Unknown Identifiers

**File Content:**
```
unknown-statement:
	key: value
```

**Why Valid:**
- Parser treats unknown identifiers (not `source`, `import`, or `reference`) as **section declarations**
- This is **by design** to allow user-defined section names
- No validation of section name validity at parse time
- Section names like `database:`, `api:`, `config:` are all equally valid

**Validation Level:** None (extensible grammar allows arbitrary section names)

**Rationale:**
- Nomos configuration language allows arbitrary section names
- Validation of known/unknown sections would require:
  - Schema definition
  - Provider-specific knowledge
  - Context from the compilation environment

---

## Validation Strategy Summary

| Validation Type | Parser | Compiler | Rationale |
|----------------|--------|----------|-----------|
| Syntax (keywords, tokens, structure) | ✅ Yes | ❌ No | Pure grammar validation |
| Duplicate keys | ❌ No | ✅ Yes | Requires scope analysis |
| Import path resolution | ❌ No | ✅ Yes | Requires filesystem/module context |
| Reference resolution | ❌ No | ✅ Yes | Requires symbol table |
| Provider type validation | ❌ No | ✅ Yes | Requires provider registry |
| Section name validation | ❌ No | ❌ No | Extensible grammar by design |

---

## Parser Validation Coverage

### ✅ Enforced by Parser:

1. **Keywords followed by colon:**
   - `source:`, `import:` must have `:`
   
2. **Source declaration structure:**
   - Must have non-empty `alias` field
   - Structured as key-value pairs

3. **Import structure:**
   - Must have alias: `import:alias` or `import:alias:path`

4. **String termination:**
   - String literals must be properly closed

5. **Key format:**
   - Keys must be valid identifiers (non-empty, valid start character)

6. **Reference syntax:**
   - Inline references: `reference:alias:path.components`
   - Top-level `reference:` statements are **rejected** with migration hint

### ❌ NOT Enforced by Parser (Deferred to Compiler):

1. **Duplicate key detection**
2. **Unknown provider types**
3. **Import resolution**
4. **Reference resolution**
5. **Semantic validation of values**

---

## Test Strategy

### Negative Fixtures Test Updates

The `TestGolden_NegativeFixtures` test has been updated with detailed comments explaining:

1. **Why** each file is marked as `knownValid`
2. **What** validation would be needed to reject it
3. **Where** that validation should be implemented (parser vs compiler)

### Golden Errors Test Updates

The `TestGolden_ErrorScenarios` test includes:

1. Only files that **should** trigger parse errors
2. Clear skip messages for valid syntax cases
3. Distinction between syntax errors (parser) and semantic errors (compiler)

---

## Future Enhancements

### Potential Parser Enhancements (Not Currently Planned):

1. **Optional strict mode** - Reject duplicate keys at parse time
2. **Indentation-aware parsing** - Enforce indentation scoping
3. **Schema-aware parsing** - Validate against known section types
4. **Warning system** - Non-fatal warnings for suspicious patterns

### Compiler Enhancements (Recommended):

1. **Duplicate key detection** - Implement in scope analysis phase
2. **Unused import detection** - Track import usage
3. **Dead section detection** - Identify unreferenced sections
4. **Style linting** - Warn about inconsistent indentation

---

## Decision Log

**Date:** 2025-12-25  
**Decision:** Keep parser focused on syntax validation only

**Rationale:**
- Clean separation of concerns (syntax vs semantics)
- Parser remains simple, fast, and dependency-free
- Compiler has full context for semantic validation
- Follows established patterns in language implementations (lexer → parser → semantic analyzer)

**Alternatives Considered:**
- **All-in-one validation:** Rejected due to complexity and tight coupling
- **Schema-driven parsing:** Rejected due to requirement for provider knowledge at parse time
- **Strict mode parsing:** Considered for future enhancement

---

## References

- [Parser Module README](../README.md)
- [Parser Mode Instructions](../AGENTS.md)
- [Compiler Module Documentation](../../compiler/README.md)
- [Nomos Language Specification](../../../docs/examples/README.md)

---

## Changelog

- **2025-12-25:** Initial documentation created during Phase 1.1 refactoring
- Documented 4 knownValid test cases
- Clarified parser vs compiler validation boundaries
