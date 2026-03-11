English | [中文](README_zh.md)

# SnipFly CLI

<img src="docs/appicon.png" alt="SnipFly Icon" width="96">

A TUI tool for running multiple code snippets, in a single terminal tab.

<!-- screenshot -->
<!-- <img src="docs/screenshot.png" alt="SnipFly screenshot" width="800"> -->

## Quick Start

Install SnipFly first:

```bash
brew install txperl/tap/snipfly
# or
go install github.com/txperl/snipfly@latest
```

Then, create a `.snipfly/` directory in your project and add a snippet:

```bash
mkdir .snipfly
echo 'echo Hello from SnipFly!' > .snipfly/hello.sh
```

Let's snipfly!

```bash
snipfly
```

##### The `.snipfly/` subfolder

SnipFly will automatically detect snippets from the subfolder (`.snipfly/`) in the working directory.

If there's no that folder, SnipFly will look for snippets of working directory directly.

## Usage

```bash
# snipfly -h
Usage: snipfly [options] [directory]

Options:
  -e, --exact     use exact directory without auto-detecting ./.snipfly/
  -g, --global    scan global snippets directory (~)
  -v, --version   print version information

# Run snippets in the current directory
snipfly
snipfly .

# Run snippets in a specific directory
snipfly /path/to/project

# Run global snippets
snipfly -g
```

### Keybindings

| Key     | Action                 |
| ------- | ---------------------- |
| `↑`     | Move up                |
| `↓`     | Move down              |
| `Space` | Run/stop snippet       |
| `r`     | Restart snippet        |
| `Tab`   | Switch to output panel |
| `q`     | Quit                   |

## Snippets

### Structure

```
project/
└── .snipfly/
    ├── deploy.sh            # Root level (no group)
    ├── dev/                 # Group: "dev"
    │   ├── server.sh
    │   └── watch.ts
    └── tools/               # Group: "tools"
        ├── lint.sh
        └── format.py
```

### Annotations

Add annotations as comments at the top of your snippet files. The parser reads lines starting with `#` or `//` until the first non-comment, non-empty line.

| Annotation     | Type    | Default             | Description                                   |
| -------------- | ------- | ------------------- | --------------------------------------------- |
| `@name`        | string  | filename (no ext)   | Display name in the list                      |
| `@desc`        | string  | --                  | Short description                             |
| `@type`        | string  | `oneshot`           | `oneshot`, `service`, or `interactive`        |
| `@dir`         | string  | snippet's directory | Working directory (`~` supported)             |
| `@env`         | string  | --                  | Environment variable `KEY=VALUE` (repeatable) |
| `@interpreter` | string  | auto-detect         | Override interpreter command                  |
| `@pty`         | boolean | `false`             | Allocate a pseudo-terminal                    |

Example:

```bash
# @name: Clock Server
# @desc: Print current time every second
# @type: service
# @dir: /tmp
# @env: TZ=UTC
# @env: LANG=en_US.UTF-8
# @pty: true

while true; do date; sleep 1; done
```

Minimum example:

```bash
echo "Hello from SnipFly!"
```

## Build from Source

```
git clone https://github.com/txperl/snipfly.git
cd snipfly
go build -o snipfly .
```

## License

- [MIT](LICENSE)
