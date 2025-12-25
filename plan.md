# Refactoring Plan

Practical cleanup to reduce repetition before adding new features.

## High Priority (Quick Wins)

### 1. Remove duplicate Config struct
**Files:** `internal/cli/config.go`, `internal/config/config.go`

The CLI package redefines a `Config` struct instead of importing from config package. Consolidate to single definition.

- [ ] Update `internal/cli/config.go` to import `config.Config`
- [ ] Remove local struct definition

### 2. Create shared utilities file
**New file:** `internal/cli/utils.go`

Move scattered helper functions to a single location:

- [ ] Move `orDefault()` from `import.go:188-193`
- [ ] Move `sanitizeFilename()` from `export.go:143-155`
- [ ] Move `formatDuration()` from `inbox_list.go:99-107`
- [ ] Update imports in source files

### 3. Add shared box styles
**File:** `internal/styles/styles.go`

Add reusable styles for success/warning/error boxes used in export, import, inbox_create.

- [ ] Add `WarningBoxStyle`
- [ ] Add `SuccessBoxStyle`
- [ ] Add box title styles
- [ ] Update `export.go`, `import.go`, `inbox_create.go` to use shared styles

---

## Medium Priority

### 4. Simplify GetEmailByIDOrLatest
**File:** `internal/cli/helpers.go:40-94`

Split 55-line function into smaller, reusable parts:

- [ ] Extract `LoadAndImportInbox()` function
- [ ] Simplify `GetEmailByIDOrLatest()` to use new helper
- [ ] Ensure consistent cleanup pattern

### 5. Standardize client creation
**File:** `internal/config/client.go` (new or existing)

Add helper to ensure consistent client lifecycle:

- [ ] Create `WithClient(ctx, fn)` wrapper function
- [ ] Update commands to use wrapper where appropriate

---

## Low Priority (Optional)

### 6. Remove dead code
- [ ] Delete unused `Load()` function in `config/config.go:63-70` (verify it's unused first)

### 7. Error message constants
Not worth the abstraction - skip unless we add more error handling later.

---

## Order of Execution

1. #1 (Config struct) - isolated change, low risk
2. #2 (utils.go) - simple moves, no behavior change
3. #3 (styles) - visual consistency
4. #4 (helpers refactor) - more involved, test after
5. #5 (client wrapper) - optional, depends on comfort level
6. #6 (dead code) - cleanup last
