# Refactoring Plan: TUI Package

## 1. Rename `watch` folder to `emails`

The folder name "watch" is misleading - it's a full email viewer TUI, not just a watcher.

```
internal/tui/watch/  →  internal/tui/emails/
```

Update import in `internal/cli/root.go`.

---

## 2. Split model.go into focused files

Current `model.go` is ~765 lines doing too much. Split into:

| File | Contents |
|------|----------|
| `model.go` | Types (Model, EmailItem, messages), NewModel, Init |
| `update.go` | Update method and helpers |
| `view.go` | View, viewList, viewDetail, render* methods |
| `commands.go` | Tea commands (openFirstURL, viewHTML, deleteEmail, etc.) |
| `keys.go` | KeyMap type and DefaultKeyMap |

---

## 3. Add helpers to reduce duplication

In `model.go` or `helpers.go`:

```go
const noSubject = "(no subject)"

func (m Model) selectedEmail() *vaultsandbox.Email {
    if m.viewing && m.viewedEmail != nil {
        return m.viewedEmail.Email
    }
    filtered := m.filteredEmails()
    if i := m.list.Index(); i >= 0 && i < len(filtered) {
        return filtered[i].Email
    }
    return nil
}

func wrapIndex(current, delta, length int) int {
    if length == 0 {
        return 0
    }
    return (current + delta + length) % length
}
```

---

## Order of changes

1. Rename folder `watch` → `emails`
2. Update import in `root.go`
3. Split into separate files
4. Add helpers and update callers
5. Run `go build` to verify
