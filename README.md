# LSP Inspector

Real-time web UI for viewing LSP client/server communication in a chat-style timeline.

![screenshot](https://github.com/user-attachments/assets/placeholder)

Supports **Neovim** `lsp.log` (live watching) and **VS Code** trace logs (static viewing).

## Install

### From release

Download the latest binary from [Releases](https://github.com/abonckus/lsp-inspector/releases) and add it to your PATH.

### From source

```sh
go install github.com/abonckus/lsp-inspector/cmd/lsp-inspector@latest
```

## Usage

```sh
# Neovim — live watch (auto-opens browser)
lsp-inspector ~/.local/state/nvim/lsp.log

# VS Code — static view
lsp-inspector ~/Downloads/trace.log

# Custom port, no auto-open
lsp-inspector --port 8080 --no-open lsp.log
```

## Features

- Chat-style timeline with send/receive bubbles
- Auto-detects Neovim and VS Code log formats
- Live file watching with WebSocket updates (Neovim)
- Collapsible JSON payloads with syntax highlighting
- Consecutive duplicate messages collapsed with count badge
- Filter by LSP server or method name
- Request/response correlation with elapsed time
- Clear log button (truncates file on disk)

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | random | HTTP port to serve on |
| `--no-open` | false | Don't auto-open browser |

## Supported log formats

### Neovim

The standard `lsp.log` written by Neovim's built-in LSP client. Typically found at:
- Linux/macOS: `~/.local/state/nvim/lsp.log`
- Windows: `%LOCALAPPDATA%\nvim-data\lsp.log`

Enable with `:lua vim.lsp.set_log_level("debug")` in Neovim.

### VS Code

Trace logs from VS Code's LSP output channel. To capture:
1. Open the Output panel (`Ctrl+Shift+U`)
2. Select your language server from the dropdown
3. Set trace level to `Verbose`
4. Save the output to a file

## License

MIT
