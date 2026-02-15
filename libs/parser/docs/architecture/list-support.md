# ADR-001: List/Array Support in CSL Parser

## Status

Accepted

## Context

The Nomos Configuration Scripting Language (CSL) currently supports scalar values (strings, numbers, booleans) and map/object structures through indentation-based syntax similar to YAML. However, there is no support for ordered lists or arrays, which are essential for many configuration use cases:

- **Network Configuration**: Lists of IP addresses, CIDR blocks, DNS servers
- **Resource Collections**: Multiple instances, security groups, availability zones
- **Ordered Operations**: Sequential steps, deployment stages, migration phases
- **Data Structures**: Any configuration requiring ordered, indexed collections

Without list support, users must work around the limitation by:
1. Using numbered keys in maps (`ip_0`, `ip_1`, `ip_2`)
2. Using comma-separated strings and parsing downstream
3. Duplicating resource blocks instead of using collections

These workarounds make configurations harder to read, maintain, and validate. List support is a fundamental requirement for a configuration language to be practical and expressive.

### Requirements

1. **Block-style List Syntax**: Support YAML-style dash notation (`-`) for list items
2. **Nesting**: Allow lists within lists and lists within maps
3. **Type Flexibility**: Support heterogeneous element types at parser level (strings, numbers, maps, nested lists)
4. **Reference Support**: Enable list indexing in references (`@alias:config:IPs[0]`)
5. **Empty Lists**: Explicit syntax for empty collections (`[]`)
6. **Validation**: Enforce consistent indentation (2 spaces) and provide clear error messages
7. **Performance**: Minimal overhead compared to existing parsing operations

## Decision

### 1. Parsing Strategy: Line-based Indentation Tracking

We extend the existing scanner's line-based indentation tracking to recognize list item markers:

- **List Marker**: Dash (`-`) followed by whitespace at consistent indentation
- **Recognition**: Scanner detects dash at start of line after indentation
- **Parsing**: Parser recursively parses list item values (scalars, maps, or nested lists)
- **Termination**: List ends on dedent or non-dash line at same indentation level

**Rationale**: This approach aligns with the existing scanner architecture that already tracks indentation for map structures. It provides clean separation between lexical analysis (scanner) and syntactic analysis (parser).

**Implementation Pattern**:
```go
// Scanner: Detect list item marker
func (s *Scanner) IsListItemMarker() bool {
    return s.PeekChar() == '-' && unicode.IsSpace(s.PeekCharAt(1))
}

// Parser: Parse list expression
func (p *Parser) parseListExpr(s *scanner.Scanner, baseIndent int) (*ast.ListExpr, error) {
    var elements []ast.Expr
    listIndent := s.GetIndentLevel()
    
    for !s.IsEOF() {
        if s.GetIndentLevel() < listIndent {
            break // Dedent signals end
        }
        if !s.IsListItemMarker() {
            break // No dash means list ended
        }
        
        s.Advance() // Skip '-'
        s.SkipWhitespace()
        
        elem, err := p.parseValueExpr(s, ...)
        if err != nil {
            return nil, err
        }
        elements = append(elements, elem)
    }
    
    return &ast.ListExpr{Elements: elements, ...}, nil
}
```

### 2. AST Representation: Dedicated ListExpr Node

Add a new expression type to the AST:

```go
type ListExpr struct {
    Elements   []Expr      // Ordered list of expressions
    SourceSpan SourceSpan  // Location information for error reporting
}

func (l *ListExpr) Span() SourceSpan { return l.SourceSpan }
func (l *ListExpr) node() {}
func (l *ListExpr) expr() {}
```

**Key Design Choices**:

- **Elements as []Expr**: Each element can be any expression type (StringLiteral, NumberLiteral, MapExpr, nested ListExpr, or ReferenceExpr)
- **Type Homogeneity**: Not enforced by parser; deferred to compiler's semantic analysis phase
- **Empty Lists**: Represented as zero-length slice with special `[]` syntax parsing
- **Ordering**: Slice naturally maintains insertion order

**Rationale**: A dedicated node type provides clear semantics and type safety. Using `[]Expr` allows full nesting and composition while keeping the parser simple. Type validation belongs in the compiler where full type information is available.

### 3. Indentation Validation: Strict 2-Space Enforcement

List items must align vertically with consistent indentation:

**Rules**:
1. First list item establishes baseline indentation for all items
2. Each subsequent item must match baseline exactly
3. Nested structures indent 2 spaces deeper than parent
4. Only spaces allowed (no tabs)

**Validation Strategy**:
```go
func (s *Scanner) ValidateListIndentation(expectedIndent int) error {
    actual := s.GetIndentLevel()
    if actual != expectedIndent {
        return NewParseError(
            SyntaxError,
            s.Filename(), s.Line(), s.Column(),
            fmt.Sprintf("inconsistent list indentation: expected %d spaces, got %d", 
                expectedIndent, actual),
        )
    }
    return nil
}
```

**Valid Example**:
```yaml
IPs:
  - 10.0.0.1
  - 10.1.0.1
  - 10.2.0.1
```

**Invalid Example**:
```yaml
IPs:
  - 10.0.0.1
   - 10.1.0.1  # Error: 3 spaces instead of 2
```

**Rationale**: Strict validation at parse time catches errors early and enforces consistency across all CSL files. This prevents ambiguous structures and makes configurations easier to read and diff.

### 4. Error Handling: Rich Contextual Messages

Provide actionable error messages with source context:

**Error Categories**:

1. **Empty List Items**:
```
error: empty list item at line 3, column 3

  2 | IPs:
  3 |   - 
      ^
Empty list items are not allowed. Either provide a value or use empty list syntax: []
```

2. **Inconsistent Indentation**:
```
error: inconsistent list indentation at line 4, column 4

  3 |   - 10.0.0.1
  4 |    - 10.1.0.1
       ^
Expected 2 spaces (matching previous list item), got 3 spaces
```

3. **Nesting Depth Exceeded**:
```
error: maximum nesting depth exceeded at line 15, column 5

  14 |         - level19
  15 |           - level20
                ^
Maximum nesting depth is 20 levels. Consider restructuring your configuration.
```

4. **Mixed Whitespace**:
```
error: tab character in indentation at line 3, column 1

  2 | IPs:
  3 | →	- 10.0.0.1
      ^
Use spaces for indentation, not tabs. List items must use 2-space indentation.
```

**Error Structure**:
```go
type ParseError struct {
    Kind       ParseErrorKind
    Filename   string
    Line       int
    Column     int
    Message    string
    Snippet    string    // Source context with caret
    Suggestion string    // Actionable fix guidance
}
```

### 4.1 Comprehensive Error Examples

The parser returns errors with file, line, and column detail, plus a caret and
actionable guidance. Below are representative examples for list-related failures.

1) **Empty list item** (use `[]` for empty lists):

```
config.csl:3:5: syntax error: empty list item not allowed

    2 | servers:
    3 |   -
            ^
List items cannot be empty. Use [] for an empty list.
```

2) **Whitespace-only list** (no values present):

```
config.csl:6:3: syntax error: list contains only whitespace

    5 | tags:
    6 |   -
    7 |   -
            ^
List must contain at least one non-whitespace value or use [].
```

3) **Inconsistent indentation** (must be 2 spaces per level):

```
config.csl:9:4: syntax error: inconsistent list item indentation

    8 | zones:
    9 |    - us-east-1a
             ^
Expected 2-space indentation for list items, got 3 spaces.
```

4) **Tab character in indentation** (spaces only):

```
config.csl:12:1: syntax error: tab character in indentation

	- us-east-1b
^
Indentation must use spaces only. Replace tabs with 2 spaces per level.
```

5) **Nesting depth exceeded** (max 20 levels):

```
config.csl:22:41: syntax error: list nesting depth exceeded

                                                                                - level21
                                                                                ^
Maximum list nesting depth is 20 levels. Current depth: 21.
Simplify your data structure or split into multiple sections.
```

**Rationale**: Clear, actionable error messages improve developer experience and reduce debugging time. Context snippets help users locate and understand issues quickly.

### 5. Index Syntax in References: String-based Path Components

Support list indexing in reference expressions:

**Syntax**:
```
@alias:config:IPs[0]           # Single index
@alias:config:nested.lists[1][2]      # Multi-dimensional
```

**Implementation**: Extend reference path parsing to include index notation as part of path components:

```go
type ReferenceExpr struct {
    Alias string
    Path  []string  // ["config", "IPs[0]"] - index embedded in string
    ...
}
```

**Parsing Approach**:
```go
func (p *Parser) parseReferencePath(s *scanner.Scanner) ([]string, error) {
    var components []string
    
    for {
        ident := s.ReadIdentifier()
        
        // Check for bracket notation
        if s.PeekChar() == '[' {
            s.Advance()
            index := s.ReadDigits()
            if s.PeekChar() != ']' {
                return nil, NewParseError(...)
            }
            s.Advance()
            ident += "[" + index + "]"
        }
        
        components = append(components, ident)
        
        if s.PeekChar() != '.' {
            break
        }
        s.Advance()
    }
    
    return components, nil
}
```

**Rationale**: String-based representation keeps parser changes minimal. The compiler can parse index notation during reference resolution when it has full context for bounds checking and type validation. This maintains the parser's role as syntactic analyzer without semantic concerns.

### 6. Type Checking Strategy: Deferred to Compiler

The parser accepts any expression types in list elements:

**Parser Responsibility**: Syntactic correctness only
- Valid list syntax
- Correct indentation
- Well-formed elements

**Compiler Responsibility**: Semantic validation
- Type homogeneity (if required by use case)
- Reference resolution
- Index bounds checking
- Type compatibility

**Rationale**: Separating syntactic and semantic concerns keeps the parser simple and focused. The compiler has full context (type information, resolved references, value evaluation) needed for meaningful semantic validation. This also allows flexibility for different use cases (some may want heterogeneous lists).

### 6.1 Merge Semantics: Handled by Compiler

List merge behavior is a compilation concern. The parser only produces a list AST when the syntax is valid.
When multiple configuration layers are merged, lists **replace** prior lists rather than merging element-by-element.

**Rationale**: Merge rules require knowledge of configuration layering and evaluation order, which lives in the compiler.
Keeping the parser focused on syntax prevents accidental semantic drift and keeps list handling deterministic.

### 7. Empty List Representation: Explicit [] Syntax

Support explicit empty list syntax:

**Syntax**:
```yaml
emptyList: []
```

**AST Representation**:
```go
&ast.ListExpr{
    Elements:   []Expr{},  // Zero-length slice
    SourceSpan: ...,
}
```

**Rationale**: Explicit syntax removes ambiguity between empty lists, whitespace-only content, and missing values. It makes intent clear and aligns with JSON/YAML conventions.

### 8. Nesting Limits: 20 Levels Maximum

Enforce maximum nesting depth to prevent stack overflow and overly complex configurations:

**Limit**: 20 levels of nested lists
**Enforcement**: Track depth during recursive parsing
**Error**: Clear message suggesting restructuring

**Rationale**: Deeply nested structures (>20 levels) indicate design problems and can cause stack overflow during recursive parsing. This limit is generous for legitimate use cases while protecting against pathological inputs.

## Consequences

### Positive

1. **Feature Completeness**: CSL becomes feature-complete with fundamental list support
2. **User Experience**: Natural syntax familiar to YAML users
3. **Maintainability**: Dedicated AST node makes lists first-class citizens
4. **Extensibility**: Foundation for future list operations (filtering, mapping, etc.)
5. **Error Quality**: Rich error messages reduce debugging time
6. **Performance**: Minimal overhead using existing scanner patterns
7. **No Dependencies**: Implemented using existing infrastructure and standard library

### Negative

1. **Parser Complexity**: Additional parsing logic increases code size (~200-300 lines)
2. **Testing Burden**: Need comprehensive tests for edge cases (nesting, indentation errors, etc.)
3. **Breaking Changes**: None - purely additive feature
4. **Migration**: None required - existing files parse unchanged

### Neutral

1. **Indentation Strictness**: Some users may prefer more lenient validation
2. **Type Flexibility**: Heterogeneous lists may surprise users expecting type enforcement
3. **String-based Indexing**: Trade-off between parser simplicity and type safety

## Alternatives Considered

### Alternative 1: Reuse go-yaml Parser

**Approach**: Use `gopkg.in/yaml.v3` library for list parsing.

**Rejected Because**:
- Adds external dependency
- Parser already handles YAML-like structures successfully
- Overkill for our specific needs
- Less control over error messages and behavior

### Alternative 2: Lookahead-based State Machine

**Approach**: Use multi-character lookahead to detect list patterns before committing to list parsing.

**Rejected Because**:
- More complex state management
- Harder to test and debug
- Doesn't align with existing scanner architecture
- Line-based approach is simpler and sufficient

### Alternative 3: Reuse MapExpr with Integer Keys

**Approach**: Represent lists as maps with integer string keys (`"0"`, `"1"`, `"2"`).

**Rejected Because**:
- Confuses semantic meaning (maps are unordered)
- Loses ordering guarantees
- Makes list operations awkward
- Poor user experience in error messages

### Alternative 4: Typed Path Components in References

**Approach**: Parse index notation into dedicated `IndexAccess` types in reference paths:
```go
type PathComponent interface {}
type PropertyAccess struct { Name string }
type IndexAccess struct { Index int }
```

**Rejected Because**:
- Increases parser complexity significantly
- Parser doesn't have context for bounds checking anyway
- String-based approach is simpler and sufficient
- Can revisit if needed for future optimizations

### Alternative 5: Dynamic Indentation (any consistent spacing)

**Approach**: Accept any consistent indentation (2, 4, 8 spaces) as long as items align.

**Rejected Because**:
- CSL already mandates 2-space indentation
- Consistency across all files is valuable
- Strict validation catches errors early
- No significant benefit for the added complexity

## Implementation Notes

### Parser Changes Required

1. **Scanner Enhancements**:
   - `IsListItemMarker()`: Detect dash-space pattern
   - `GetIndentLevel()`: Calculate current indentation
   - `ValidateListIndentation()`: Enforce consistency

2. **AST Additions**:
   - `ListExpr` type with `Elements []Expr`
   - Update `Expr` interface implementations

3. **Parser Extensions**:
   - `parseListExpr()`: Main list parsing logic
   - Update `parseValueExpr()` to recognize list markers
   - Depth tracking for nesting limit enforcement

4. **Reference Parsing**:
   - Extend `parseReferencePath()` to handle `[index]` notation
   - Maintain backward compatibility with non-indexed paths

### Testing Strategy

1. **Unit Tests**:
   - Valid list syntax variations
   - Nested lists (up to 20 levels)
   - Empty lists
   - Indentation error cases
   - Mixed content (scalars, maps, lists)
   - Index notation in references

2. **Integration Tests**:
   - End-to-end parsing of list-containing files
   - Reference resolution with list indexing
   - Error message clarity and context

3. **Performance Tests**:
   - Large lists (1000+ elements)
   - Deeply nested structures
   - Compare overhead vs non-list parsing

### Migration Path

**No migration required** - this is a purely additive feature. Existing CSL files without lists continue to parse unchanged. Users can adopt list syntax incrementally as needed.

### Performance Targets

- List parsing overhead: <10% compared to equivalent map-based structures
- Memory overhead: Minimal (slice backing array + node overhead)
- Nesting overhead: O(depth) stack frames, limited to 20 levels

### Documentation Requirements

1. **Syntax Guide**: Add list syntax section with examples
2. **Reference Guide**: Document list indexing syntax
3. **Migration Guide**: Show before/after examples for common patterns
4. **Error Catalog**: Document all list-related error messages

## Related Documents

- Research: [/specs/004-csl-list-support/research.md](/specs/004-csl-list-support/research.md)
- Parser Architecture: [libs/parser/docs/architecture/parser-architecture.md](parser-architecture.md)
- AST Design: `libs/parser/pkg/ast/types.go`
- TESTING_GUIDE: [/docs/TESTING_GUIDE.md](/docs/TESTING_GUIDE.md)

## References

- [Keep a Changelog](https://keepachangelog.com/)
- [YAML Specification](https://yaml.org/spec/1.2/spec.html) - Block sequences
- [Go Slices](https://go.dev/blog/slices-intro) - Slice internals and performance

---

**Decision Date**: 2026-01-18  
**Decision Makers**: Architecture Team  
**Review Date**: After implementation and user feedback
