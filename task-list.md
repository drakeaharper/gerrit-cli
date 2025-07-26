# Gerrit CLI (gerry) Implementation Task List

## Overview
Building a Go-based CLI tool for Gerrit Code Review with the command structure `gerry <command>`.

## Task Categories

### 1. Project Setup
- [x] Create task-list.md for tracking progress
- [x] Initialize Go module (`go mod init github.com/drakeaharper/gerrit-cli`)
- [x] Create directory structure
- [x] Set up basic Makefile for building
- [x] Create .gitignore file

### 2. Core Infrastructure
- [x] Implement configuration management (config.go)
- [x] Create Gerrit SSH client wrapper
- [x] Create Gerrit REST API client wrapper
- [x] Implement error handling utilities
- [x] Add logging framework

### 3. Commands Implementation
- [x] `gerry init` - Interactive setup wizard
  - [x] Prompt for server details
  - [x] Test SSH connectivity
  - [x] Test REST API authentication
  - [x] Save configuration
- [ ] `gerry list` - List changes
  - [ ] List user's open changes
  - [ ] Add --detailed flag
  - [ ] Add --reviewer flag
- [ ] `gerry comments <change-id>` - View comments
  - [ ] Fetch all comments
  - [ ] Filter unresolved comments
  - [ ] Format output nicely
- [ ] `gerry details <change-id>` - Show change details
  - [ ] Fetch change metadata
  - [ ] Show files changed
  - [ ] Show review scores
- [ ] `gerry fetch <change-id>` - Fetch changes
  - [ ] Calculate refs path
  - [ ] Execute git fetch
  - [ ] Checkout FETCH_HEAD
- [ ] `gerry cherry-pick <change-id>` - Cherry-pick changes
  - [ ] Fetch change
  - [ ] Execute cherry-pick

### 4. Documentation
- [x] Update README.md with installation guide
- [ ] Add usage examples
- [ ] Document configuration options
- [ ] Create man page

### 5. Testing
- [ ] Unit tests for config management
- [ ] Unit tests for API clients
- [ ] Integration tests for commands
- [ ] Mock Gerrit server for testing

### 6. Build & Release
- [ ] Set up GitHub Actions for CI
- [ ] Configure goreleaser
- [ ] Create release binaries for multiple platforms
- [ ] Add homebrew formula

## Implementation Notes

### Dependencies
- cobra - CLI framework
- viper - Configuration management
- survey - Interactive prompts
- color - Colored output
- golang.org/x/crypto/ssh - SSH client

### Configuration File Location
`~/.gerry/config.json`

### Environment Variables
- GERRIT_SERVER
- GERRIT_PORT
- GERRIT_USER
- GERRIT_HTTP_PASSWORD
- GERRIT_PROJECT

## Progress Log
- 2025-07-26: Created task-list.md for tracking implementation progress
- 2025-07-26: Initialized Go module and created basic project structure
- 2025-07-26: Created .gitignore for Go project
- 2025-07-26: Updated README.md with comprehensive installation guide
- 2025-07-26: Completed core infrastructure implementation:
  - Created Makefile with build, test, and cross-compilation targets
  - Implemented configuration management with JSON storage and environment variable support
  - Built SSH client wrapper for Gerrit SSH API interactions
  - Built REST client wrapper for Gerrit REST API interactions
  - Implemented error handling utilities with custom error types
  - Added logging framework with configurable log levels
- 2025-07-26: Implemented basic CLI structure using Cobra:
  - Created main.go entry point with version tracking
  - Set up root command with global flags
  - Implemented `gerry init` command with interactive setup wizard
  - Added placeholder commands for list, comments, details, fetch, and cherry-pick
- 2025-07-26: Successfully built and tested the project compilation