# ChronoGo Build Scripts

This directory contains build scripts for ChronoGo, providing cross-platform support for building, testing, linting, and cleaning the project.

## Available Scripts

### PowerShell Script (`build.ps1`)

For Windows users or anyone with PowerShell installed:

```powershell
# Build in debug mode
.\scripts\build.ps1

# Build with all steps (clean, lint, build, test)
.\scripts\build.ps1 -All

# Build in release mode
.\scripts\build.ps1 -Release

# Run tests after building
.\scripts\build.ps1 -Test

# Clean build artifacts
.\scripts\build.ps1 -Clean

# Run linter
.\scripts\build.ps1 -Lint

# Specify custom output name
.\scripts\build.ps1 -Output "custom_name.exe"
```

### Bash Script (`build.sh`)

For Linux, macOS, or Git Bash users:

```bash
# Build in debug mode
./scripts/build.sh

# Display help
./scripts/build.sh --help

# Build with all steps (clean, lint, build, test)
./scripts/build.sh --all

# Build in release mode
./scripts/build.sh --release

# Run tests after building
./scripts/build.sh --test

# Clean build artifacts
./scripts/build.sh --clean

# Run linter
./scripts/build.sh --lint

# Specify custom output name
./scripts/build.sh --output custom_name
```

### Makefile

For users who prefer Make:

```bash
# Build in debug mode
make build

# Build with all steps (clean, lint, build, test)
make all

# Build in release mode
make release

# Run tests
make test

# Clean build artifacts
make clean

# Run linter
make lint

# Display help
make help
```

## GitHub Actions

The project also includes GitHub Actions workflows for CI/CD:

- Automatic building and testing on push and pull requests
- Multi-platform builds (Windows, Linux, macOS) with multiple Go versions
- Automatic release creation when pushing version tags

## Requirements

- Go 1.20 or later
- PowerShell 5+ (for Windows builds using build.ps1)
- Bash (for Unix builds using build.sh)
- Make (optional, for using the Makefile)
- golangci-lint (can be installed with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`) 