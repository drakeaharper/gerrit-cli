# gerry - Gerrit CLI Tool

A command-line interface for interacting with Gerrit Code Review, designed for developers who prefer terminal workflows over web interfaces.

## Features

- **Easy Setup**: Interactive configuration wizard with `gerry init`
- **List Changes**: View your open changes with `gerry list`
- **Review Comments**: Read review comments directly in terminal with `gerry comments`
- **Change Details**: Get comprehensive change information with `gerry details`
- **Local Workflow**: Fetch and cherry-pick changes with `gerry fetch` and `gerry cherry-pick`
- **Cross-Repo Analysis**: Analyze merged changes across all repositories with `gerry analyze`

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

## Usage Examples

### Daily Workflow

**Morning code review routine:**
```bash
# See what changes need your review
gerry list --reviewer

# Check changes where you're CC'd for awareness
gerry list --cc

# Check details on an interesting change
gerry details 384465

# Read through the comments
gerry comments 384465

# Fetch the change to test locally
gerry fetch 384465

# After testing, cherry-pick it to your branch
git checkout my-feature-branch
gerry cherry 384465
```

**Working with your own changes:**
```bash
# List your open changes
gerry list

# Get detailed view with files
gerry list --detailed

# Check a specific change with file listing
gerry details 384465 --files

# View only unresolved comments
gerry comments 384465

# View all comments (resolved and unresolved)
gerry comments 384465 --all
```

### Analysis Workflows

**Generate contribution reports:**
```bash
# Analyze all repos for the current year
gerry analyze --start-date 2025-01-01 --end-date 2025-12-31

# Analyze a specific repository
gerry analyze --repo canvas-lms --start-date 2025-01-01

# Export to different formats
gerry analyze --start-date 2025-01-01 --format json -o report.json
gerry analyze --start-date 2025-01-01 --format csv -o report.csv

# Monthly team report
gerry analyze --start-date 2025-11-01 --end-date 2025-11-30 -o nov_report.md
```

### Advanced Usage

**Cherry-picking workflows:**
```bash
# Cherry-pick without committing (for review/modification)
gerry cherry 384465 --no-commit

# Cherry-pick a specific patchset
gerry cherry 384465 3

# Cherry-pick skipping git hooks
gerry cherry 384465 --no-verify
```

**Fetching workflows:**
```bash
# Fetch without checking out (stays on current branch)
gerry fetch 384465 --no-checkout

# Fetch a specific patchset
gerry fetch 384465 2

# Fetch and skip git hooks during checkout
gerry fetch 384465 --no-verify
```

**Using different filters:**
```bash
# See merged changes
gerry list --status merged

# See changes where you are CC'd
gerry list --cc

# Limit number of results
gerry list --limit 10

# Combine filters
gerry list --reviewer --status open --limit 5
```

## Updating

How you update gerry depends on how you installed it:

### If installed from source (git clone)

Use the built-in update command from anywhere:
```bash
gerry update
```

Or manually from the source directory:
```bash
cd gerrit-cli
git pull
make install
```

### If installed with go install

Reinstall with the latest version:
```bash
go install github.com/drakeaharper/gerrit-cli/cmd/gerry@latest
```

Or install a specific version:
```bash
go install github.com/drakeaharper/gerrit-cli/cmd/gerry@v0.2.0
```

### Check your current version

```bash
gerry version
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
- `--cc`: Show changes where you are CC'd

### `gerry analyze`
Analyze merged changes across all repositories or a specific repository within a date range.
- `--start-date`: Start date for analysis (YYYY-MM-DD)
- `--end-date`: End date for analysis (YYYY-MM-DD)
- `--repo`: Filter by specific repository
- `--format`: Output format (markdown, json, csv)
- `--output`: Save report to file

See [docs/analyze_command.md](docs/analyze_command.md) for detailed usage examples.

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
- `make install-man` - Install man page to appropriate location
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

## Troubleshooting

### Common Issues

**"Config file not found" error:**
```bash
# Run the setup wizard
gerry init
```

**"Authentication failed" error:**
```bash
# Check your HTTP password in Gerrit Settings → HTTP Password
# Regenerate if needed and run init again
gerry init
```

**"Not in a git repository" error:**
```bash
# Make sure you're in a git repository
git status

# Or clone the repository first
git clone <repo-url>
cd <repo-name>
```

**"Working directory is not clean" error (cherry-pick):**
```bash
# Commit or stash your changes first
git add .
git commit -m "Work in progress"

# Or stash them
git stash
```

**SSH connection issues:**
```bash
# Test SSH connection manually
ssh -p 29418 username@gerrit.server.com gerrit version

# Check SSH key permissions
chmod 600 ~/.ssh/id_rsa
```

**REST API timeout issues:**
```bash
# Check if the HTTP port is correct
# Common ports: 443 (HTTPS), 8080 (HTTP)
gerry init  # Reconfigure with correct port
```

### Getting Help

**Command-specific help:**
```bash
gerry <command> --help
```

**Verbose output for debugging:**
```bash
gerry --verbose list
```

**Check version:**
```bash
gerry version
```

**View manual page:**
```bash
man gerry  # If man page is installed
```

**Update to latest version:**
```bash
# From source directory
gerry update
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT
