# Contributing to SDBX

Thank you for your interest in contributing to SDBX! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)

## Code of Conduct

This project follows a simple code of conduct:
- Be respectful and constructive
- Welcome newcomers and help them learn
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sdbx.git
   cd sdbx
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/maiko/sdbx.git
   ```

## Development Setup

### Prerequisites

- Go 1.25.5 or later
- Make
- Docker and Docker Compose (for integration testing)
- Git

### Building from Source

```bash
# Install dependencies
go mod download

# Build the binary
make build

# Run tests
make test

# Run linter
make lint
```

### Running Locally

```bash
# Build and run
./bin/sdbx --help

# Initialize a test project
./bin/sdbx init --skip-wizard \
  --domain test.local \
  --expose lan \
  --admin-password testpass123
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-new-addon` - New features
- `fix/auth-bug` - Bug fixes
- `docs/update-faq` - Documentation updates
- `test/improve-coverage` - Test improvements
- `refactor/cleanup-docker` - Code refactoring

### Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**:
   - Write clear, concise code
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   make test
   make lint
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

5. **Keep your branch updated**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

## Testing

### Test Requirements

- All new code **must** include tests
- Tests must pass before PR can be merged
- Maintain or improve test coverage (target: 80%)
- Use table-driven tests where appropriate

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run with race detector
go test -race ./...

# Run specific package
go test ./internal/config

# Run specific test
go test -run TestValidate ./internal/config
```

### Writing Tests

Example test structure:

```go
func TestYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := YourFunction(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("YourFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if result != tt.expected {
                t.Errorf("YourFunction() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Code Style

### Go Style Guidelines

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by CI)
- Run `go vet` to catch common mistakes
- Use meaningful variable and function names
- Keep functions small and focused
- Add comments for exported functions and types

### Format Before Committing

```bash
# Format all Go files
go fmt ./...

# Run goimports (if installed)
goimports -w .
```

### Linting

The project uses `golangci-lint` with custom configuration:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Auto-fix issues (where possible)
golangci-lint run --fix
```

## Commit Messages

### Format

Follow the Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `style`: Code style changes (formatting, etc.)
- `perf`: Performance improvements
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

### Examples

```
feat(addon): add Jellyfin addon support

Add support for Jellyfin as an alternative to Plex.
Includes configuration templates and health checks.

Closes #123
```

```
fix(secrets): prevent backup overwrite during rotation

Backup files now include timestamps to prevent accidental
overwrites when rotating secrets multiple times.

Fixes #456
```

```
docs(readme): update installation instructions

Add missing prerequisites and troubleshooting steps.
```

### Guidelines

- Use imperative mood ("add feature" not "added feature")
- First line should be 50 characters or less
- Body should wrap at 72 characters
- Reference issues and PRs where applicable

## Pull Request Process

### Before Submitting

1. ✅ Ensure all tests pass locally
2. ✅ Run linter and fix all issues
3. ✅ Update documentation if needed
4. ✅ Add tests for new functionality
5. ✅ Rebase on latest `main` branch
6. ✅ Write clear commit messages

### PR Description

Include in your PR description:

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manually tested

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings generated
```

### Review Process

1. **Automated checks** will run (tests, linting, coverage)
2. **Maintainer review** - usually within 2-3 days
3. **Address feedback** by pushing new commits
4. **Approval and merge** once all checks pass

### After Merge

- Your PR will be included in the next release
- You'll be credited in the CHANGELOG
- Close any related issues

## Issue Reporting

### Before Creating an Issue

1. **Search existing issues** to avoid duplicates
2. **Check documentation** for known solutions
3. **Update to latest version** to see if issue persists

### Bug Reports

Include:
- **Description**: Clear description of the bug
- **Steps to Reproduce**: Numbered steps to trigger the bug
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Environment**:
  - SDBX version (`sdbx version`)
  - OS and version
  - Docker version
  - Relevant configuration
- **Logs**: Include relevant error messages
- **Screenshots**: If applicable

### Feature Requests

Include:
- **Problem Statement**: What problem does this solve?
- **Proposed Solution**: How should it work?
- **Alternatives**: Other approaches considered
- **Use Case**: Real-world scenario
- **Willingness to Contribute**: Can you implement this?

### Questions

- Use GitHub Discussions for questions
- Search existing discussions first
- Provide context about what you're trying to achieve

## Development Tips

### Project Structure

```
sdbx/
├── cmd/sdbx/          # Main application
│   └── cmd/           # CLI commands
├── internal/          # Internal packages
│   ├── config/        # Configuration management
│   ├── docker/        # Docker Compose wrapper
│   ├── doctor/        # Health checks
│   ├── generator/     # Project generation
│   ├── secrets/       # Secret management
│   └── tui/           # Terminal UI components
├── docs/              # Documentation
├── .github/           # GitHub workflows
└── Makefile           # Build targets
```

### Useful Commands

```bash
# Run specific linters
golangci-lint run --disable-all --enable=errcheck

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Build for specific platform
GOOS=darwin GOARCH=arm64 go build -o bin/sdbx-darwin-arm64 ./cmd/sdbx

# Update dependencies
go get -u ./...
go mod tidy
```

### Debugging

```bash
# Enable verbose output
./bin/sdbx --debug up

# Use delve debugger
dlv debug ./cmd/sdbx -- init
```

## Getting Help

- **Documentation**: Check the `/docs` directory
- **Discussions**: Use GitHub Discussions for questions
- **Issues**: Report bugs and feature requests
- **Discord**: [Join our community] (if available)

## Recognition

Contributors are recognized in:
- CHANGELOG.md for each release
- GitHub contributors page
- README.md (for significant contributions)

Thank you for contributing to SDBX! 🎉
