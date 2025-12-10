# todototum

[![Release](https://img.shields.io/github/v/release/valerioTomassi/todototum)](https://github.com/valerioTomassi/todototum/releases)
[![Build](https://github.com/valerioTomassi/todototum/actions/workflows/ci.yml/badge.svg)](https://github.com/valerioTomassi/todototum/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/valerioTomassi/todototum)](https://github.com/valerioTomassi/todototum/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/valerioTomassi/todototum)](https://goreportcard.com/report/github.com/valerioTomassi/todototum)
[![License](https://img.shields.io/github/license/valerioTomassi/todototum)](./LICENSE)

See the whole TODO picture. `todototum` scans your codebase for `TODO`, `FIXME`, `BUG`, and `NOTE` comments across any language, then prints a clear table to the terminal or generates HTML, JSON, and Markdown reports.

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
  - [Go Install](#go-install)
  - [Binaries](#binaries)
  - [Homebrew](#homebrew)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Development](#development)
- [Release & Publish](#release--publish)
- [License](#license)

---

## Features

- Fast CLI powered by Cobra
- Sensible ignores (respects `.gitignore` plus extra patterns)
- Multiple outputs: table (TTY), HTML, JSON, Markdown
- Optionally open the HTML report in your browser

## Requirements

- Go 1.25+ (see [`go.mod`](./go.mod))

## Installation

### Go Install

```bash
go install github.com/valerioTomassi/todototum@latest
```

### Binaries

Download from the Releases page and add to your `PATH`:

- https://github.com/valerioTomassi/todototum/releases

### Homebrew

Install via the reusable tap:

```bash
brew tap valerioTomassi/taps
brew install todototum
```

Or in one line:

```bash
brew install valerioTomassi/taps/todototum
```

## Quick Start

Scan the current folder:

```bash
todototum scan
```

Open the HTML report in your browser:

```bash
todototum scan --serve
```

Choose an output format and directory:

```bash
todototum scan --report html|json|md --out-dir reports
```

Ignore common folders:

```bash
todototum scan --ignore vendor,.git,node_modules
```

## Usage

- See all flags: `todototum --help` or `todototum scan --help`
- Version info: `todototum version`

## Development

If you use `go-task`:

```bash
task fmt   # format
task lint  # lint
task test  # test
task run   # sample run
```

## Release & Publish

Releases and the Homebrew formula are automated via GitHub Actions and GoReleaser.

Cut a new release:
1. Update code as needed and merge to `main`.
2. Tag a version and push the tag (semantic versioning):

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. The Release workflow will:
   - Build binaries for darwin/linux (amd64, arm64)
   - Create a GitHub Release with archives and checksums
   - Generate/update the Homebrew formula in `valerioTomassi/homebrew-taps`

Verify after release:
- `brew update`
- `brew install valerioTomassi/taps/todototum`
- `todototum version`
