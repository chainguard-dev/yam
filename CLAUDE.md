# Claude Development Guidelines for yam

This file contains guidelines for Claude when working on the yam YAML formatter project.

## Pre-commit Checklist

Before committing any changes, **ALWAYS** run the following commands in order:

### 1. Run Tests
```bash
# Run all tests to ensure nothing is broken
go test ./...

# For new features, add comprehensive test cases to the appropriate _test.go file
# Ensure test coverage includes edge cases, especially around YAML parsing and formatting
```

### 2. Format Code
```bash
# Format all Go files to ensure consistent style
gofmt -w .

# Alternative: use goimports if available
goimports -w .
```

### 3. Run yam on Changed Files
```bash
# Build the latest version
go build

# Run yam on any YAML files that have been modified to ensure they're properly formatted
# Example for test files:
./yam pkg/yam/formatted/testdata/format/*.yaml

# Run yam on any configuration files in the repo
./yam .yam.yaml
```

### 4. Run Linter
```bash
# Install golangci-lint if not available
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the linter to catch style and potential issues
golangci-lint run

# Fix any issues reported by the linter before committing
```

## Development Practices

### Testing Requirements
- **Always add tests** for new functionality
- **Test edge cases**, especially:
  - YAML files with comments
  - Complex YAML structures (nested mappings, sequences)
  - Different YAML node types (scalar, mapping, sequence)
  - Error conditions and malformed YAML
- **Use table-driven tests** when testing multiple similar scenarios
- **Name tests descriptively** to explain what they're testing

### Code Quality
- **Follow Go conventions** for naming, structure, and documentation
- **Add comments** for complex logic, especially around YAML AST manipulation
- **Handle errors properly** - don't ignore error returns
- **Use meaningful variable names** that explain the purpose

### YAML-Specific Considerations
- **Preserve comments and formatting** when possible
- **Test with real-world YAML files** that contain comments, complex structures
- **Be careful with path construction** - ensure comments don't break path matching
- **Consider different YAML node kinds** (Document, Mapping, Sequence, Scalar)

### yam Tool Usage
- **Test formatting features** with actual yam commands
- **Verify config file handling** (`.yam.yaml` integration)
- **Test CLI flag combinations** to ensure they work together properly
- **Check sorting, gap insertion, and deduplication** features work correctly

## Commit Message Format

Use conventional commit format:
```
type: brief description

Longer description explaining:
- What was changed and why
- Any breaking changes
- How to test the changes

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Common Commands

```bash
# Full development cycle
go test ./...                    # Run tests
gofmt -w .                      # Format code
go build                        # Build binary
./yam testfile.yaml             # Test on sample file
golangci-lint run              # Check for issues

# Run specific test packages
go test ./pkg/yam/formatted -v  # Run formatter tests with verbose output
go test -run TestSorting        # Run specific test functions

# Debug test failures
go test -v -run TestName        # Verbose output for specific test
go test -race ./...             # Check for race conditions
```

## Files to Always Check

When making changes, always verify these files are properly formatted:
- `.yam.yaml` (project config)
- `pkg/yam/formatted/testdata/format/*.yaml` (test data)
- Any YAML files added for testing

## Linter Exceptions

If you need to ignore specific linter warnings, use:
```go
//nolint:linter-name // explanation of why this is needed
```

Only ignore linter warnings when absolutely necessary and always include an explanation.

## Project Structure Notes

- `pkg/yam/formatted/encoder.go` - Core YAML formatting logic
- `pkg/yam/formatted/encoder_test.go` - Tests for formatter
- `pkg/yam/formatted/path/` - YAML path parsing and matching
- `pkg/cmd/root.go` - CLI command handling and config precedence
- `pkg/yam/apply.go` - Main formatting application logic

When adding features, follow the existing package structure and patterns.