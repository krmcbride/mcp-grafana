# Development Best Practices

## Architecture-First Approach

When implementing new features, start with architectural planning:

- Identify key interfaces and integration points
- Consider consistency requirements across related operations (e.g., registration and lookup)
- Anticipate potential edge cases and normalization needs

**API Research Protocol**

- Use Context7 or WebSearch to proactively to research exact API structures before implementation
- For unfamiliar libraries or APIs, always verify field names, types, and methods
- Don't assume API structures - verify through documentation before coding

**Incremental Development**

- Break complex features into smaller, testable increments
- Validate each component with `make build` and `make test` before proceeding
- Test intermediate states rather than implementing entire features before first validation
- **CRITICAL**: Build and test after each significant change to catch issues early
- Use `git status` to verify file changes before and after bulk operations

## Modern Go Guidelines

**Type Declarations**

- Use `any` instead of `interface{}` for better readability
- Example: `var content any` instead of `var content interface{}`
- This applies to function parameters, return types, and variable declarations

**Safe File Manipulation**

- **AVOID** batch `sed` operations on multiple files - they can corrupt files
- Use `MultiEdit` tool for multiple related changes in a single file
- Use individual `Edit` calls for changes across multiple files
- Always check `git status` before and after bulk operations
- Use `git restore <file>` to recover from file corruption
- Test build after any file manipulation to ensure no corruption

**Error Handling**

- Avoid error variable shadowing in nested scopes
- Use descriptive error variable names when redeclaring in inner scopes
- Example: Use `parseErr` instead of redeclaring `err` in parsing operations

**Safety-First Design**

- All MCP tools are deliberately read-only to prevent accidental cluster modifications
- Tools provide comprehensive data for analysis without mutation capabilities
- Resource mappers extract relevant fields while preserving original structure

**Resource and Performance Considerations**

- Consider MCP token limits when designing features (25k response limit)
- Account for memory usage patterns, especially for large resource collections
- Design with pagination and chunking in mind for scalable operations
- Document resource usage implications (e.g., "100 Events ≈ 12k tokens")

**Backward Compatibility**

- When adding new optional parameters, ensure existing usage patterns remain functional
- Test that new features don't break existing tool behavior
- Design APIs to be extensible without breaking changes

**MCP Server Logging**

- **CRITICAL**: When using stdio transport, all logging MUST go to stderr only
- The MCP protocol uses stdout for communication; any output to stdout will corrupt the protocol
- Use `fmt.Fprintf(os.Stderr, ...)` or configure loggers to write to stderr
- This applies to all debug messages, status updates, and error logging
- Reference: MCP specification states servers can write logs to stderr while using stdout for protocol messages

## Documentation-Driven Development

- Write comprehensive comments for public APIs during implementation, not after
- Explain design decisions and component relationships
- Include usage examples for complex interfaces
- Maintain package comments in the primary file of each package (except `main`):

## Test-Driven Validation

- Propose and implement tests early in the feature development process
- Use tests to validate design assumptions and catch integration issues
- Test edge cases like case variations, empty inputs, and error conditions
- **Safety Protocol**: Always run `make build` and `make test` after file modifications
- If tests fail after bulk file operations, check for file corruption with `git status`
- Use incremental approach: make one change, test, then proceed to next change
