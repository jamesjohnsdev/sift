# sift

A terminal email client for Gmail and Outlook, built with Go and Bubble Tea.
Multi-account, themable, vim-style navigation, configured via Lua.

Status: early development. No account setup wizard yet, so adding a real
account currently requires manual steps (below).
design.

## Requirements

- Go 1.27 or later

## Build and run

```sh
go build -o sift .
./sift
```

or just `go run .`

On first run sift creates `~/.config/sift/` (or your OS's config dir) and
opens `sift.db` there for its local mail cache. With no accounts configured
it starts empty.

## Configuration

Config lives at `~/.config/sift/init.lua`. It's optional; sift runs on
defaults if the file doesn't exist. Example:

```lua
sift.setup({
  theme = "catppuccin", -- "tokyo-night" (default), "catppuccin", or "dracula"
  keymap = {
    quit = { "q", "ctrl+c" },
  },
  accounts = {
    { id = "work", kind = "gmail", email = "me@example.com", client_id = "..." },
  },
})
```

`init.lua` must call `sift.setup({ ... })` (the `sift` global is injected
by the binary before your file runs) - it's not read as a plain returned
table, so you can put arbitrary Lua (conditionals, `require`s of other
files) around the call.

A bad config file fails loudly on stderr and falls back to defaults rather
than refusing to start. Error messages are shown which can be traced.

### Editor autocomplete and type-checking

`stubs/sift.lua` has LuaLS (`---@class`/`---@field`) annotations for
`sift.setup`'s options table. Point lua-language-server at it (e.g. a
`.luarc.json` next to `init.lua` with
`{"workspace.library": ["/path/to/sift/stubs"]}`) and your editor will
autocomplete fields and flag typos/wrong types at the `sift.setup(` call
site - no annotation needed in your own `init.lua`.

## Adding an account (manual, for now)

There's no in-app wizard yet. To connect a real account:

1. Register an OAuth app with Google or Microsoft and get a client ID.
2. Add the account to `accounts` in the `sift.setup({ ... })` call in
   `init.lua` (see above).
3. Get an OAuth token into your OS keychain under the service `sift`,
   entry name matching the account's `id`. `internal/auth.Authenticate`
   runs the OAuth flow. There's no CLI wired up to call it yet, so this
   currently means writing a small throwaway Go program that imports
   `internal/auth` and `internal/config` (see `internal/auth/*_test.go`
   for how the pieces fit together).

Once a token exists, sift picks the account up on next launch and syncs it.

## Running tests

```sh
go build ./...
go vet ./...
go test ./...
```

## Layout

- `internal/provider` - Gmail and Outlook clients behind a shared interface
- `internal/storage` - local SQLite cache (accounts, tags, messages, outbox)
- `internal/config` - Lua config and theme loading
- `internal/auth` - OAuth2 flow and OS keychain token storage
- `internal/syncengine` - pulls mail into storage and watches for updates
- `internal/tui` - the Bubble Tea UI
