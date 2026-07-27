# `fusaops pr`

Manage software problem reports per DO-178C §11.17, stored in
`.fusaops-problems.json`.

```bash
fusaops pr <init|add|list|close> [flags]
```

## Subcommands

### `init`

Creates an empty `.fusaops-problems.json`.

### `add`

```bash
fusaops pr add --id <id> --title <text> [--desc <text>]
               [--phase planning|development|verification|integration|operation]
               [--severity critical|major|minor]
```

| Flag | Required | Default | Description |
|---|---|---|---|
| `--id` | yes | — | Problem report ID |
| `--title` | yes | — | Short description |
| `--desc` | no | — | Detailed description |
| `--phase` | no | `development` | Phase found |
| `--severity` | no | `minor` | Severity |

### `list`

```bash
fusaops pr list [--format text|json]
```

### `close`

```bash
fusaops pr close --id <id> [--resolution <text>]
```

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Load/save error |
| 2 | `init` when the file already exists, missing required flags, or usage error |

## Example

```bash
fusaops pr init
fusaops pr add --id PR-001 --title "Race in check aggregation" --severity major
fusaops pr close --id PR-001 --resolution "Fixed in v1.130.0"
```

Serves the DO-178C §11.17 Software Problem Report objective by recording
problems and their disposition against the multi-language toolchain.
