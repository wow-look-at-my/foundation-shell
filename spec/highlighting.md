# Syntax Highlighting Specification

This document defines the complete specification for Foundation Shell's syntax highlighting system. The highlighter provides real-time visual feedback using ANSI escape codes.

## Overview

Syntax highlighting in Foundation Shell operates on two levels:

1. **Analysis** - Semantic tokenization and error detection (analyzer.go)
2. **Rendering** - ANSI escape code application (highlighter.go)

The analyzer is the single source of truth for syntax structure. The highlighter applies visual styling based on analyzer output.

## Architecture

```
Input String
     │
     ▼
┌──────────┐
│ Analyzer │  → AnalysisResult (tokens + errors)
└────┬─────┘
     │
     ▼
┌─────────────┐
│ Highlighter │  → ANSI-colored string
└─────────────┘
```

## Semantic Types

The analyzer classifies tokens into semantic types:

```go
type SemanticType int

const (
    TypeUnknown SemanticType = iota
    TypeCommand             // First word in a command
    TypeArgument            // Command arguments
    TypeOperator            // Control operators (|, &&, ||, ;)
    TypeRedirection         // Redirection operators (>, <, >>, 2>, 2>>)
    TypeRedirectionTarget   // Target file for redirections
    TypeSingleQuotedString  // Content in single quotes
    TypeDoubleQuotedString  // Content in double quotes
    TypeBacktick            // Backtick command substitution
    TypeSubshell            // $(...) command substitution
    TypeVariable            // Variable reference ($VAR, ${VAR})
    TypeParenGroup          // Parenthesized group (...)
    TypeError               // Invalid/error regions
    TypeWhitespace          // Whitespace between tokens
)
```

### Type Priority

When multiple classifications could apply, this priority is used:

1. **Errors** - Syntax errors override other types
2. **Quote context** - Quoted strings maintain their type
3. **Structural** - Redirection targets, operators
4. **Positional** - Command vs argument based on position

## Analyzed Token

```go
type AnalyzedToken struct {
    Type  SemanticType
    Value string
    Start int  // Character position (0-indexed)
    End   int  // Character position (exclusive)
    Depth int  // Nesting depth for quotes/parens/subshells
}
```

### Depth Tracking

The `Depth` field tracks nesting for:
- Single quotes
- Double quotes
- Backticks
- Subshells `$()`

Depth uses even/odd counting:
- Even count (0, 2, 4, ...) = outside/closed
- Odd count (1, 3, 5, ...) = inside/unclosed

## Analysis Result

```go
type AnalysisResult struct {
    Tokens []AnalyzedToken
    Errors []SyntaxError
    Valid  bool  // true if len(Errors) == 0
}
```

## Theme System

### Theme Definition

```go
type Theme map[SemanticType]string
```

A theme maps semantic types to ANSI escape codes.

### Default Theme

```go
var DefaultTheme = Theme{
    TypeCommand:            "\033[1;36m", // Bold cyan
    TypeArgument:           "\033[0m",    // Default
    TypeOperator:           "\033[1;33m", // Bold yellow
    TypeRedirection:        "\033[35m",   // Magenta
    TypeRedirectionTarget:  "\033[0m",    // Default
    TypeSingleQuotedString: "\033[32m",   // Green
    TypeDoubleQuotedString: "\033[33m",   // Yellow
    TypeBacktick:           "\033[36m",   // Cyan
    TypeSubshell:           "\033[36m",   // Cyan
    TypeVariable:           "\033[34m",   // Blue
    TypeParenGroup:         "\033[35m",   // Magenta
    TypeError:              "\033[4;31m", // Underline red
    TypeWhitespace:         "\033[0m",    // Reset
    TypeUnknown:            "\033[0m",    // Reset
}
```

### Color Summary

| Type | Color | ANSI Code | Sample |
|------|-------|-----------|--------|
| Command | Bold Cyan | `\033[1;36m` | `ls` |
| Argument | Default | `\033[0m` | `-la` |
| Operator | Bold Yellow | `\033[1;33m` | `\|` |
| Redirection | Magenta | `\033[35m` | `>` |
| RedirectionTarget | Default | `\033[0m` | `file.txt` |
| SingleQuotedString | Green | `\033[32m` | `'hello'` |
| DoubleQuotedString | Yellow | `\033[33m` | `"world"` |
| Backtick | Cyan | `\033[36m` | `` `cmd` `` |
| Subshell | Cyan | `\033[36m` | `$(cmd)` |
| Variable | Blue | `\033[34m` | `$HOME` |
| ParenGroup | Magenta | `\033[35m` | `(...)` |
| Error | Underline Red | `\033[4;31m` | Invalid syntax |

### ANSI Escape Code Reference

```
\033[0m     - Reset
\033[1m     - Bold
\033[4m     - Underline
\033[30-37m - Foreground colors (black, red, green, yellow, blue, magenta, cyan, white)
\033[1;XXm  - Bold + color
\033[4;XXm  - Underline + color
```

## Highlighter API

### Constructor

```go
func NewHighlighter(theme Theme) *Highlighter
```

Creates a highlighter with the specified theme.

### Highlight

```go
func (h *Highlighter) Highlight(input string) string
```

Returns ANSI-colored string. Errors are silently highlighted with error color.

### HighlightResult

```go
func (h *Highlighter) HighlightResult(input string) (string, []SyntaxError)
```

Returns both the colored string and any syntax errors found.

## Analyzer Logic

### State Machine

The analyzer maintains state:

```go
type analyzer struct {
    input  []rune
    pos    int
    tokens []AnalyzedToken
    errors []SyntaxError

    isFirstInCommand bool  // Track command vs argument
    afterRedirection bool  // Track redirection targets
}
```

### Analysis Flow

```
1. Initialize: isFirstInCommand = true, afterRedirection = false
2. For each character position:
   a. Skip and record whitespace
   b. Check for operators (match longest first)
   c. Parse words (handling quotes, escapes, substitutions)
3. Check for trailing errors (unclosed operators, missing targets)
```

### Operator Matching

Operators are matched longest-first:

```
Priority:
1. 3-char: 2>>
2. 2-char: &&, ||, >>, 2>
3. 1-char: |, ;, >, <, (, )
```

### Word Parsing

Words are parsed with depth-tracked quote handling:

```go
// Track quote depths
singleQuoteDepth := 0
doubleQuoteDepth := 0
backtickDepth := 0
parenDepth := 0  // For $()

// Even count = outside quotes (valid boundary)
// Odd count = inside quotes (continue parsing)
outsideQuotes := singleQuoteDepth%2 == 0 &&
                 doubleQuoteDepth%2 == 0 &&
                 backtickDepth%2 == 0 &&
                 parenDepth == 0
```

### Escape Handling

Escape sequences are handled outside single quotes:

```
\' - Escaped single quote (outside single quotes)
\" - Escaped double quote
\` - Escaped backtick
\\ - Escaped backslash
\n - Escaped newline (etc.)
```

Inside single quotes, backslash is literal.

## Type Determination

### Decision Tree

```
1. Has unclosed quotes/parens? → TypeError
2. After redirection operator? → TypeRedirectionTarget
3. Entire token single-quoted? → TypeSingleQuotedString
4. Entire token double-quoted? → TypeDoubleQuotedString
5. Entire token backtick? → TypeBacktick
6. Starts with $( and ends with )? → TypeSubshell
7. Starts with $ (length > 1)? → TypeVariable
8. First in command? → TypeCommand
9. Otherwise → TypeArgument
```

### Quote Detection

```go
// Check if entire token is quoted
if len(value) >= 2 {
    if value[0] == '\'' && value[len(value)-1] == '\'' {
        return TypeSingleQuotedString
    }
    if value[0] == '"' && value[len(value)-1] == '"' {
        return TypeDoubleQuotedString
    }
    if value[0] == '`' && value[len(value)-1] == '`' {
        return TypeBacktick
    }
}
```

## Error Detection

### Unclosed Constructs

```go
// After parsing a word:
if singleQuoteDepth%2 != 0 {
    error: "unclosed single quote (odd count)"
}
if doubleQuoteDepth%2 != 0 {
    error: "unclosed double quote (odd count)"
}
if backtickDepth%2 != 0 {
    error: "unclosed backtick (odd count)"
}
if parenDepth > 0 {
    error: "unclosed subshell $(...)"
}
```

### Trailing Errors

```go
// After full analysis:
lastToken := tokens[len(tokens)-1]

if lastToken.Type == TypeOperator && lastToken.Value != ";" {
    error: "unexpected operator at end"
}
if lastToken.Type == TypeRedirection {
    error: "missing redirection target"
}
```

## REPL Integration

### Readline Painter

The shell implements `readline.Painter` for real-time highlighting:

```go
type syntaxPainter struct {
    highlighter *syntax.Highlighter
}

func (p *syntaxPainter) Paint(line []rune, pos int) []rune {
    return []rune(p.highlighter.Highlight(string(line)))
}
```

### Configuration

```go
cfg := &readline.Config{
    Painter: &syntaxPainter{
        highlighter: syntax.NewHighlighter(syntax.DefaultTheme),
    },
    // ...
}
```

## Theme Application

### Algorithm

```go
func (h *Highlighter) applyTheme(tokens []AnalyzedToken) string {
    var builder strings.Builder

    for _, token := range tokens {
        // Get color for token type
        color, ok := h.theme[token.Type]
        if !ok {
            color = ansiReset  // "\033[0m"
        }

        // Apply: color + value + reset
        builder.WriteString(color)
        builder.WriteString(token.Value)
        builder.WriteString(ansiReset)
    }

    return builder.String()
}
```

### Output Format

Each token is wrapped:
```
[color code][token value][reset code]
```

Example:
```
echo hello
```
Becomes:
```
\033[1;36mecho\033[0m\033[0m \033[0m\033[0m\033[0mhello\033[0m
```

## Examples

### Simple Command

```
Input:  ls -la
Tokens: [Command:"ls", Whitespace:" ", Argument:"-la"]
Output: \033[1;36mls\033[0m\033[0m \033[0m\033[0m-la\033[0m
Visual: ls -la (cyan "ls", default "-la")
```

### Pipeline

```
Input:  cat file | grep pattern
Tokens: [Command:"cat", Whitespace:" ", Argument:"file", Whitespace:" ",
         Operator:"|", Whitespace:" ", Command:"grep", Whitespace:" ",
         Argument:"pattern"]
Visual: cat file | grep pattern
        (cyan commands, yellow pipe)
```

### Quoted Strings

```
Input:  echo 'hello' "world"
Tokens: [Command:"echo", Whitespace:" ", SingleQuotedString:"'hello'",
         Whitespace:" ", DoubleQuotedString:"\"world\""]
Visual: echo 'hello' "world"
        (cyan echo, green 'hello', yellow "world")
```

### Variables

```
Input:  echo $HOME ${USER}
Tokens: [Command:"echo", Whitespace:" ", Variable:"$HOME",
         Whitespace:" ", Variable:"${USER}"]
Visual: echo $HOME ${USER}
        (cyan echo, blue variables)
```

### Command Substitution

```
Input:  echo $(whoami)
Tokens: [Command:"echo", Whitespace:" ", Subshell:"$(whoami)"]
Visual: echo $(whoami)
        (cyan echo, cyan subshell)
```

### Redirection

```
Input:  cat < in.txt > out.txt
Tokens: [Command:"cat", Whitespace:" ", Redirection:"<", Whitespace:" ",
         RedirectionTarget:"in.txt", Whitespace:" ", Redirection:">",
         Whitespace:" ", RedirectionTarget:"out.txt"]
Visual: cat < in.txt > out.txt
        (cyan cat, magenta redirections, default targets)
```

### Error Highlighting

```
Input:  echo 'unclosed
Tokens: [Command:"echo", Whitespace:" ", Error:"'unclosed"]
Visual: echo 'unclosed
        (cyan echo, underline red error)
```

## State Transitions

### Command/Argument Tracking

```
State: isFirstInCommand = true

Token: "echo"    → TypeCommand,    isFirstInCommand = false
Token: " "       → TypeWhitespace, (no change)
Token: "hello"   → TypeArgument,   isFirstInCommand = false
Token: "|"       → TypeOperator,   isFirstInCommand = true
Token: "grep"    → TypeCommand,    isFirstInCommand = false
```

### Redirection Target Tracking

```
State: afterRedirection = false

Token: ">"       → TypeRedirection, afterRedirection = true
Token: " "       → TypeWhitespace,  (no change)
Token: "file"    → TypeRedirectionTarget, afterRedirection = false
```

## Custom Themes

### Creating a Theme

```go
customTheme := syntax.Theme{
    syntax.TypeCommand:    "\033[1;32m",  // Bold green
    syntax.TypeArgument:   "\033[37m",    // White
    syntax.TypeOperator:   "\033[1;35m",  // Bold magenta
    // ... other types
}

highlighter := syntax.NewHighlighter(customTheme)
```

### Missing Types

If a type is not in the theme, it defaults to reset (`\033[0m`).

## Performance Considerations

1. **Single Pass** - Analysis runs in O(n) time
2. **Rune-based** - Proper Unicode handling
3. **Minimal Allocation** - StringBuilder for output
4. **No Regex** - Hand-written parsing for speed

## Testing

### Token Verification

```go
result := Analyze("echo hello")
assert(len(result.Tokens) == 3)
assert(result.Tokens[0].Type == TypeCommand)
assert(result.Tokens[0].Value == "echo")
```

### Error Verification

```go
result := Analyze("echo 'unclosed")
assert(!result.Valid)
assert(len(result.Errors) == 1)
assert(strings.Contains(result.Errors[0].Message, "unclosed single quote"))
```

### Highlight Output

```go
hl := NewHighlighter(DefaultTheme)
output := hl.Highlight("echo hello")
assert(strings.Contains(output, "\033[1;36m"))  // Contains command color
assert(strings.Contains(output, "\033[0m"))     // Contains reset
```

## Known Behaviors

### Depth Tracking

Foundation Shell uses **depth-tracked** quotes rather than strict matching:

```
'outer 'inner' end'
```

Quote count: 4 (even = valid)
- Each `'` increments depth
- Even total means properly balanced

This differs from POSIX shells where each quote must explicitly close its pair.

### Whitespace Preservation

All whitespace is tokenized and highlighted:
- Spaces between tokens → TypeWhitespace
- Multiple spaces preserved
- Tabs, newlines if present

### Partial Input

The analyzer handles partial/incomplete input gracefully:
- Unclosed quotes → TypeError token + error
- Trailing operators → Error reported
- Real-time highlighting works during typing
