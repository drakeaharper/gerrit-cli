# gerry - Gerrit CLI Tool

A command-line interface for interacting with Gerrit Code Review, designed for developers who prefer terminal workflows over web interfaces.

## Features

- **Easy Setup**: Interactive configuration wizard with `gerry init`
- **List Changes**: View your open changes with `gerry list`
- **Review Comments**: Read review comments directly in terminal with `gerry comments`
- **Change Details**: Get comprehensive change information with `gerry details`
- **Local Workflow**: Fetch and cherry-pick changes with `gerry fetch` and `gerry cherry-pick`

## Installation

### From Source

1. **Prerequisites**
   - Go 1.21 or later
   - Git
   - SSH access to your Gerrit server
   - Gerrit HTTP password (found in Gerrit Settings → HTTP Password)

2. **Clone the repository**
   ```bash
   git clone https://github.com/drakeaharper/gerrit-cli.git
   cd gerrit-cli
   ```

3. **Build and install**
   ```bash
   # Easy install (recommended - handles PATH automatically)
   ./install.sh
   
   # Or manually
   make install
   
   # Or install to $GOPATH/bin using go install
   make install-go
   ```

4. **Initialize configuration**
   ```bash
   gerry init
   ```

### Using Go Install

```bash
go install github.com/drakeaharper/gerrit-cli/cmd/gerry@latest
gerry init
```

## Quick Start

1. **Initialize gerry** (first time only)
   ```bash
   gerry init
   ```
   This will walk you through setting up your Gerrit server connection.

2. **List your open changes**
   ```bash
   gerry list
   ```

3. **View comments on a change**
   ```bash
   gerry comments 384465
   ```

4. **Fetch a change locally**
   ```bash
   gerry fetch 384465
   ```

## Configuration

Configuration is stored in `~/.gerry/config.json`. You can also use environment variables:

- `GERRIT_SERVER`: Gerrit server hostname
- `GERRIT_PORT`: SSH port (default: 29418)
- `GERRIT_USER`: Your Gerrit username
- `GERRIT_HTTP_PASSWORD`: Your HTTP password
- `GERRIT_PROJECT`: Default project

### Configuration File Format

```json
{
  "server": "gerrit.example.com",
  "port": 29418,
  "http_port": 8080,
  "user": "your-username",
  "http_password": "your-http-password",
  "ssh_key": "/path/to/ssh/key",
  "project": "default-project"
}
```

**Note about ports:**
- `port`: SSH port (usually 29418)
- `http_port`: HTTP/HTTPS port for REST API (common values: 443, 8080, 8443)
  - If not specified, auto-detection will try to determine the correct port
  - For SSH port 29418, it defaults to HTTPS on port 443

Environment variables take precedence over configuration file values.

## Commands

### `gerry init`
Interactive setup wizard that configures your Gerrit connection.

### `gerry list`
List your open changes.
- `--detailed`: Show detailed information including patch set numbers
- `--reviewer`: Show changes that need your review

### `gerry comments <change-id>`
View comments on a specific change.
- `--all`: Show all comments (default: unresolved only)

### `gerry details <change-id>`
Show comprehensive information about a change including files, reviewers, and scores.

### `gerry fetch <change-id> [patchset]`
Fetch a change and checkout to FETCH_HEAD. If patchset is not specified, fetches the current patch set.

### `gerry cherry <change-id> [patchset]`
Fetch and cherry-pick a change. If patchset is not specified, uses the current patch set.
- `--no-commit`: Don't commit the cherry-pick
- `--no-verify`: Skip git hooks during cherry-pick

Also available as `gerry cherry-pick` for familiarity with git.

### `gerry update`
Update gerry to the latest version by pulling from git and rebuilding. Must be run from the source directory.
- `--skip-pull`: Skip git pull and just rebuild

## Development

### Building from Source

```bash
# Build for current platform
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Cross-compile for multiple platforms
make build-all

# Clean build artifacts
make clean

# Format code
make fmt

# Run go vet
make vet

# Install dependencies
make deps
```

### Makefile Targets

- `make build` - Build the binary for current platform
- `make install` - Build and install to best available location
- `make install-go` - Build and install to $GOPATH/bin
- `make update` - Clean, build, and install (for development)
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make deps` - Download and tidy dependencies
- `make clean` - Remove build artifacts
- `make build-linux` - Build for Linux AMD64
- `make build-windows` - Build for Windows AMD64
- `make build-darwin` - Build for macOS (Intel and Apple Silicon)
- `make build-all` - Build for all platforms
- `make run` - Run the application directly
- `make fmt` - Format Go code
- `make vet` - Run go vet
- `make lint` - Run golangci-lint (requires golangci-lint installation)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT
