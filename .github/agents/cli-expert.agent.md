# CLI Expert Agent

## Purpose
Provides comprehensive CLI design expertise covering POSIX/GNU conventions, command structure, I/O handling, and Cobra framework patterns for building command-line tools that serve both human operators and automated systems effectively.

## Standards Source
All content from: https://github.com/autonomous-bits/development-standards/tree/main/cli_design
Last synced: 2025-12-25

## Coverage Areas
- Command Structure & Naming
- POSIX & GNU Flag Conventions
- Flag Preference Over Positional Arguments
- Standard I/O Patterns (stdin/stdout/stderr)
- Help & Version Commands
- Output Design & Verbosity
- Configuration Management
- Exit Code Handling
- Error Message Standards
- Cobra Framework Best Practices
- CLI UX Guidelines (Human-First Philosophy)
- Terminal Independence
- Composability Patterns

## Content

### Command Structure & Naming

#### Command Anatomy

Command-line instructions follow a consistent, predictable structure analogous to human language grammar:

**Fundamental Components:**
```bash
git commit -m "A descriptive message"
```

**Component Breakdown:**
1. **Command:** `git` - The name of the program you are running
2. **Subcommand:** `commit` - A specific action or function within the program
3. **Flag:** `-m` - A modifier, also called an option or switch
4. **Argument:** `"A descriptive message"` - The value or data provided to the flag

**Grammar Analogy:**
- Command → Program
- Subcommand → Verb
- Argument → Noun
- Flag → Adjective

This consistent grammar makes the CLI a learnable and expressive language for interacting with a computer.

#### Naming Principles

**Command Naming:**
- Should be a simple, memorable word
- Should be short and easy to type
- Should use only lowercase letters
- Should not conflict with common system utilities (e.g., `ls`, `ps`, `cd`)

**Subcommand Structure:**
- First word: the command (the program itself)
- Second word: the subcommand (a specific task within that program)
- Examples:
  - `heroku apps` - lists your applications
  - `heroku apps:create` - creates a new application
  - Colon notation indicates nested subcommand relationships

**Design Philosophy:**
Well-designed command names reduce cognitive load and make the tool intuitive and guessable. They are a primary factor in user adoption and satisfaction.

---

### POSIX & GNU Flag Conventions

**Summary:**
Adherence to POSIX and GNU flag conventions is critical for CLI interoperability and user intuition. These standardized conventions enable users to predict command behavior and reduce the cognitive load required to learn new tools.

#### Short Options

**Format:** Single alphanumeric character preceded by a single dash (`-`)

**Rules:**
- Must be exactly one character
- Preceded by single dash: `-a`, `-v`, `-h`
- Multiple short options that do not take arguments MAY be grouped
- Grouping example: `-ab` is equivalent to `-a -b`

**Examples:**
```bash
command -v          # Single short option
command -a -b -c    # Multiple short options
command -abc        # Grouped short options (equivalent to -a -b -c)
command -f file.txt # Short option with argument
```

#### Long Options

**Format:** Descriptive names preceded by double dash (`--`)

**Rules:**
- Must be descriptive, multi-character names
- Preceded by double dash: `--help`, `--version`, `--output`
- All short options MUST have a corresponding long version
- Enhances script readability and self-documentation

**Examples:**
```bash
command --help              # Long option
command --verbose           # Long option equivalent to -v
command --output file.txt   # Long option with argument
command --force --quiet     # Multiple long options
```

**Mandatory Correspondence:**
- `-h` MUST have `--help`
- `-v` MUST have `--verbose` (or context-appropriate alternative)
- `-o` MUST have `--output`
- Every short option MUST have a long equivalent

#### End of Options Marker

**Syntax:** Double dash `--` signifies end of all options

**Purpose:**
- Any subsequent arguments are treated as operands, not options
- Essential for handling filenames that begin with dashes
- Enables processing of special characters

**Examples:**
```bash
command -- -filename.txt           # "-filename.txt" treated as operand
command --output file.txt -- -v    # "-v" treated as filename, not option
rm -- -rf                          # Removes file named "-rf"
```

#### Implementation Requirements

**For Go Projects:**
- **MUST** use the `pflag` library for flag parsing
- `pflag` is a drop-in replacement for Go's standard `flag` package
- Implements POSIX/GNU-style flag conventions by default
- Cobra framework utilizes `pflag` automatically

**Rationale for Adherence:**
- **User Expectations:** Users have "hardwired" expectations from decades of terminal conventions
- **Consistency Across Programs:** Users can transfer knowledge from existing tools
- **Script Reliability:** Long options enhance script readability and remain understandable months or years later

---

### Flag Preference Over Positional Arguments

**Mandatory Standard:**
This tool explicitly prefers flags over positional arguments to enhance clarity and future-proofing. Positional arguments MUST be limited to a maximum of one. Two are questionable, and three are strictly forbidden.

#### Rationale

**1. Clarity and Self-Documentation:**
```bash
# ✅ Clear: Flags are self-explanatory
deploy --env production --region us-west-2

# ❌ Unclear: Must remember order and meaning
deploy production us-west-2
```

**2. Future-Proof Extensibility:**
- Flags: Easy to add new options without breaking existing scripts
- Positional arguments: Adding new arguments is a breaking change

**3. Reduced Cognitive Load:**
- Users don't need to remember argument order
- Can specify arguments in any order
- Missing arguments are explicitly named in errors

**4. Discoverability:**
- Flags are visible in `--help` output with descriptions
- Easier to discover available options
- Self-documenting in shell history

#### Acceptable Use of Positional Arguments

**Maximum Allowed:** 1 positional argument
**Common Use Case:** Input files

**Examples:**
```bash
# ✅ Acceptable: Single input file
command input.txt --output result.txt

# ⚠️ Questionable: Two positional arguments
command input.txt result.txt

# ❌ Forbidden: Three positional arguments
command input.txt result.txt config.txt
```

#### Standard I/O Exception

The lone dash (`-`) for stdin/stdout is the one exception to flag preference:
```bash
# ✅ Accepted convention
cat file.txt | command - | other-command
command - --output result.txt  # Read from stdin, write to result.txt
```

---

### Standard I/O Patterns

**Summary:**
Proper use of standard input (stdin), standard output (stdout), and standard error (stderr) is fundamental to CLI composability and UNIX philosophy. These conventions enable tools to work together seamlessly in pipelines and automated systems.

#### Standard Dash (-) Convention

**Rule:** A lone dash (`-`) MUST be interpreted as:
- **stdin** where an input file is expected
- **stdout** where an output file is expected

**Examples:**
```bash
# Read from stdin, write to stdout
cat file.txt | command - | other-command

# Read from stdin, write to file
cat input.txt | command - --output result.txt

# Read from file, write to stdout
command input.txt - | less

# Explicit stdin/stdout in pipelines
tar czf - directory/ | ssh remote "tar xzf -"
```

#### stdout: Successful Command Output

**Purpose:** Reserved exclusively for successful program output

**Rules:**
- MUST contain only the command's primary output
- All informational messages MUST go to stderr, not stdout
- On success, for interactive use, a brief confirmation is acceptable
- For script use, successful commands SHOULD display no output (UNIX tradition)
- Must be structured and parseable when used in pipelines

**Examples:**
```bash
# ✅ Correct: Only data on stdout
find . -name "*.txt"

# ✅ Correct: Silent success for scripts
rm file.txt  # No output on success

# ❌ Wrong: Informational message on stdout
echo "Deleting file..." >&1  # Should go to stderr
```

#### stderr: Everything Else

**Purpose:** All non-data output

**Must Use stderr For:**
- Error messages
- Warning messages
- Informational messages
- Progress indicators
- User prompts
- Debug output
- Logging statements

**Rationale:**
- Keeps stdout clean for piping and redirection
- Allows users to see progress while capturing output
- Enables separate handling of errors and data

**Examples:**
```bash
# ✅ Correct: Info to stderr, data to stdout
echo "Processing files..." >&2
find . -name "*.txt"

# ✅ Correct: Progress to stderr
echo "Progress: 50%" >&2

# ✅ Correct: Errors to stderr
echo "Error: File not found" >&2
exit 1
```

#### Input/Output Specification

**Input Files:**
Input files SHOULD be specified as ordinary arguments

```bash
# ✅ Input as argument
command input.txt
cat file1.txt file2.txt

# ✅ Input from stdin
cat input.txt | command -
command < input.txt
```

**Output Files:**
Output files MUST be specified using an option, preferably `-o` or `--output`

```bash
# ✅ Correct: Output flag
command input.txt --output result.txt
command input.txt -o result.txt

# ✅ Also acceptable: stdout redirection
command input.txt > result.txt

# ⚠️ Questionable: Positional output
command input.txt result.txt  # Unclear which is input/output
```

#### Composability Patterns

**Pipeline Compatibility:**

**Requirements:**
- Accept input from stdin when appropriate
- Output data to stdout
- Keep informational messages on stderr
- Support `-` for stdin/stdout
- Handle broken pipe errors gracefully

**Examples:**
```bash
# ✅ Multi-stage pipeline
cat data.txt | command1 - | command2 --filter active | command3 --json | jq .

# ✅ Parallel processing
find . -name "*.log" | xargs -P 4 command process -

# ✅ Conditional execution
command process input.txt && command notify --message "Done"
```

**Error Handling in Pipes:**
- Exit with non-zero status on errors
- Write errors to stderr, never stdout
- Handle SIGPIPE gracefully when downstream closes

#### UNIX Philosophy Integration

**Rule of Silence:**
- Successful programs should produce no unnecessary output
- Only output what is requested or essential
- Let the user's shell confirm success (via exit code)

**Rule of Composition:**
- Write programs that work together
- Use stdin/stdout as universal interfaces
- Keep output format stable and parseable

---

### Help & Version Commands

**Summary:**
Standard help and version commands are mandatory for all CLI tools. These commands provide essential discoverability, documentation, and version information that users expect from professional command-line interfaces.

#### Mandatory Standard Commands and Flags

All CLI tools MUST implement the following standard commands and flags:

| Flag/Command | Alias | Required Behavior |
|--------------|-------|-------------------|
| `--help` | `-h` | Prints well-formatted, terminal-independent usage information, including command description, options, and examples |
| `--version` | (none) | Prints the program's version number and exits |
| `help [subcommand]` | (none) | Displays the comprehensive manual-style help page for the specified subcommand |

#### --help Flag

**Format:**
- Long form: `--help` (mandatory)
- Short form: `-h` (mandatory)
- Both forms must be implemented

**Behavior:**
- Prints usage information to stdout
- Exits with status code 0 (success)
- Available for all commands and subcommands
- Must work consistently across all levels of command hierarchy

**Content Requirements:**
- Command description (brief summary of what it does)
- Usage syntax (how to invoke the command)
- Available options/flags with descriptions
- Examples demonstrating common usage patterns
- Subcommands (if applicable)

**Formatting Requirements:**
- MUST use formatting (e.g., bold headings) to be easily scannable
- MUST be implemented in a terminal-independent way
- Avoid printing raw escape characters when output is redirected or piped
- Respect terminal width when possible

**Help Output Structure:**

```
NAME
    command - Brief description of what the command does

SYNOPSIS
    command [global flags] <subcommand> [flags] [arguments]

DESCRIPTION
    Detailed description of the command's purpose and behavior.

GLOBAL FLAGS
    -h, --help         Show help information
    --version          Show version information
    -v, --verbose      Enable verbose output
    --json             Output in JSON format

COMMANDS
    init               Initialize a new project
    deploy             Deploy application
    status             Check deployment status

EXAMPLES
    # Example 1: Initialize project
    command init --name my-project

    # Example 2: Deploy to production
    command deploy --env production --region us-west-2

Use "command <command> --help" for more information about a command.
```

**Essential Components:**
1. **Description:** Short, clear description of what the command does
2. **USAGE Section:** Shows the basic command structure with its parameters
3. **OPTIONS/FLAGS Section:** List of available options or flags with explanations
4. **COMMANDS/SUBCOMMANDS Section:** List of available subcommands (if the tool has them)
5. **EXAMPLES Section:** Common usage patterns - often the most-read part of help text

#### --version Flag

**Format:**
- Long form: `--version` (mandatory)
- Short form: typically none (version rarely needs short form)

**Behavior:**
- Prints version number to stdout
- Exits with status code 0 (success)
- Available at the top level only (not typically for subcommands)
- Should be fast (no network calls, no expensive operations)

**Version Format:**
- Should follow Semantic Versioning (SemVer): `MAJOR.MINOR.PATCH`
- May include additional metadata (commit hash, build date)
- Keep format consistent and parseable

**Examples:**
```bash
# ✅ Simple version
$ command --version
command version 1.2.3

# ✅ Detailed version
$ command --version
command version 1.2.3
commit: a1b2c3d
built: 2025-12-23T10:30:00Z
go version: go1.21.0

# ✅ Machine-readable version
$ command --version --json
{"version": "1.2.3", "commit": "a1b2c3d", "buildDate": "2025-12-23T10:30:00Z"}
```

#### help Command (for Subcommands)

**Format:**
- Command form: `help [subcommand]`
- Equivalent to: `[subcommand] --help`

**Examples:**
```bash
# ✅ General help
command help

# ✅ Subcommand help
command help deploy
command help init

# ✅ Equivalent forms
command deploy --help
command help deploy
```

#### Typo Detection and Suggestions

If a user enters an invalid command that is a likely typo of a valid command, the CLI MUST suggest the correction.

**Behavior:**
- Detect similarity between invalid command and valid commands
- Suggest the most likely correct command
- MAY prompt the user to run the suggested command
- MUST NOT automatically run the suggested command

**Examples:**
```bash
# ✅ Helpful suggestion
$ command pss
Error: unknown command "pss" for "command"

Did you mean this?
    ps

Run 'command --help' for usage.
```

**Implementation:**
- Use string similarity algorithms (e.g., Levenshtein distance)
- Suggest commands within 1-2 character edits
- Limit to top 3-5 most similar commands

#### Terminal Independence

**Requirement:** Help text formatting MUST be terminal-independent

**Considerations:**
- Detect if output is a terminal using `term.IsTerminal()`
- Use bold, colors only when output is interactive terminal
- Plain text when piped or redirected
- Respect NO_COLOR environment variable

**Implementation (Go):**
```go
import "golang.org/x/term"

func formatHelp(text string) string {
    if term.IsTerminal(int(os.Stdout.Fd())) {
        // Use colors and bold
        return formatWithColors(text)
    }
    // Plain text
    return text
}
```

---

### Output Design

**Summary:**
Output design follows strict conventions for stream separation, machine readability, and information density to serve both human operators and automated systems.

#### Core Principles

**Standard Output (stdout):**
- Reserved for the successful output of a command
- All informational messages and warnings MUST be written to stderr (not stdout)
- On success in interactive use: brief confirmation message is acceptable
- In scripts (UNIX tradition): successful command SHOULD display no output on stdout

**Standard Error (stderr):**
- All error messages MUST be written to stderr
- All user prompts MUST be written to stderr
- All progress indicators MUST be written to stderr

**Machine-Readable Output:**
- A `--json` flag MUST be implemented
- When passed, all output on stdout MUST be formatted as valid JSON
- Facilitates scripting and composition with tools like `jq`
- JSON output format MUST be treated as a stable, versioned API

**Information Density:**
- Use formatting to increase scannability
- Example: permission strings in `ls` output (`-rwxr-xr-x`)
- Pattern-rich design conveys large amounts of data compactly

**Pager Usage:**
- Commands producing large text output SHOULD automatically pipe to a pager like `less`
- Options `-FIRX` MUST be used with `less`
- Pager MUST only be used if stdout is detected to be an interactive terminal

#### Machine-Readable Output

**JSON Output Flag:**

**Requirement:** A `--json` flag MUST be implemented

**Rules:**
- When `--json` is passed, all output on stdout MUST be valid JSON
- JSON output format MUST be treated as a stable, versioned API
- Informational messages still go to stderr
- Enables composition with tools like `jq`

**Examples:**
```bash
# ✅ Human-readable output (default)
command list
  user-1  active  admin
  user-2  active  user

# ✅ JSON output
command list --json
{"users": [{"id": "user-1", "status": "active", "role": "admin"}]}

# ✅ Compose with jq
command list --json | jq '.users[] | select(.role == "admin")'
```

**Additional Format Options:**
Optional formats to consider:
- `--csv`: Comma-separated values
- `--tsv`: Tab-separated values
- `--yaml`: YAML format
- `--xml`: XML format

#### Stream Detection

**Interactive vs Non-Interactive:**

**Requirement:** Detect if stdout is connected to a terminal

**Use Cases:**
- **Interactive terminal:** Provide colorized, formatted output
- **Pipe or redirection:** Provide plain, parseable output
- **Pager usage:** Only use pager if stdout is a terminal

**Implementation (Go):**
```go
import "golang.org/x/term"

// Check if stdout is a terminal
isTerminal := term.IsTerminal(int(os.Stdout.Fd()))

if isTerminal {
    // Use colors, pager, interactive progress
} else {
    // Plain output for piping
}
```

**Examples:**
```bash
# Interactive: Colorized output with pager
command list

# Piped: Plain output, no pager, no colors
command list | grep pattern

# Redirected: Plain output
command list > output.txt
```

---

### Configuration Management

**Summary:**
Configuration settings are resolved using a mandatory hierarchy where more specific sources override general ones, from explicit program calls down to hardcoded defaults.

#### Precedence Order (Highest to Lowest)

1. **Explicit calls to Set:** Direct programmatic overrides, often for testing or runtime adjustments
2. **Command-line flags:** Explicit runtime parameters representing the user's immediate intent
3. **Environment variables:** Configuration supplied by the operating system or container orchestration environment
4. **Configuration files:** Persistent settings stored in project-specific or user-level files
5. **External key/value stores:** Dynamic configuration retrieved from services like Etcd or Consul
6. **Default values:** Sensible defaults hardcoded into the application to ensure out-of-the-box functionality

#### Comparative Analysis

| Source | Strengths | Weaknesses |
|--------|-----------|------------|
| **Flags** | Highest precedence; most discoverable via `--help`; explicitly shows user intent | Unwieldy for complex configurations; can expose secrets in process list; recorded in shell history |
| **Environment Variables** | Portable; follows 12-factor app methodology; good for cloud-native environments and CI/CD | Lacks structure (key-value only); silent errors if misspelled; can leak to subprocesses; visible in `/proc/<id>/env` |
| **Config Files** | Provides structure for complex data (lists, nested objects); can be version-controlled | Requires file system access and proper management to ensure availability |

#### When to Use Each

**Use Flags when:**
- Running one-off commands
- Need discoverability through --help
- Working interactively

**Use Environment Variables when:**
- Deploying to cloud environments
- Following 12-factor app methodology
- Need lightweight configuration
- Configuring tools in different deployment environments

**Use Configuration Files when:**
- Need structured, complex configuration
- Sharing project settings across a team
- Managing nested or hierarchical settings
- Working with lists and objects

#### XDG Base Directory Specification

**Requirements:**
- The tool MUST adhere to the XDG Base Directory Specification
- Location: Store in `~/.config/<appname>/` rather than `~/.<appname>`
- Avoids cluttering the user's home directory with dotfiles

**Implementation:**
1. **User-level config:** Store in `~/.config/<appname>/`
2. **Project-specific config:** SHOULD search for project-level configuration files (e.g., `.app.yaml`) in the current working directory
3. **Priority:** Project-specific config allows for project-specific overrides of user-level settings

---

### Exit Code Handling

**Summary:**
Exit codes provide the fundamental contract between CLIs and automated systems, signaling success or failure types in a machine-readable way.

#### Primary Convention (Binary Model)

- **Mandate:** Simple binary success/failure model MUST be the default
- **Success:** Exit with status `0`
- **Failure:** Exit with non-zero status (typically `1`)
- **Rationale:** Most common and widely understood convention

#### Extended POSIX Codes

For more granular error reporting useful in scripts, SHOULD use standard POSIX exit codes where applicable:

| Code | Meaning |
|------|---------|
| **0** | Success |
| **1** | General failure |
| **126** | Command found but is not executable |
| **127** | Command not found (e.g., for a subcommand) |

#### Advanced Exit Codes (Optional)

For specific, well-defined failure modes, the tool MAY use descriptive exit codes from BSD `<sysexits.h>` standard (codes 64-78):

**Use Cases:**
- User authentication failure
- Configuration error
- Other specific, well-defined failure modes

**Guidelines:**
- Provides advanced scripting capabilities
- SHOULD be used judiciously
- MUST be documented clearly

#### Purpose

Exit codes are the universal signal that allows complex scripts and automation pipelines (like CI/CD) to function reliably, making decisions based on the success or failure of previous steps. This binary (or multi-level) signal is fundamental to reliable automation.

**The Universal Convention:**
- Exit code 0: Success
- Any non-zero number: Failure

This allows automation pipelines to function reliably:
```bash
# Example: Conditional execution based on exit codes
command process input.txt && command notify --message "Done"
```

---

### Error Message Standards

**Summary:**
Error messages must be human-centric, actionable, and properly routed to stderr, transforming moments of frustration into learning opportunities.

#### Core Principles

**Human-Centric Errors:**
- Low-level system errors MUST be caught and rewritten for humans
- Error message SHOULD function as documentation
- Guide the user toward a solution

**Example Transformation:**
- ❌ Bad: "permission denied"
- ✅ Good: "Can't write to file.txt. You might need to make it writable by running 'chmod +w file.txt'."

**Signal-to-Noise Ratio:**
- Error output MUST be concise and avoid irrelevant information
- Most important information SHOULD be placed at the end of the output
- Multiple similar errors SHOULD be grouped under a single explanatory header
- Avoid printing many redundant lines

**Output Stream:**
- All error messages, without exception, MUST be written to stderr
- stdout is reserved for successful command output

**Debug Information:**
- For unexpected errors, provide mechanism to access detailed traceback
- Information SHOULD be written to a log file OR gated behind a debug flag (e.g., `--debug`)
- Avoid overwhelming the user by default
- Message for unexpected error MUST include:
  - Instructions on how to access detailed information
  - How to submit a bug report

#### Design Philosophy

Clear, actionable error handling is one of the most critical components of a professional CLI. A well-crafted error message can turn a moment of user frustration into a learning opportunity.

**Best Practices:**
1. **Be Descriptive:** Technical errors should be rewritten for humans. An error should explain what went wrong and, if possible, suggest a solution.
2. **Use Separate Streams:** CLIs have two primary output streams: stdout for normal information and stderr for errors and warnings. By writing errors to stderr, tools allow users and scripts to separate successful output from failure messages.
3. **Use Exit Codes:** Every program signals success or failure to the operating system with a numeric exit code.

---

### Cobra Framework Best Practices

**Summary:**
Cobra is the mandatory framework for building professional-grade CLIs in Go, used by industry-standard tools like kubectl.

#### Why Cobra

- **Mandate:** The Cobra framework MUST be used for building the CLI
- **Rationale:** Most popular and widely-used framework for complex, subcommand-based CLIs in the Go ecosystem
- **Key Features:**
  - Automatic help generation
  - Subcommand hierarchy
  - Shell completion support
  - Powers industry-standard tools like kubectl
- **Integration:** Works seamlessly with Viper (configuration) and pflag (flag parsing)
- **Ecosystem Position:** Cobra utilizes pflag by default, which implements POSIX/GNU-style flag conventions

#### Implementation with Cobra

**Automatic Help:**
```go
// Cobra automatically adds --help and -h flags
rootCmd := &cobra.Command{
    Use:   "command",
    Short: "Brief description",
    Long:  "Detailed description with examples",
}
```

**Version Flag:**
```go
// Set version
rootCmd.Version = "1.2.3"

// Cobra automatically adds --version flag
// Customize version template if needed
rootCmd.SetVersionTemplate(`{{.Version}}`)
```

**Custom Help Template:**
```go
// Customize help output format
rootCmd.SetHelpTemplate(`
{{.Long}}

Usage:
  {{.UseLine}}

{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}
{{end}}

{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
```

**Typo Suggestions:**
```go
// Cobra provides suggestions automatically
rootCmd.SuggestionsMinimumDistance = 2
```

#### Prescribed Frameworks for Go CLI Projects

**Primary Framework:**
- **Cobra:** MUST be used for building the CLI

**Configuration Library:**
- **Viper:** MUST be used for all configuration management
- Offers seamless integration with Cobra
- Natural choice for handling configuration from multiple sources in a 12-factor app methodology

**Flag Parsing:**
- **pflag:** MUST be used for all flag parsing
- Drop-in replacement for Go's standard `flag` package
- Implements POSIX/GNU-style flag conventions
- Cobra utilizes pflag by default

---

### CLI UX Guidelines

**Summary:**
The contemporary command-line interface must serve two masters: the human operator who requires clarity, discoverability, and intuitive feedback, and the automated system that demands predictability, composability, and machine-readable output.

#### Human-First Philosophy

**Core Principles:**

**1. Feedback is a Gift**
Commands should never leave users wondering if they're working or broken.

**Responsiveness Requirements:**
- Print something to the user in under 100 milliseconds
- Especially critical before starting long-running tasks (e.g., network requests)
- Assures the user that the program has started and isn't stuck

**Progress Indicators:**
- Use spinners or progress bars for long-running operations
- Show that work is actively happening
- For complex operations, consider multiple parallel progress bars (e.g., `docker pull`)
- Provides crucial insight into task status

**2. Serve Two Masters**

**Interactive Mode (Human Use):**
- Use prompts for critical actions (e.g., deleting resources)
- Prevents accidental, destructive operations
- Provides safety guardrails

**Scriptable Mode (Automation):**
- Provide bypass flags (e.g., `-f` or `--force`)
- Allow commands to run without human intervention
- Enable use in CI/CD pipelines and automated environments

**The Balance:**
Any action that requires a prompt must also have a corresponding flag to bypass that prompt, making the tool versatile for both human and automated use.

#### Prompts and Force Flags

**The Pattern:**

**Interactive Example:**
```bash
$ tool delete-resource
Are you sure you want to delete this resource? (y/N): 
```

**Scripted Example:**
```bash
$ tool delete-resource --force
# Deletes without prompting
```

**Common Force Flag Patterns:**
- `-f` or `--force`: Most common pattern
- `-y` or `--yes`: Alternative pattern (assumes "yes" to all prompts)
- `--no-confirm`: Explicit flag name
- `--assume-yes`: Clear intent flag

**Implementation Guidelines:**
1. **Always provide both modes:** Every destructive or critical operation should support both interactive prompts and bypass flags
2. **Document clearly:** Help text must explain both the interactive behavior and the automation flag
3. **Use consistent naming:** Stick to `-f`/`--force` across your tool for predictability
4. **Make safe by default:** Interactive mode should be the default; force flags opt into danger
5. **Consider dry-run:** For destructive operations, also provide `--dry-run` to preview changes

**Why This Matters:**
This dual-mode design ensures tools are:
- **Safe** for human operators (prompts prevent accidents)
- **Versatile** for automation (force flags enable scripting)
- **Truly useful** in both development and production contexts

#### Design Principles

**Principle: Human-First Design**
- **Rationale:** Traditionally, UNIX commands were designed primarily for other programs. Today, CLIs are often used exclusively by humans. We must shed the baggage of the past and prioritize the human user's experience.
- **Implementation Mandate:** The tool MUST provide clear feedback, helpful error messages, and intuitive interaction patterns over terse, script-optimized output by default. Any operation that may take longer than 100ms MUST provide immediate feedback and never appear to hang.

**Principle: Consistency Across Programs**
- **Rationale:** Users have "hardwired" expectations based on decades of terminal conventions (e.g., POSIX, GNU). Adhering to these established patterns makes a new tool intuitive, guessable, and reduces the cognitive load required to master it.
- **Implementation Mandate:** The tool MUST follow established POSIX and GNU syntax conventions for flags and arguments to ensure a predictable user experience. Standard flags like --help MUST be implemented consistently across all commands.

**Principle: Composability**
- **Rationale:** Designing for composability—where the tool can be used as a building block by other programs and scripts—is not at odds with a human-first design. A well-designed CLI can and should serve both interactive users and automated systems effectively.
- **Implementation Mandate:** The tool MUST provide machine-readable output formats (e.g., JSON via a --json flag) and adhere to standard exit code conventions to facilitate scripting, automation, and integration with other tools like jq.

#### Best Practices Summary

✅ **Do:**
- Implement both `--help` and `-h`
- Provide examples in help text
- Use clear, scannable formatting
- Make help terminal-independent
- Suggest corrections for typos
- Keep help concise but informative
- Show version in consistent format
- Use stdout only for program output/data
- Write all messages (info, warnings, errors) to stderr
- Implement `--json` for machine-readable output
- Support `-` for stdin/stdout
- Detect terminal vs pipe/redirect
- Keep output format stable and documented

❌ **Don't:**
- Output raw ANSI escape codes to non-terminals
- Make help text too verbose (save details for `man` pages or docs)
- Forget to update help when adding features
- Automatically execute suggested commands
- Write help to stderr (it's not an error)
- Write error messages to stdout
- Write progress indicators to stdout
- Assume stdout is always a terminal
- Break output format in minor versions
- Ignore SIGPIPE errors
- Output ANSI codes to non-terminals

---

## Usage

Consult this agent for CLI-specific questions including:
- Command design and structure
- Flag conventions and positional arguments
- Output formatting (stdout/stderr separation, machine-readable formats)
- Help and version command implementation
- Error message design
- Exit code handling
- Configuration management and precedence
- Cobra framework usage in Go
- Terminal detection and independence
- Pipeline composability
- Human-first UX design
- Interactive vs. scriptable modes
- POSIX/GNU compliance

This agent provides authoritative standards for building professional-grade command-line tools that serve both human operators and automated systems effectively.
