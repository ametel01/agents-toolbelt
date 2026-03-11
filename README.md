# agents-toolbelt (`atb`)

**One installer for the CLI tools modern developers actually use.**

`atb` is an interactive command-line tool that installs and manages a curated set of productivity-focused CLI utilities commonly used in coding workflows and AI-assisted development.

Instead of manually discovering, installing, and configuring dozens of tools across different package managers, `atb` provides a **single, organized installer** that sets them up in one place.

---

## Installation

Install `atb` with a single `curl | bash` command. **Go does not need to be installed on the host machine**.

```bash
curl -fsSL https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh | bash
```

By default the installer places `atb` in `/usr/local/bin`.

To install to a different directory on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh | ATB_INSTALL_DIR="$HOME/.local/bin" bash
```

To install a specific release tag instead of the latest:

```bash
curl -fsSL https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh | ATB_VERSION="v0.1.0" bash
```

Verify the installation:

```bash
atb --version
```

Manual release downloads are also available at:

`https://github.com/ametel01/agents-toolbelt/releases`

---

## Why `atb` exists

Modern development environments often depend on many small command-line tools:

* navigation and fuzzy search tools
* better file viewers and diff tools
* JSON and API utilities
* environment managers
* task runners and workflow helpers

Setting these up manually can be tedious and inconsistent across machines.

`atb` solves this by providing:

* a **curated catalog of useful CLI tools**
* a **fast interactive installer**
* **automatic platform detection**
* **safe installation and verification**
* a **single inventory of installed tools**

Once installed, your terminal environment is ready for productive coding and automation.

---

## What `atb` does

`atb` installs and manages a curated set of CLI tools designed to improve everyday terminal workflows.

It provides:

### Interactive installation

Run a single command to open a terminal UI where you can browse and select tools by category.

Tools are grouped and prioritized so you can install what you want quickly:

* **Must have** tools are preselected
* **Should have** tools are optional
* **Nice to have** tools are hidden until expanded

You remain in control of exactly what gets installed.

---

### Automatic platform detection

`atb` detects your system and chooses the best installation method automatically.

Supported environments include:

* macOS
* Linux

Package managers are automatically discovered (for example `brew`, `apt`, `dnf`, or `pacman`) and used when available.

---

### Safe installs and verification

Every tool installation is verified after it completes.

`atb` checks that the binary is available and working before adding it to your environment.
If an installation fails, the process continues and reports the failure at the end.

Existing tools already installed on your system are detected automatically and reused.

---

### A clean inventory of available tools

`atb` keeps track of which tools are installed and which ones it manages.

You can always see your environment with:

```
atb status
```

This provides a simple overview of:

* installed tools
* their location on your system
* whether they were installed by `atb` or already existed

---

### Built for coding agents

Many developers now work with coding agents such as **Claude Code** or **Codex**.

`atb` automatically generates a **`cli-tools` skill** that lists the verified CLI tools available on your machine.

This allows agents to understand which tools exist in your environment without needing configuration.

The generated skill contains:

* the list of available binaries
* grouped by category
* no tutorials or command documentation

This keeps the skill minimal while allowing agents to leverage your installed tools effectively.

---

## Typical workflow

A typical setup looks like this:

```
atb install
```

You will see an interactive interface where you can select tools to install.

After installation finishes, `atb`:

1. verifies each tool
2. records your tool inventory
3. generates the `cli-tools` skill
4. suggests optional shell integrations when needed

Once complete, your terminal environment is ready to use.

---

## Key commands

Install tools interactively:

```
atb install
```

Install recommended tools without prompts:

```
atb install -y
```

Check the status of installed tools:

```
atb status
```

View the available tool catalog:

```
atb catalog
```

Update tools installed by `atb`:

```
atb update
```

Uninstall tools managed by `atb`:

```
atb uninstall <tool>
```

---

## Operational usage

These are the core commands users will run after `atb` is installed.

### Browse the available catalog

```
atb catalog
```

Use this to inspect the embedded tool catalog, including tier, category, and current detected install status.

### Inspect the current machine state

```
atb status
```

This shows:

* whether each tool is installed
* whether it is managed by `atb` or external
* the detected binary path
* the last recorded verification result

### Install in interactive mode

```
atb install
```

This opens the terminal picker so users can:

* keep `must` tools preselected
* optionally add `should` tools
* expand and choose `nice` tools

### Install defaults without prompts

```
atb install -y
```

This runs in headless mode and installs the tools marked as default selections for the current platform.

### Update tools managed by `atb`

```
atb update
```

To update one managed tool only:

```
atb update fzf
```

`atb update` does not update tools that are merely detected on `PATH` without an `atb` receipt.

### Uninstall tools managed by `atb`

```
atb uninstall fzf
```

To remove all managed tools:

```
atb uninstall --all
```

`atb uninstall` refuses to remove tools that were not installed by `atb`.

### Shell integration behavior

Some tools such as `zoxide`, `direnv`, `atuin`, and `starship` can add shell initialization lines.

`atb` can:

* suggest those shell hook lines
* record whether the user accepted or declined them
* apply confirmed changes idempotently

It does **not** modify shell rc files without explicit confirmation.

---

## Philosophy

`atb` focuses on a simple idea:

**make powerful CLI tools easy to discover, install, and use.**

It does not replace those tools or change how they work.
It simply gives you a clean way to install and manage them from one place.

The result is a faster, more consistent development environment across machines.

---

## Quick start

Run:

```
atb install
```

Choose the tools you want, and your CLI environment will be ready in minutes.
