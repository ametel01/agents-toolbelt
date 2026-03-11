# agents-toolbelt (`atb`)

**Install the CLI toolbelt that autonomous and semi-autonomous coding agents need to do real work.**

`atb` is a command-line installer and manager for the utilities AI coding agents depend on: code search tools, file discovery tools, JSON and YAML processors, API clients, text transformation tools, runtime managers, task runners, and infrastructure CLIs.

It is built on a simple assumption:

**coding agents work best when they are given broad, verified access to the tools they need, not a thin environment with missing binaries and constant operational friction.**

Instead of hand-assembling that environment across different package managers and machines, `atb` installs and manages it from one place.

---

## Installation

Install `atb` with a single `curl | bash` command. **Go does not need to be installed on the host machine**.

```bash
curl -fsSL https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh | bash
```

By default the installer:

* installs to `~/.local/bin` for a normal user
* installs to `/usr/local/bin` only when run as `root`

It does **not** invoke `sudo` automatically.

To install to a different directory on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh | ATB_INSTALL_DIR="$HOME/.local/bin" bash
```

For a system-wide install, inspect the script first and then run it with explicit privileges:

```bash
curl -fsSLO https://raw.githubusercontent.com/ametel01/agents-toolbelt/main/scripts/install.sh
less install.sh
sudo ATB_INSTALL_DIR=/usr/local/bin bash install.sh
```

If `~/.local/bin` is not already on your `PATH`, add it:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
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

Autonomous coding agents are only as capable as the environment they are dropped into.

If an agent cannot search quickly, inspect files clearly, query APIs, parse structured output, compare changes, or move through a repository efficiently, it slows down or fails outright. In practice, many agent sessions start in incomplete environments where the necessary CLI tools are missing, inconsistently installed, or spread across different package managers.

`atb` exists to solve that problem.

It gives you:

* a curated catalog of CLI tools useful to coding agents
* a fast installer for assembling that toolbelt on a fresh machine
* automatic platform and package-manager detection
* post-install verification so the environment is actually usable
* a machine-readable inventory of what is available

The goal is not minimalism. The goal is operational completeness.

---

## Built for coding agents

`atb` is designed for environments where agents such as **Claude Code** or **Codex** are expected to act with a high degree of autonomy.

The generated skills are specifically for **Claude Code** and **Codex**. They are not intended as a generic skill format for other agent runtimes.

The underlying assumption is straightforward:

* agents perform better when they have complete tool access
* verified local binaries are better than implicit assumptions
* fewer operational constraints means less wasted agent effort
* a well-equipped terminal is a prerequisite for strong agent performance

After installation, `atb` automatically generates a **`cli-tools` skill** in the format expected by **Claude Code** and **Codex**, listing the verified CLI tools available on the machine.

This gives the agent immediate visibility into:

* which binaries exist
* how tools are grouped by category
* what is actually available right now, not what someone assumes is installed

The generated skill is intentionally minimal. It is there to expose capability, not to bury the agent in tutorials.

---

## What `atb` does

`atb` installs and manages a curated set of CLI tools that expand what coding agents can do in a terminal session.

It provides:

### Interactive installation

Run one command to open a terminal UI where you can browse and select tools by category.

Tools are grouped and prioritized so you can assemble a useful default environment quickly:

* **Must have** tools are highlighted as the recommended baseline
* **Should have** tools are optional
* **Nice to have** tools are hidden until expanded

This lets you provision a strong baseline fast while keeping control over what gets installed.

---

### Automatic platform detection

`atb` detects your system and chooses the best installation method automatically.

Supported environments include:

* macOS
* Linux

Package managers are automatically discovered, including `brew`, `apt`, `dnf`, and `pacman` when available.

---

### Verification after install

Every installation is verified after it completes.

`atb` checks that the binary is actually available and working before recording it as usable. If an installation fails, the process continues and reports the failure at the end.

Existing tools already present on your system are detected automatically and reused.

---

### Inventory and machine state

`atb` keeps track of which tools are installed and which ones it manages.

You can inspect the current environment with:

```bash
atb status
```

This provides a clear overview of:

* installed tools
* binary locations
* whether they were installed by `atb` or already existed
* the last recorded verification state

---

## Typical workflow

A typical setup looks like this:

```bash
atb install
```

You will see an interactive interface where you can select tools to install.

After installation finishes, `atb`:

1. verifies each tool
2. records your tool inventory
3. generates the `cli-tools` skill for Claude Code and Codex
4. suggests optional shell integrations when needed

Once complete, the machine is ready to support agent-driven coding work with a broader and more reliable CLI surface area.

---

## Key commands

Install tools interactively:

```bash
atb install
```

Install recommended tools without prompts:

```bash
atb install -y
```

Check the status of installed tools:

```bash
atb status
```

View the available tool catalog:

```bash
atb catalog
```

Update tools installed by `atb`:

```bash
atb update
```

Uninstall tools managed by `atb`:

```bash
atb uninstall <tool>
```

---

## Operational usage

These are the core commands used after `atb` is installed.

### Browse the available catalog

```bash
atb catalog
```

Use this to inspect the embedded tool catalog, including tier, category, and current detected install status.

### Inspect the current machine state

```bash
atb status
```

This shows:

* whether each tool is installed
* whether it is managed by `atb` or external
* the detected binary path
* the last recorded verification result

### Install in interactive mode

```bash
atb install
```

This opens the terminal picker so users can:

* review the recommended `must` tools
* optionally add `should` tools
* expand and choose `nice` tools

### Install defaults without prompts

```bash
atb install -y
```

This runs in headless mode and installs the tools marked as default selections for the current platform.

### Update tools managed by `atb`

```bash
atb update
```

To update one managed tool only:

```bash
atb update rg
```

`atb update` does not update tools that are merely detected on `PATH` without an `atb` receipt.

### Uninstall tools managed by `atb`

```bash
atb uninstall rg
```

To remove all managed tools:

```bash
atb uninstall --all
```

`atb uninstall` refuses to remove tools that were not installed by `atb`.

### Shell integration behavior

Some tools such as `direnv` can add shell initialization lines.

`atb` can:

* suggest those shell hook lines
* record whether the user accepted or declined them
* apply confirmed changes idempotently

It does **not** modify shell rc files without explicit confirmation.

---

## Philosophy

`atb` is built around a clear operating model:

**if you want strong coding-agent performance, give the agent a complete and verified CLI environment.**

That means:

* more available tools
* fewer missing dependencies
* less time wasted on environment friction
* better odds that the agent can complete work end to end

`atb` does not replace those tools or abstract them away. It assembles and manages the toolbelt so agents can use the machine more effectively.

---

## Quick start

Run:

```bash
atb install
```

Choose the tools you want, let `atb` verify them, and the machine will be ready for agent-driven coding work in minutes.
