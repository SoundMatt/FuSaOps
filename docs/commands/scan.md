# `fusaops scan`

Detect the languages present in a repository and which adapters apply.

```bash
fusaops scan [--dir <path>]
```

## Output

- Detected languages with source-file counts (descending).
- Applicable adapters and whether each tool is installed on `PATH`.

## Example

```text
Detected languages:
  go    34 source file(s)

Applicable adapters:
  go-FuSa    gofusa   (installed)
```

Use `scan` before `check` to confirm the expected languages are discovered and
the expected tools are available — an unexpected "NOT installed" is a coverage
gap (Tool Safety Manual AoU-3).
