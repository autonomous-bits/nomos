# Parser Architecture Diagrams

This document contains visual diagrams and flowcharts that complement the main [Parser Architecture](./parser-architecture.md) document.

## System Context

```
┌─────────────────────────────────────────────────────────────┐
│                     Nomos Toolchain                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────┐      ┌───────────┐      ┌──────────┐          │
│  │   CLI    │─────►│ Compiler  │─────►│  Parser  │          │
│  │(command) │      │           │      │          │          │
│  └──────────┘      └───────────┘      └──────────┘          │
│                           │                   │             │
│                           │                   │             │
│                           ▼                   ▼             │
│                    ┌──────────┐       ┌──────────┐          │
│                    │ Snapshot │       │   AST    │          │
│                    │  (JSON)  │       │          │          │
│                    └──────────┘       └──────────┘          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Module Architecture

```
libs/parser/
│
├─── Public API Layer ───────────────────────────────────
│    parser.go
│    - ParseFile(path) → AST
│    - Parse(reader, filename) → AST
│    - NewParser(options...) → Parser
│
├─── Parser Implementation ──────────────────────────────
│    parser.go
│    - parseStatements() → []Stmt
│    - parseStatement() → Stmt
│    - parseSourceDecl() → SourceDecl
│    - parseImportStmt() → ImportStmt
│    - parseSectionDecl() → SectionDecl
│    - parseConfigBlock() → map[string]Expr
│    - parseValueExpr() → Expr
│
├─── Scanner/Lexer ──────────────────────────────────────
│    internal/scanner/scanner.go
│    - New(input, filename) → Scanner
│    - PeekChar() → rune
│    - Advance()
│    - ReadIdentifier() → string
│    - ReadValue() → string
│    - IsIndented() → bool
│
├─── AST Definitions ────────────────────────────────────
│    pkg/ast/types.go
│    - AST { Statements, SourceSpan }
│    - SourceDecl, ImportStmt, SectionDecl (Stmt)
│    - StringLiteral, ReferenceExpr (Expr)
│    - SourceSpan { File, Line, Col }
│
└─── Error Handling ─────────────────────────────────────
     errors.go
     - ParseError { kind, location, message, snippet }
     - FormatParseError() → formatted string
     - generateSnippet() → context with caret
```

## Parsing Flow

```
┌───────────────┐
│ User calls    │
│ ParseFile()   │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│ Open file &   │
│ Read content  │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│ Create        │
│ Scanner       │
└───────┬───────┘
        │
        ▼
┌───────────────────────────────────────────┐
│ Loop: parseStatements                     │
│                                           │
│  while !EOF:                              │
│    ┌───────────────────────────────────┐  │
│    │ 1. Skip whitespace                │  │
│    │ 2. Peek token                     │  │
│    │ 3. Dispatch:                      │  │
│    │    - "source"   → parseSourceDecl │  │
│    │    - "import"   → parseImportStmt │  │
│    │    - "reference" → ERROR          │  │
│    │    - identifier → parseSectionDecl│  │
│    │ 4. Add to statements[]            │  │
│    └───────────────────────────────────┘  │
│                                           │
└───────┬───────────────────────────────────┘
        │
        ▼
┌───────────────┐
│ Build AST     │
│ with all      │
│ statements    │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│ Return AST    │
└───────────────┘
```

## Statement Parsing Decision Tree

```
                    ┌─────────────┐
                    │  Peek Token │
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
            ▼              ▼              ▼
     ┌──────────┐   ┌──────────┐   ┌───────────┐
     │"source"  │   │"import"  │   │"reference"│
     └────┬─────┘   └────┬─────┘   └────┬──────┘
          │              │              │
          ▼              ▼              ▼
   ┌────────────┐ ┌────────────┐ ┌─────────────┐
   │parseSource │ │parseImport │ │  ERROR!     │
   │    Decl    │ │    Stmt    │ │ (deprecated)│
   └────┬───────┘ └────┬───────┘ └─────────────┘
        │              │
        └──────┬───────┘
               │
               ▼
        ┌──────────────┐
        │   identifier │
        │  + colon     │
        └──────┬───────┘
               │
               ▼
        ┌──────────────┐
        │parseSection  │
        │    Decl      │
        └──────────────┘
```

## Config Block Parsing

```
section-name:
    key1: value1          ◄─── Line is indented
    key2: @alias:...      ◄─── Line is indented
    nested:               ◄─── Line is indented
        sub: value        ◄─── Line is indented
next-section:             ◄─── NOT indented → stop
```

**Algorithm:**
```
parseConfigBlock():
    config = {}
    
    while not EOF:
        if not IsIndented():
            break  // End of block
        
        key = ReadIdentifier()
        Expect(':')
        value = parseValueExpr()
        
        config[key] = value
    
    return config
```

## Value Expression Parsing

```
                 ┌─────────────────┐
                 │  ReadValue()    │
                 │  (until EOL)    │
                 └────────┬────────┘
                          │
          ┌───────────────┴───────────────┐
          │                               │
          ▼                               ▼
  ┌──────────────┐              ┌─────────────────┐
  │ Starts with  │              │   Plain text    │
    │"@"?           │              │   (or quoted)   │
  └──────┬───────┘              └────────┬────────┘
         │ Yes                           │ No
         ▼                               ▼
  ┌──────────────┐              ┌─────────────────┐
  │ Split by ':' │              │ Strip quotes    │
  │ [ref, alias, │              │                 │
  │    path]     │              │                 │
  └──────┬───────┘              └────────┬────────┘
         │                               │
         ▼                               ▼
  ┌──────────────┐              ┌─────────────────┐
  │ Split path   │              │ StringLiteral   │
  │ by '.' into  │              │ { value, span } │
  │ components   │              └─────────────────┘
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │ReferenceExpr │
  │{ alias,      │
  │  path[],     │
  │  span }      │
  └──────────────┘
```

## AST Node Hierarchy

```
Node (interface)
├── Span() SourceSpan
└── node() // marker

AST (root node)
├── Statements: []Stmt
└── SourceSpan

Stmt (interface)
├── Node
├── stmt() // marker
│
├── SourceDecl
│   ├── Alias: string
│   ├── Type: string
│   ├── Config: map[string]Expr
│   └── SourceSpan
│
├── ImportStmt
│   ├── Alias: string
│   ├── Path: string
│   └── SourceSpan
│
└── SectionDecl
    ├── Name: string
    ├── Entries: map[string]Expr
    └── SourceSpan

Expr (interface)
├── Node
├── expr() // marker
│
├── StringLiteral
│   ├── Value: string
│   └── SourceSpan
│
├── ReferenceExpr
│   ├── Alias: string
│   ├── Path: []string
│   └── SourceSpan
│
├── PathExpr (future)
│   ├── Components: []string
│   └── SourceSpan
│
└── IdentExpr (future)
    ├── Name: string
    └── SourceSpan
```

## Scanner State Machine

```
Scanner maintains:
┌──────────────────────────────────┐
│ input:     "source:\n  alias..." │
│ filename:  "config.csl"          │
│ pos:       0 → len(input)        │
│ line:      1, 2, 3, ...          │
│ col:       1 → max col per line  │
│ lineStart: byte pos of line      │
└──────────────────────────────────┘

Operations:
• PeekChar()   → Look at current rune
• Advance()    → Move pos, update line/col
• ReadIdentifier() → Consume alphanumeric+dash
• ReadValue()  → Consume until EOL
• IsIndented() → Check if at line start with space/tab
```

## Error Handling Flow

```
┌─────────────────┐
│ Error detected  │
│ during parsing  │
└────────┬────────┘
         │
         ▼
┌────────────────────────────┐
│ Create ParseError          │
│ - kind (Lex/Syntax/IO)     │
│ - filename                 │
│ - line, column             │
│ - message                  │
└────────┬───────────────────┘
         │
         ▼
┌────────────────────────────┐
│ Generate snippet           │
│ - Extract context lines    │
│ - Add caret marker (^)     │
└────────┬───────────────────┘
         │
         ▼
┌────────────────────────────┐
│ Return to caller           │
│ (stop parsing)             │
└────────┬───────────────────┘
         │
         ▼
┌────────────────────────────┐
│ User formats error         │
│ FormatParseError()         │
│                            │
│ Output:                    │
│ file:line:col: message     │
│   N | context              │
│     | ^                    │
└────────────────────────────┘
```

## Concurrency Model

```
Main Program
│
├─── Goroutine 1 ───────────────────────────┐
│                                           │
│  p := NewParser()                         │
│  ast1, _ := p.Parse(reader1, "file1.csl") │ 
│                                           │
│  [Parser is stateless - safe]             │   
│                                           │
├─── Goroutine 2 ───────────────────────────┤
│                                           │
│  ast2, _ := p.Parse(reader2, "file2.csl") │
│                                           │
│  [Each parse creates own Scanner]         │
│                                           │
└─── Goroutine N ───────────────────────────┘
     
     ast3, _ := p.Parse(reader3, "file3.csl")

All goroutines share same Parser instance safely!
```

## Memory Layout

```
Parse Operation Memory:

Input File (100KB)
    ↓
┌─────────────────────────┐
│ Source Text String      │  100KB
│ (kept for errors)       │
└─────────────────────────┘
    ↓
┌─────────────────────────┐
│ Scanner                 │  ~100 bytes
│ - pointers to string    │
│ - position counters     │
└─────────────────────────┘
    ↓
┌─────────────────────────┐
│ AST Nodes               │  ~1-2KB per statement
│ - Statements slice      │  (10 statements = 10-20KB)
│ - String maps           │
│ - SourceSpans           │
└─────────────────────────┘

Total: ~100KB input + ~20KB AST = ~120KB
```

## Integration Example

```
┌──────────────────────────────────────┐
│         Compiler Module              │
├──────────────────────────────────────┤
│                                      │
│  import "...libs/parser"             │
│  import "...libs/parser/pkg/ast"     │
│                                      │
│  func Compile(file string) {         │
│                                      │
│    // Parse                          │
│    ast, err := parser.ParseFile(file)│──┐
│    if err != nil {                   │  │
│      return err                      │  │
│    }                                 │  │
│                                      │  │
│    // Semantic Analysis              │  │
│    for _, stmt := range ast.Statements  │
│      switch s := stmt.(type) {       │  │
│      case *ast.SourceDecl:           │  │
│        registerSource(s)             │  │
│      case *ast.ImportStmt:           │  │
│        resolveImport(s)              │  │
│      case *ast.SectionDecl:          │  │
│        compileSection(s)             │  │
│      }                               │  │
│    }                                 │  │
│                                      │  │
│    // Generate output                │  │
│    return generateSnapshot()         │  │
│  }                                   │  │
│                                      │  │
└──────────────────────────────────────┘  │
                                          │
        ┌─────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────┐
│         Parser Module                │
├──────────────────────────────────────┤
│                                      │
│  func ParseFile(path) (*ast.AST) {   │
│    // Read file                      │
│    // Create scanner                 │
│    // Parse statements               │
│    // Build AST                      │
│    return ast                        │
│  }                                   │
│                                      │
└──────────────────────────────────────┘
```

## Testing Architecture

```
Test Suite Organization:

Unit Tests (Package Level)
├── scanner_test.go
│   └── Test scanner methods
├── types_test.go
│   └── Test AST node creation
└── parser.go (inline tests)

Integration Tests (test/)
├── parser_api_test.go
│   └── Test public API
├── parser_grammar_test.go
│   └── Test each grammar rule
├── golden_test.go
│   └── AST snapshot testing
├── golden_errors_test.go
│   └── Error message stability
└── integration/
    ├── concurrency_test.go
    └── parse_error_test.go

Test Data (testdata/)
├── fixtures/           ← Valid .csl files
│   └── *.csl
├── golden/            ← Expected ASTs
│   ├── *.csl.json
│   └── errors/
│       └── *.error.json
└── errors/            ← Invalid .csl files
    └── *.csl
```

## Summary

These diagrams illustrate the key architectural patterns of the Nomos Parser:

1. **Layered Architecture** - Clear separation between API, parser, scanner, and AST
2. **Single-Pass Parsing** - Linear flow from input to AST
3. **Decision Tree** - Token-based dispatch to appropriate parser methods
4. **Recursive Structure** - Config blocks can contain nested blocks
5. **Expression Polymorphism** - Values can be literals or references
6. **Stateless Concurrency** - Shared parser instances, isolated scanner state
7. **Comprehensive Testing** - Multiple test strategies for reliability

For detailed explanations, see the main [Parser Architecture](./parser-architecture.md) document.
