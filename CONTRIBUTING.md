# Contributing to Router Sync

Thank you for your interest in contributing to Router Sync! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [Commit Message Convention](#commit-message-convention)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Release Process](#release-process)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make
- Docker (optional, for containerized development)

### Initial Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/fcastello/router-sync.git
   cd router-sync
   ```

2. **Install development tools**
   ```bash
   make install-tools
   ```

3. **Install dependencies**
   ```bash
   make deps
   ```

4. **Generate API documentation**
   ```bash
   make docs
   ```

### Development Commands

```bash
# Build the application
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Run all checks (fmt, vet, lint, test)
make check

# Run locally
make run

# Run with debug logging
make run-debug

# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

## Commit Message Convention

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages. This enables automatic changelog generation and semantic versioning.

### Commit Message Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to the build process or auxiliary tools and libraries

### Examples

```bash
# New feature
git commit -m "feat: add IPv6 support for routing policies"

# Bug fix
git commit -m "fix: resolve memory leak in NATS connection"

# Documentation
git commit -m "docs: update API documentation with new endpoints"

# Breaking change
git commit -m "feat!: change API response format for providers

BREAKING CHANGE: The provider API now returns additional fields and the response structure has changed."
```

### Scope (Optional)

You can specify a scope to provide additional contextual information:

```bash
git commit -m "feat(api): add new endpoint for bulk operations"
git commit -m "fix(nats): handle connection timeout properly"
git commit -m "docs(readme): update installation instructions"
```

## Development Workflow

### 1. Create a Feature Branch

```bash
# Create and switch to a new feature branch
git checkout -b feat/your-feature-name

# Or for bug fixes
git checkout -b fix/your-bug-description
```

### 2. Make Your Changes

- Write your code following the project's coding standards
- Add tests for new functionality
- Update documentation as needed

### 3. Run Quality Checks

```bash
# Run all checks before committing
make check
```

### 4. Commit Your Changes

```bash
# Stage your changes
git add .

# Commit with a conventional commit message
git commit -m "feat: add your new feature description"
```

### 5. Push and Create Pull Request

```bash
# Push your branch
git push origin feat/your-feature-name
```

Then create a Pull Request on GitHub.

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run benchmarks
make bench
```

### Writing Tests

- Write tests for all new functionality
- Follow the existing test patterns in the codebase
- Aim for good test coverage (80%+)
- Use descriptive test names

### Test Structure

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange
    // Set up test data and mocks
    
    // Act
    // Call the function being tested
    
    // Assert
    // Verify the results
}
```

## Release Process

### Version Management

The project uses semantic versioning (MAJOR.MINOR.PATCH). Version information is stored in the `VERSION` file.

```bash
# Check current version
make version

# Bump patch version (0.1.0 -> 0.1.1)
make version-bump-patch

# Bump minor version (0.1.0 -> 0.2.0)
make version-bump-minor

# Bump major version (0.1.0 -> 1.0.0)
make version-bump-major
```

### Changelog Generation

```bash
# Generate changelog for next version
make changelog

# Preview changelog for next release
make changelog-preview
```

### Release Workflow

The correct release workflow follows these steps in order:

#### Step 1: Generate Changelog (Before Version Bump)
```bash
# Generate changelog for the next version
make changelog

# Review the generated CHANGELOG.md file
# Edit if needed, then commit
git add CHANGELOG.md
git commit -m "docs: update changelog for v0.1.1"
```

#### Step 2: Bump Version and Create Tag
```bash
# Bump version (this also creates the git tag)
make version-bump-patch  # or version-bump-minor / version-bump-major

# Push changes and tag to GitHub
git push origin main
git push origin v0.1.1
```

#### Step 3: Create Release Artifacts
```bash
# Build binaries and create tarballs
make release
```

#### Step 4: Create GitHub Release
```bash
# Create GitHub release with changelog and artifacts
make release-github
```

### Alternative: Use the Workflow Helper

For convenience, you can use the workflow helper that guides you through the process:

```bash
make release-workflow
```

This will show you the exact steps to follow.

### Complete Release Example

Here's a complete example for releasing version 0.1.1:

```bash
# 1. Generate changelog
make changelog

# 2. Review and commit changelog
git add CHANGELOG.md
git commit -m "docs: update changelog for v0.1.1"

# 3. Bump version (creates tag v0.1.1)
make version-bump-patch

# 4. Push changes and tag
git push origin main
git push origin v0.1.1

# 5. Create release artifacts
make release

# 6. Create GitHub release
make release-github
```

### Release Checklist

- [ ] All tests pass (`make check`)
- [ ] Documentation is up to date
- [ ] Changelog is generated and reviewed
- [ ] Version is bumped and tagged
- [ ] Changes and tag are pushed to GitHub
- [ ] Release artifacts are built
- [ ] GitHub release is created with proper notes
- [ ] Docker image is built and pushed (if applicable)

### Common Issues and Solutions

#### "Tag already exists" Error
If you get an error like `ERROR "v0.1.1" tag already exists`, it means you're trying to generate a changelog for a version that already has a tag. The solution is to:

1. **Generate changelog BEFORE bumping version**
2. **Or bump to the next version first**

#### "git-chglog: No such file or directory"
Install the required tools:
```bash
make install-tools
```

#### Tarball Creation Fails
The release process now creates tarballs in the `release/` directory at the project root. If you encounter issues, ensure you have write permissions in the project directory.

## Pull Request Process

### Before Submitting

1. **Ensure your code follows the project standards**
   ```bash
   make check
   ```

2. **Update documentation** if your changes affect:
   - API endpoints
   - Configuration options
   - Installation process
   - Usage examples

3. **Add tests** for new functionality

4. **Update changelog** if your changes are user-facing

### Pull Request Guidelines

- **Title**: Use conventional commit format (e.g., "feat: add IPv6 support")
- **Description**: Clearly describe what the PR does and why
- **Related Issues**: Link to any related issues
- **Breaking Changes**: Clearly mark any breaking changes
- **Testing**: Describe how to test your changes

### Example Pull Request

```markdown
## Description
Adds IPv6 support for routing policies, allowing users to specify IPv6 addresses and CIDR ranges in their routing policies.

## Changes
- Add IPv6 address validation in models
- Update API handlers to support IPv6 addresses
- Add tests for IPv6 functionality
- Update documentation with IPv6 examples

## Testing
- [x] Unit tests pass
- [x] Integration tests pass
- [x] Manual testing with IPv6 addresses
- [x] Documentation updated

## Breaking Changes
None

## Related Issues
Closes #123
```

### Review Process

1. **Automated Checks**: All PRs must pass CI checks
2. **Code Review**: At least one maintainer must approve
3. **Testing**: Changes must be properly tested
4. **Documentation**: Documentation must be updated if needed

## Code Style Guidelines

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to process data: %w", err)
}

// Avoid
if err != nil {
    return err
}
```

### Logging

```go
// Use structured logging with logrus
logrus.WithFields(logrus.Fields{
    "provider_id": providerID,
    "error":       err,
}).Error("Failed to create provider")
```

## Getting Help

- **Issues**: Use GitHub Issues for bug reports and feature requests
- **Discussions**: Use GitHub Discussions for questions and general discussion
- **Documentation**: Check the README and API documentation

## License

By contributing to Router Sync, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Router Sync! 🚀 