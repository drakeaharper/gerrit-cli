# Refactor Task List

Quality improvements identified from broad codebase review (2026-04-13).

## Status Legend
- [ ] Not started
- [~] In progress
- [x] Done

---

## High Priority

### 1. Deduplicate list.go / team.go
- [x] Extract shared helpers (`getStringValue`, `getOwnerName`, SSH JSON parsing) into `internal/cmd/change_helpers.go`
- [x] Extract shared display logic (`displayDetailedChanges`)
- [x] Delete all `getTeam*` duplicates from `team.go`
- [x] Verify both commands still build

### 2. Consolidate label extraction functions
- [x] Replace 8 label functions with single `getLabelStatus(change, labelName)`
- [x] Handle SSH fallback path inside the same function
- [x] Fixed QR bug (team.go passed "QR" instead of "QA-Review" to FormatScore)
- [x] Net ~450 lines removed (combined with #1)

### 3. Introduce typed Gerrit response structs
- [x] Define `Change`, `Account`, `RevisionInfo`, `FileInfo`, `CommentInfo`, `ChangeMessageInfo`, `SSHPatchSet` structs in `internal/gerrit/types.go`
- [x] Update `RESTClient` methods to return typed structs
- [x] Helper methods: `ChangeNumber()`, `UpdatedTime()`, `CurrentPatchSetNumber()`, `DisplayName()`
- [x] Migrate all command code to use struct fields
- [x] SSH responses unmarshal into same Change struct via dual json tags
- [x] Labels kept as `map[string]interface{}` for Gerrit version compatibility

---

## Medium Priority

### 4. Add unit tests
- [x] `internal/utils/` — validation, formatting, time formatting, table output
- [x] `internal/gerrit/` — types, Account, Change methods, JSON round-trips
- [ ] `internal/config/` — Load, Save, Validate, GetRESTURL (needs filesystem mocking)
- [ ] `internal/cmd/` — comment threading, label extraction (needs more setup)

### 5. Remove os.Exit from business logic
- [x] Change all command `Run` functions to `RunE`
- [x] Return errors instead of calling `utils.ExitWithError`
- [x] Convert helpers: `loadConfigAndClient`, `selectThread`, `promptMessage`, `listWorktrees`, `runResolveAction`
- [x] Replaced `os.Exit(0)` in cherrypick conflict path with `return nil`
- [x] Replaced `os.Exit(1)` in init SSH test with returned error

### 6. Fix CSV generation
- [x] Replaced manual string escaping with `encoding/csv.Writer`

---

## Low Priority

### 7. Move stripANSI regex to package-level var
- [x] Moved to package-level `var ansiRegex`

### 8. Migrate off deprecated ExecuteCommand
- [x] All callers migrated to `ExecuteCommandArgs`
- [x] `ExecuteCommand` and `StreamCommand` removed

### 9. Fix TruncateString for multi-byte characters
- [x] Switched to `[]rune` conversion

### 10. Fix FormatTimeAgo timezone handling
- [x] `ParseInLocation` with explicit UTC
- [x] `time.Now().UTC()` in `timeAgo`

### 11. Clean up REST client URL construction
- [x] Removed `getBaseURL()` method, `doRequest` calls `GetRESTURL` directly

---

## Out of Scope (noted for future)
- CI/CD pipeline (GitHub Actions)
- Keychain/keyring integration for HTTP password
- Package-level flag variables (Cobra convention, low ROI to change)
