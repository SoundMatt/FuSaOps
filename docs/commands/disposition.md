# `fusaops disposition`

Manage finding disposition entries — recorded accept/fix decisions with a
reviewer and rationale — in `.fusaops-dispositions.json`.

```bash
fusaops disposition <add|list|show> [flags]
```

## Subcommands and flags

### `add`

```bash
fusaops disposition add --rule <ruleID> --reviewer <name> --rationale <text>
                         [--lang <language>] [--action accept|fix] [--ref <ticket>]
```

| Flag | Required | Description |
|---|---|---|
| `--rule` | yes | Rule ID to disposition |
| `--reviewer` | yes | Reviewer name |
| `--rationale` | yes | Rationale for the disposition |
| `--lang` | no | Language (e.g. `go`, `rust`, `python`) |
| `--action` | no | `accept` (default) or `fix` |
| `--ref` | no | Optional reference (issue, ticket, etc.) |

### `list`

Prints every disposition entry.

### `show`

```bash
fusaops disposition show --rule <ruleID> [--lang <language>]
```

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Load/save error, or `show` found no matching entry |
| 2 | Usage error or missing/invalid required flags |

## Example

```bash
fusaops disposition add --rule SAFETY017 --reviewer "J. Rivera" \
  --rationale "False positive: guarded by static_assert" --action accept
fusaops disposition list
```

Supports ISO 26262-8 §6 / DO-178C §7.2.4 review-and-disposition workflows for
findings a team has knowingly accepted rather than fixed.
