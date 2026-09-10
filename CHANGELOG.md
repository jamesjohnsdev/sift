# Changelog

## [0.1.4](https://github.com/jamesjohnsdev/sift/compare/v0.1.3...v0.1.4) (2026-09-10)


### Features

* add live Markdown preview to compose ([271dfe0](https://github.com/jamesjohnsdev/sift/commit/271dfe015340fff1bf8bac208f05bb7355b47e40))
* add theme-aware Markdown/HTML content renderer ([a9b5b46](https://github.com/jamesjohnsdev/sift/commit/a9b5b464ffada09c36e885d357384849301b5b00))
* render message preview through the content renderer ([5da26a6](https://github.com/jamesjohnsdev/sift/commit/5da26a6852b7b3e0ca1536ad50fdf8f44aab2c62))

## [0.1.3](https://github.com/jamesjohnsdev/sift/compare/v0.1.2...v0.1.3) (2026-09-10)


### Features

* add 'sift account add' CLI command ([63c2f58](https://github.com/jamesjohnsdev/sift/commit/63c2f58af740aab27d234f79d6791ab9a2e2030f))
* add account config to Lua loader ([5e04fa9](https://github.com/jamesjohnsdev/sift/commit/5e04fa9840177d1a977fbb8b3894fa67bb9653ae))
* add compose/reply keymap actions ([b07ceda](https://github.com/jamesjohnsdev/sift/commit/b07cedadcba58d951fc5a118d5237848f4c96cfa))
* add compose/reply screen to the TUI ([3402057](https://github.com/jamesjohnsdev/sift/commit/3402057e570f357ccb0becefb507b9d95812b737))
* add Kong CLI with man page generation ([7134ef3](https://github.com/jamesjohnsdev/sift/commit/7134ef3c16c19a2ddf75918828a8f9fe4db93c59))
* add Manager.Send for composing new mail ([9288ec2](https://github.com/jamesjohnsdev/sift/commit/9288ec2421005770dfc1ffc16a0da85538af32c0))
* add sync engine ([0e338bf](https://github.com/jamesjohnsdev/sift/commit/0e338bfb0d9c9eec739f4141960416d7f648aa3e))
* add sync engine update notification channel ([fac7f05](https://github.com/jamesjohnsdev/sift/commit/fac7f05b5d1d1596ee099d7bc97504bfcab66275))
* wire tui and main to storage.Store and the sync engine ([baf5877](https://github.com/jamesjohnsdev/sift/commit/baf5877818b2f60bd3335a92ea6fe390f6989c5b))
* wire tui to reload on sync engine updates ([b1a1de2](https://github.com/jamesjohnsdev/sift/commit/b1a1de2e5661345adb0850e2726c5b4d1b68ef62))


### Bug Fixes

* include body in Outlook's message list select ([003b97b](https://github.com/jamesjohnsdev/sift/commit/003b97b0df11be746a37051e35b903a77b8425a9))
* reject wrong-type account fields instead of silently emptying them ([fc22d57](https://github.com/jamesjohnsdev/sift/commit/fc22d57221e6b91e862ad838a052e02b6883adb2))
* use per-provider loopback redirect host, drop path ([5230b24](https://github.com/jamesjohnsdev/sift/commit/5230b24805038914f513821da6ba648a93f1c054))

## [0.1.2](https://github.com/jamesjohnsdev/sift/compare/v0.1.1...v0.1.2) (2026-09-10)


### Features

* add shared markdown-to-HTML render for the send path ([3e78b9c](https://github.com/jamesjohnsdev/sift/commit/3e78b9cff3cf107451a81db274a5355aeb729d98))
* add shared OAuth HTTP client and poll-based Watch helper ([5ac20fd](https://github.com/jamesjohnsdev/sift/commit/5ac20fdd631f7ed69cd7629af490d73e9443e101))
* implement Gmail provider (read path) ([13841a3](https://github.com/jamesjohnsdev/sift/commit/13841a39e617a76a89a2047135fc40c6b16f9c30))
* implement Gmail Send and Attachment ([8d64c76](https://github.com/jamesjohnsdev/sift/commit/8d64c76a67b0f43d95e03254c934ed101ada4619))
* implement Outlook provider (read path) ([616937b](https://github.com/jamesjohnsdev/sift/commit/616937be6f7768d537ad664d6789742c7fdf4afe))
* implement Outlook Send and Attachment ([512dd7e](https://github.com/jamesjohnsdev/sift/commit/512dd7e1fa25ce9a1df1b45d8d425fa178f38435))

## [0.1.1](https://github.com/jamesjohnsdev/sift/compare/v0.1.0...v0.1.1) (2026-09-09)


### Features

* add bubbletea TUI shell ([88b7de0](https://github.com/jamesjohnsdev/sift/commit/88b7de039a4f9db0ee218ee8af417b6ae48930a8))
* add Lua config package ([9095710](https://github.com/jamesjohnsdev/sift/commit/90957101c7a2bcb3dc4f76a3ddc791365d1c29a9))
* add provider facade interface ([7a95c7b](https://github.com/jamesjohnsdev/sift/commit/7a95c7b8d92838def2f588978c730129b572528d))
* add SQLite storage layer ([0df6828](https://github.com/jamesjohnsdev/sift/commit/0df6828ae70754b62df591c59287957a20d6704e))


### Bug Fixes

* check errcheck-flagged Close/Rollback errors via errors.Join ([bb9bf77](https://github.com/jamesjohnsdev/sift/commit/bb9bf77e829d41916fde525cba7abc387ca4cbd7))
* correct pane height/width for lipgloss v2's total-size Style() ([d371825](https://github.com/jamesjohnsdev/sift/commit/d3718255b3dfd5511fecab0a621f667b9647c887))
* scope binary ignore rule to repo root ([87ebde0](https://github.com/jamesjohnsdev/sift/commit/87ebde0424bbd3f650d24530282088b40fab91e3))
