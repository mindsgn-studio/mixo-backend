# AGENTS.md - Project Rules for mixo-backend

## Go Best Practices

### Code Style
- Follow standard Go conventions: `gofmt`, `go vet`, `golangci-lint`
- Use meaningful variable/function names; avoid abbreviations unless widely understood
- Keep functions small and focused (< 50 lines preferred)
- Handle all errors explicitly; never use `_` for error returns unless in deferred close
- Use `fmt.Errorf` with `%w` for error wrapping
- Prefer table-driven tests
- Use `t.Helper()` in test helper functions
- Use `t.Cleanup()` over `defer` in tests for resource cleanup
- Export only what's necessary; prefer unexported types within packages

### Package Structure
- `cmd/` - Entry points only; no business logic
- `internal/` - Private packages organized by domain
- Each package should have a clear, single responsibility
- Avoid circular dependencies
- Keep `go.mod` tidy; run `go mod tidy` after dependency changes

### Error Handling
```go
// Good
if err != nil {
    return fmt.Errorf("failed to process song: %w", err)
}

// Bad
if err != nil {
    return err
}
```

### Database
- Use parameterized queries; never interpolate user input into SQL
- Use transactions for multi-step operations
- Always check and handle `sql.ErrNoRows` explicitly
- Use `defer rows.Close()` or `defer func() { _ = rows.Close() }()`

### HTTP Handlers
- Validate request methods at the start of handlers
- Return appropriate HTTP status codes
- Use `http.Error()` for error responses
- Check all `w.Write()` return values
- Set Content-Type headers before writing response body

## Versioning Rules

### Semantic Versioning (SemVer)
- **MAJOR** (X.0.0): Breaking changes to API or database schema
- **MINOR** (0.X.0): New features, backward compatible
- **PATCH** (0.0.X): Bug fixes, backward compatible

### Version Location
- Current version is defined in `cmd/server/main.go` as `const version`
- Update version before each release
- Version format: `MAJOR.MINOR.PATCH`

### Database Schema Changes
- Add new columns with `DEFAULT` values for backward compatibility
- Never remove columns without a migration plan
- Test migrations with existing data

## Git Workflow Rules

### Branch Strategy
- `main` - Production-ready code
- `staging` - Pre-production testing
- Feature branches - `feat/description`, `fix/description`, `chore/description`

### For Every Change (Agent Workflow)
1. **Create a new branch** from `staging`:
   ```bash
   git checkout staging
   git pull origin staging
   git checkout -b feat/description
   ```

2. **Write tests first** (TDD when possible):
   - Add test files alongside implementation
   - Ensure all tests pass before proceeding

3. **Make the changes**:
   - Follow Go best practices
   - Keep commits atomic and focused
   - Update documentation if needed

4. **Run tests**:
   ```bash
   go test -v -race ./...
   go build ./...
   ```

5. **Update version** in `cmd/server/main.go`

6. **Commit with descriptive message**:
   ```bash
   git add -A
   git commit -m "feat: add music crawler worker

   - Scans configured directories for MP3 files
   - Extracts metadata and updates database
   - Handles duplicate detection
   - Runs on startup and configurable interval"
   ```

7. **Push and create PR**:
   ```bash
   git push origin feat/description
   ```
   - Create PR from feature branch to `staging`
   - PR title should be concise and descriptive
   - PR description should explain what and why

### Commit Message Format
```
<type>: <short description>

<optional body>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `chore`: Maintenance tasks
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements

### PR Guidelines
- Keep PRs small and focused (< 400 lines preferred)
- Include tests for new functionality
- Update README if adding user-facing features
- Ensure CI passes before merging

## Testing Requirements

### Unit Tests
- Test all public functions
- Use table-driven tests for multiple scenarios
- Mock external dependencies (file system, network)
- Aim for > 80% coverage on business logic

### Integration Tests
- Test database operations with real SQLite (in-memory or temp files)
- Test HTTP handlers with `httptest`
- Clean up test resources with `t.Cleanup()`

### Test Naming
```go
func TestFunctionName_Scenario_ExpectedResult(t *testing.T) {
    // ...
}
```
