# LazyRG - Interactive Ripgrep TUI

A terminal user interface for ripgrep, making it easier to search through your codebase and view results interactively.

## Requirements

### Essential
- Go 1.19 or later
- [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg` command)

### Recommended
- [bat](https://github.com/sharkdp/bat) - For syntax highlighting in file preview (falls back to basic display if not available)
- A terminal that supports:
  - True color (24-bit color)
  - Unicode characters
  - Modern terminal features (most terminals like iTerm2, Alacritty, kitty, or modern versions of gnome-terminal work well)

### Optional
- A [Nerd Font](https://www.nerdfonts.com/) for optimal icon display (though regular emoji fonts work too)

## Installation

```bash
go install github.com/yourusername/lazyrg@latest
```

## Usage

Simply run:
```bash
lazyrg
```

### Key Bindings
- `ctrl+f` or `ctrl+s`: Focus search
- `enter`: Execute search/select result
- `ctrl+t`: Switch tabs
- `tab`: Navigate between inputs
- `esc`: Go back
- `?`: Toggle help
- `ctrl+c` or `q`: Quit

## Building from Source

```bash
git clone https://github.com/yourusername/lazyrg.git
cd lazyrg
go build
```

