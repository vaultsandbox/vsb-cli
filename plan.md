# Bug Fix Plan

Based on findings.md analysis. Permissions are correctly set to `0600` - no issue there.

---

## 1. HIGH: Keystore directory not created on first write

**File:** `internal/config/keystore.go`

**Problem:** `saveLocked()` (lines 252-257) doesn't call `EnsureDir()`, but `AddInbox()` uses `saveLocked()`. First-time users get "no such file or directory".

**Fix:** Add `EnsureDir()` call at the start of `saveLocked()`:
```go
func (ks *Keystore) saveLocked() error {
    if err := EnsureDir(); err != nil {
        return err
    }
    // ... rest unchanged
}
```

---

## 2. MEDIUM: Config panic on short API key

**File:** `internal/cli/config.go:54-56`

**Problem:** Slicing `existing.APIKey[:7]` and `[len-4:]` panics if key < 11 chars.

**Fix:** Add length guard before masking:
```go
if existing.APIKey != "" {
    if len(existing.APIKey) >= 11 {
        masked := existing.APIKey[:7] + "..." + existing.APIKey[len(existing.APIKey)-4:]
        prompt = fmt.Sprintf("API Key [%s]: ", masked)
    } else {
        prompt = "API Key [****]: "
    }
}
```

---

## 3. MEDIUM: TTL flag doesn't support days

**File:** `internal/cli/inbox_create.go:38-55`

**Problem:** Help text shows `7d` but `time.ParseDuration` only supports up to hours.

**Fix:** Parse days manually before calling `ParseDuration`:
```go
func parseTTL(s string) (time.Duration, error) {
    // Handle days suffix
    if strings.HasSuffix(s, "d") {
        days := strings.TrimSuffix(s, "d")
        n, err := strconv.Atoi(days)
        if err != nil {
            return 0, fmt.Errorf("invalid day value: %s", days)
        }
        return time.Duration(n) * 24 * time.Hour, nil
    }
    return time.ParseDuration(s)
}
```

---

## 4. MEDIUM: Watch goroutines spin on closed channels

**File:** `internal/tui/watch/model.go:212-249`

**Problem:** Channel reads don't check for close. If SDK closes channel before context cancellation, the loop spins with nil values.

**Fix:** Use `for range` or check `ok` value:
```go
// Option A: for-range (cleanest)
for event := range eventCh {
    p.Send(emailReceivedMsg{...})
}

// Option B: check ok
case event, ok := <-eventCh:
    if !ok {
        return
    }
    if event != nil {
        p.Send(...)
    }
```

Apply to both goroutines (lines 217-231 and 234-249).

---

## 5. LOW: Silent errors in main

**File:** `cmd/vsb/main.go:9-12`

**Problem:** If `cli.Execute()` returns an error, user sees nothing - just non-zero exit.

**Fix:** Print error to stderr:
```go
func main() {
    if err := cli.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

---

## 6. LOW: Binary committed to repo

**Problem:** `vsb` binary is tracked despite being in `.gitignore` (was committed before the rule).

**Fix:**
```bash
git rm --cached vsb
git commit -m "Remove accidentally committed binary"
```

---

## Execution Order

1. Fix #1 (keystore) - blocks first-time usage
2. Fix #2 (config panic) - crash prevention
3. Fix #3 (TTL parsing) - user-facing bug
4. Fix #4 (channel spin) - resource leak
5. Fix #5 (silent errors) - UX improvement
6. Fix #6 (binary) - repo hygiene
