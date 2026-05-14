# timerd

> A PM2-style manager for systemd timers and services. Define your scheduled jobs in a simple YAML file — **timerd** handles the rest.

[![CI](https://github.com/Xwudao/go-timer/actions/workflows/ci.yml/badge.svg)](https://github.com/Xwudao/go-timer/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/go-1.24%2B-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Why timerd?

### Why not cron?

| Issue                         | cron         | timerd                       |
| ----------------------------- | ------------ | ---------------------------- |
| Missed jobs on sleep/shutdown | ❌ lost      | ✔ persistent timers catch up |
| Logs                          | grep syslog  | `timerd logs myjob`          |
| Enable/disable                | edit crontab | `timerd enable myjob`        |
| Env vars                      | awkward      | first-class in YAML          |
| Status                        | none         | `timerd status`              |

### Why not write systemd unit files directly?

You'd need to write two files per job (`.service` + `.timer`), remember the syntax, run `daemon-reload` manually, and keep them in sync. timerd does all of that for you.

### Comparison

| Feature             | timerd | PM2 | supercronic | crontab |
| ------------------- | ------ | --- | ----------- | ------- |
| systemd-native      | ✔      | ❌  | ❌          | partial |
| Persistent catch-up | ✔      | ❌  | ✔           | ❌      |
| YAML config         | ✔      | ✔   | ❌          | ❌      |
| Cron syntax         | ✔      | ✔   | ✔           | ✔       |
| Journald logs       | ✔      | ❌  | ❌          | ❌      |
| Root & user mode    | ✔      | ❌  | ❌          | ✔       |
| No daemon process   | ✔      | ❌  | ❌          | ✔       |

---

## Installation

### Quick install (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Xwudao/go-timer/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/Xwudao/go-timer.git
cd go-timer
make install   # installs to /usr/local/bin/timerd
```

Or build the binary yourself and let `timerd` install the currently running executable:

```bash
go build -o timerd .
./timerd install
```

### Requirements

- Linux with **systemd** (Ubuntu 20.04+, Debian 11+, Arch, Fedora, CentOS Stream 8+)
- `systemctl` and `journalctl` on PATH
- For user mode: `loginctl enable-linger <user>` (timerd doctor will remind you)

---

## Quick Start

```bash
# 1. Initialise
timerd init

# 2. Add a job (interactive wizard)
timerd add backup

# 3. Start it
timerd start backup

# 4. Check status
timerd list
timerd status backup

# 5. Tail logs
timerd logs backup -f

# 6. Change the config and apply
timerd edit backup
timerd reload
```

---

## Usage

### Command Reference

| Command                 | Description                                       |
| ----------------------- | ------------------------------------------------- |
| `timerd init`           | Create config directory and starter config.yml    |
| `timerd install`        | Move the current binary into `/usr/local/bin`     |
| `timerd add <name>`     | Interactive wizard to add a job                   |
| `timerd edit [name]`    | Open config.yml in `$EDITOR`                      |
| `timerd remove <name>`  | Stop, disable, remove unit files and config entry |
| `timerd start <name>`   | Install units and start the timer                 |
| `timerd stop <name>`    | Stop the timer                                    |
| `timerd restart <name>` | Restart the timer (reinstalls units)              |
| `timerd enable <name>`  | Enable timer on boot/login                        |
| `timerd disable <name>` | Disable timer from boot/login                     |
| `timerd reload`         | Regenerate all units and reload systemd           |
| `timerd status [name]`  | Show systemd status                               |
| `timerd list`           | Table of all jobs and their status                |
| `timerd logs <name>`    | Show journal logs (`-f` to follow)                |
| `timerd run <name>`     | Trigger service immediately (bypass timer)        |
| `timerd next <name>`    | Show next scheduled trigger time                  |
| `timerd inspect <name>` | Show config, generated units, and live status     |
| `timerd gen [name]`     | Print generated unit files (no install)           |
| `timerd export`         | Export all unit files to a directory              |
| `timerd doctor`         | Environment and compatibility checks              |
| `timerd version`        | Print version information                         |

### Global Flags

| Flag             | Description                         |
| ---------------- | ----------------------------------- |
| `--user`         | Systemd user mode (default)         |
| `--system`       | Systemd system mode (requires root) |
| `--dry-run`      | Print actions without executing     |
| `-v / --verbose` | Verbose output                      |

---

## Configuration

timerd manages `~/.config/timerd/config.yml` for you. You rarely need to edit it directly; use `timerd add`, `timerd edit`, or `timerd remove` instead.

```yaml
jobs:
  backup:
    command: "/home/tim/scripts/backup.sh"
    workdir: "/home/tim/scripts"
    schedule: "hourly"
    description: "Backup task"
    persistent: true # catch up missed runs after reboot

  sync:
    command: "go"
    args: ["run", "sync.go"]
    schedule: "*/5 * * * *" # every 5 minutes (cron syntax)
    env:
      APP_ENV: production
      TOKEN: secret

  nightly:
    command: "/usr/local/bin/cleanup"
    schedule: "0 3 * * *" # daily at 03:00
    restart: "on-failure"
    restart_sec: "30"
    timeout: "120"
```

### Schedule Format

Both **systemd keywords** and **cron expressions** are supported:

| Input         | Converted to              |
| ------------- | ------------------------- |
| `hourly`      | `hourly`                  |
| `daily`       | `daily`                   |
| `weekly`      | `weekly`                  |
| `*/5 * * * *` | `*-*-* *:0/5:00`          |
| `0 9 * * 1-5` | `Mon..Fri *-*-* 09:00:00` |
| `30 2 * * 0`  | `Sun *-*-* 02:30:00`      |
| `0 0 1 * *`   | `*-*-01 00:00:00`         |

### All Job Fields

| Field         | Type   | Description                           |
| ------------- | ------ | ------------------------------------- |
| `command`     | string | **Required.** Executable path         |
| `args`        | list   | Arguments for the command             |
| `schedule`    | string | **Required.** Cron or systemd keyword |
| `workdir`     | string | `WorkingDirectory` in the unit        |
| `description` | string | Human-readable description            |
| `env`         | map    | Environment variables                 |
| `user`        | string | Run as a specific user                |
| `restart`     | string | `no` / `on-failure` / `always`        |
| `restart_sec` | string | Seconds between restarts              |
| `timeout`     | string | `TimeoutStartSec` value               |
| `oneshot`     | bool   | Use `Type=oneshot`                    |
| `persistent`  | bool   | Catch up missed timer events          |
| `after`       | list   | `After=` unit dependencies            |
| `wants`       | list   | `Wants=` soft dependencies            |
| `requires`    | list   | `Requires=` hard dependencies         |
| `enabled`     | bool   | Managed by timerd; do not edit        |

---

## Architecture

```
timerd
├── cmd/            — Cobra CLI commands
├── internal/
│   ├── config/     — YAML config load/save
│   ├── cron/       — Cron → OnCalendar converter
│   ├── systemd/    — Unit generator + systemctl wrapper
│   └── ui/         — Coloured terminal output
└── pkg/
    └── errors/     — Typed error codes
```

**How it works:**

1. `config.yml` is the single source of truth.
2. On `start` / `reload`, timerd renders Go templates into `.service` + `.timer` files and writes them to `~/.config/systemd/user/` (or `/etc/systemd/system/`).
3. timerd calls `systemctl --user daemon-reload` then `systemctl --user start`.
4. All further management (logs, status, next trigger) delegates to systemd/journalctl.

---

## Linux Compatibility

| Distro               | Tested | Notes                                      |
| -------------------- | ------ | ------------------------------------------ |
| Ubuntu 22.04 / 24.04 | ✔      | Full support                               |
| Debian 12            | ✔      | Full support                               |
| Arch Linux           | ✔      | Full support                               |
| Fedora 40+           | ✔      | Full support                               |
| CentOS Stream 9      | ✔      | Full support                               |
| WSL2                 | ⚠      | Requires `systemd=true` in `/etc/wsl.conf` |
| macOS                | ❌     | systemd is Linux-only                      |

---

## User Mode vs System Mode

**User mode** (default, `--user`):

- Units live in `~/.config/systemd/user/`
- Config in `~/.config/timerd/config.yml`
- No root required
- Tip: run `loginctl enable-linger $USER` so services survive logout (`timerd doctor` will check this)

**System mode** (`--system`, requires root):

- Units live in `/etc/systemd/system/`
- Config in `/etc/timerd/config.yml`
- Suitable for system-wide daemons

---

## FAQ

**Q: Do I need to run timerd as a daemon?**  
No. timerd is a CLI tool. systemd handles the actual scheduling.

**Q: What happens to missed runs while my laptop was off?**  
Set `persistent: true` in the job config. systemd will fire the job immediately after boot if a run was missed.

**Q: Can I use the generated unit files without timerd?**  
Yes — `timerd export -o ./units` writes all unit files to a directory. You can install them manually.

**Q: How do I run a job right now?**  
`timerd run <name>` triggers the service unit immediately.

**Q: How do I see when a job last ran and when it will next run?**  
`timerd next <name>` and `timerd status <name>`.

**Q: timerd doctor says linger is off — what should I do?**

```bash
loginctl enable-linger $USER
```

This makes user services start on boot even without an interactive session.
