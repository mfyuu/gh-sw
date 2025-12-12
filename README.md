A lightweight GitHub CLI extension that provides an interactive local branch selector for seamless switching.

## Overview

`gh-sw` = `git branch` + `git switch`

Streamlines the process of switching between local branches. It displays all local branches in an interactive selection UI, allowing you to quickly switch to any branch.

Built with [golang/go](https://github.com/golang/go), this extension uses [charmbracelet/huh](https://github.com/charmbracelet/huh) for interactive selection.

## Motivation

When working with multiple branches, developers often need to:

1. Run `git branch` to see local branches
2. Identify the branch they want to switch to
3. Run `git switch <branch>` to switch to that branch

This extension combines these steps into a single command with an interactive interface, reducing context switching and making branch management more efficient. The arrow-key navigation makes it easy to quickly jump between different branches during development.

## Installation

### Prerequisites

- [GitHub CLI](https://cli.github.com/) must be installed and authenticated

### Install as a GitHub CLI extension

```bash
gh extensions install mfyuu/gh-sw
```

## Usage

```
gh sw --help

# Output:
Interactively switch to a local branch.

USAGE
  gh sw [branch]
  gh sw [flags]

FLAGS
  --help    Show help for command

EXAMPLES
  $ gh sw              # Interactive branch selection
  $ gh sw feature/auth # Switch to specific branch
  $ gh sw -            # Switch to previous branch
```

### Modes

- **Interactive (`gh sw`)**: Display all local branches and select one to switch to
- **Direct (`gh sw <branch>`)**: Switch directly to the specified branch
- **Previous (`gh sw -`)**: Switch to the previously checked out branch
