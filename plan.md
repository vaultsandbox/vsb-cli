# Refactoring Plan: Codebase Simplification & Cleanup

## 1. Consolidate Configuration Logic
**Issue:** Logic for determining the output format (Flag > Env > Config > Default) is split between `internal/cli/utils.go` and `internal/config/config.go`.
**Solution:**
- Update `internal/config/config.go`: Modify `GetDefaultOutput()` to check `VSB_OUTPUT` environment variable.
- Update `internal/cli/utils.go`: Simplify `getOutput()` to just check the flag, then delegate to `config.GetDefaultOutput()`.

## 2. Centralize Time Formatting
**Issue:** Date formatting logic is scattered. `internal/cli/utils.go` has helpers, but `internal/tui/watch/model.go` performs manual formatting strings.
**Solution:**
- Create `internal/output/time.go`.
- Move `formatDuration` and `formatRelativeTime` from `internal/cli/utils.go` to `internal/output/time.go`.
- Update `internal/cli/list.go` and `internal/tui/watch/model.go` to use these central helpers where appropriate, ensuring consistent display styles across CLI and TUI.

## 3. Better File Organization
**Issue:** `internal/cli/utils.go` is a "grab bag" of unrelated functions (output, files, time).
**Solution:**
- Move `sanitizeFilename` to `internal/files/utils.go` (creating the file if needed).
- The remaining functions in `internal/cli/utils.go` (`outputJSON`, `orDefault`) can remain or be moved to `internal/cli/helpers.go` to potentially eliminate `utils.go` entirely.

## 4. Simplify Root Command Logic
**Issue:** `internal/cli/root.go` contains complex logic for loading the keystore, identifying the active inbox, creating the client, and importing all inboxes. This makes the `runRoot` function long and hard to test.
**Solution:**
- Extract this logic into a new helper function `LoadAndImportAllInboxes` in `internal/cli/helpers.go`.
- This mirrors the existing `LoadAndImportInbox` helper (which handles a single inbox).

## Execution Steps
1.  **Refactor Config**: Modify `internal/config/config.go` and `internal/cli/utils.go`.
2.  **Move Utilities**: Create `internal/output/time.go`, move file utils to `internal/files`.
3.  **Refactor Root**: Create `LoadAndImportAllInboxes` and update `root.go`.
4.  **Cleanup**: Verify and remove unused code.
