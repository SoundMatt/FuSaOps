# `fusaops hooks`

Manage a git pre-commit hook that runs `fusaops check --strict` before every
commit.

```bash
fusaops hooks <install|remove|show> [--dir <path>]
```

## Subcommands

| Subcommand | Description |
|---|---|
| `install` | Install the pre-commit hook into `.git/hooks/pre-commit` |
| `remove` | Remove the FuSaOps pre-commit hook |
| `show` | Print the hook script to stdout without installing it |

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root (used to locate `.git/hooks`) |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Hook already exists (`install`) or hook not found (`remove`) |
| 2 | Usage error or missing subcommand |

## Behaviour

- `install` refuses to overwrite an existing hook — run `hooks remove` first.
- The installed script is a no-op (with a warning) if `fusaops` is not on
  `PATH` at commit time, so it never blocks a commit on a machine without the
  tool installed.

## Example

```bash
fusaops hooks install     # wire fusaops check --strict into every commit
fusaops hooks show        # inspect the script before installing it
fusaops hooks remove      # uninstall
```
