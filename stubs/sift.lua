---@meta
-- LuaLS type stubs for sift's init.lua. These declare the shape Go's
-- internal/config package actually parses; kept in sync by hand for now
-- (see internal/config/lua.go's *Fields maps and internal/config/accounts.go,
-- theme.go, keymap.go for the source of truth).
--
-- Not loaded at runtime - the `sift` global is injected by the sift binary
-- itself before init.lua runs. This file exists purely so
-- lua-language-server knows its shape: point it at this directory, e.g. in
-- a .luarc.json next to your init.lua:
--   { "workspace.library": ["/path/to/sift/stubs"] }
--
-- Then just call it - LuaLS annotates the call site directly, no
-- annotation needed in your own init.lua:
--   sift.setup({
--     theme = "dracula",
--     accounts = {
--       { id = "work", kind = "gmail", email = "me@example.com", client_id = "..." },
--     },
--   })

---@alias SiftThemeName
---| "tokyo-night" # default
---| "catppuccin"
---| "dracula"

---@class SiftTheme
---@field bg? string
---@field fg? string
---@field muted? string
---@field border? string
---@field accent? string
---@field select_bg? string
---@field status_bg? string

---@class SiftKeymap
---@field up? string[]
---@field down? string[]
---@field top? string[]
---@field bottom? string[]
---@field focus_left? string[]
---@field focus_right? string[]
---@field quit? string[]
---@field compose? string[]
---@field reply? string[]

---@class SiftAccount
---@field id string
---@field kind? "gmail"|"outlook"
---@field email? string
---@field client_id? string

---@class SiftConfig
---@field theme? SiftThemeName|SiftTheme
---@field keymap? SiftKeymap
---@field accounts? SiftAccount[]

sift = {}

---@param opts SiftConfig
function sift.setup(opts) end
