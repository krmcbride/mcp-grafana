# Dependency Management

## Vendored Dependencies

This project uses Go vendoring to provide direct access to dependency source code:

- **Vendor Directory**: `vendor/` contains all dependency source code locally
- **Git Ignore**: Vendor directory is excluded from version control
- **Makefile Integration**: All Go commands use `GOFLAGS="-mod=vendor"` automatically
- **Development Workflow**: Run `go mod vendor` after dependency changes to update vendor/

**Claude Development Benefits:**

- Can read actual source code from `vendor/` instead of relying on external documentation
- Enables precise API research by examining actual method signatures and implementations
- Provides accurate error handling patterns and usage examples directly from source
- **IMPORTANT**: Always prefer reading vendor source code over guessing API structures

**Vendor Maintenance:**

- Run `go mod vendor` after updating `go.mod`
- Vendor directory is automatically excluded from git commits
- All build/test/lint commands automatically use vendored dependencies

## Key Dependencies

- `github.com/mark3labs/mcp-go`: MCP protocol implementation

## Tool Version Management

**Systematic Update Process:**

1. **Check Latest Versions:**
   - Use WebFetch tool to check GitHub releases pages for each tool
   - Target repositories: `incu6us/goimports-reviser`, `mvdan/gofumpt`, `golangci/golangci-lint`, `f/mcptools`
   - Look for latest release tags

2. **Update Makefile Versions:**
   - Update version constants in `Makefile` lines ~37-40
   - Verify package paths are correct (e.g., golangci-lint path changes between v1 and v2)
   - Test that download URLs work by cleaning `bin/` and running make targets

3. **Validation Process:**
   - Run `rm -rf bin/*` to force fresh downloads
   - Test `make format-ci` and `make lint-ci` to ensure compatibility
   - Check for deprecated flags or changed behavior in newer versions
   - Run `make build` and `make test` to ensure everything still works

4. **Documentation:**
   - Update this section if tool behavior significantly changes
   - Note any breaking changes or new features in commit messages

**Tool-Specific Notes:**

- **goimports-reviser**: v3.9.1+ is more verbose but stricter about import formatting
- **gofumpt**: v0.8.0+ requires Go 1.23+ and includes new formatting rules
- **golangci-lint**: v2.x series has different package paths and configuration format
