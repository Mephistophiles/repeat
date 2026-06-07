# repeat

Run a command N times, with logging, progress bar, and JSON output.

## Install

```bash
go install github.com/Mephistophiles/repeat/cmd/repeat@latest
```

Or build from source:

```bash
git clone https://github.com/Mephistophiles/repeat.git
cd repeat
go build -o repeat ./cmd/repeat/
```

## Usage

```
repeat [flags] <N> <command> [args...]
repeat [flags] --until-success <command> [args...]
```

### Examples

```bash
# Run echo 3 times
repeat 3 echo hello

# Run with verbose output
repeat -v 3 echo hello

# Delay between runs
repeat -d 1s 5 curl https://example.com

# Stop on first error (default)
repeat 5 ./flaky-test.sh

# Continue on errors, run all N times
repeat --continue 5 ./flaky-test.sh

# Run until command succeeds
repeat --until-success curl https://api.example.com/health

# Timeout per run
repeat -t 30s 3 ./slow-task.sh

# JSON summary at the end
repeat --json 10 pytest test_suite.py

# TUI progress bar with ETA
repeat --progress 100 ./bench.sh
repeat --progress -v 10 pytest  # with scrolling output
repeat --progress --until-success curl api
```

### Flags

| Flag | Description |
|------|-------------|
| `-v`, `--verbose` | Show command output (stdout in normal mode, scrolling area in TUI) |
| `-d`, `--delay` | Delay between runs (`1s`, `500ms`, ...) |
| `-t`, `--timeout` | Timeout per run (`30s`, `5m`, ...). No timeout by default |
| `--continue` | Run all N times even on errors |
| `--until-success` | Run until exit code 0 (incompatible with `<N>`) |
| `--progress` | Bubbletea TUI with progress bar, ETA, status |
| `--json` | JSON summary after all runs (after TUI if `--progress`) |
| `-h`, `--help` | Show help |
| `--version` | Show version |

## Logs

Each run session creates a directory `.repeat/<ISO8601_TIMESTAMP>/` with log files:

```
.repeat/
├── last -> 2026-06-07T14-30-05/   # symlink to latest session
└── 2026-06-07T14-30-05/
    ├── run.1.log
    ├── run.2.log
    └── run.3.log
```

Log file format:

```
# repeat — run 1 of 3
# command: echo hello
# started:  2026-06-07T14:30:05.123+03:00
---
hello

---
# finished: 2026-06-07T14:30:05.145+03:00
# duration: 22ms
# exit code: 0
```

View latest run output:

```bash
cat .repeat/last/run.1.log
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| N | Command exit code (propagated from failed command) |
| 124 | Run timed out (`--timeout`) |
| 130 | Interrupted (Ctrl+C) |
| 2 | Invalid arguments |

## TUI (--progress)

The TUI shows real-time progress with ETA. Two modes:

**Normal** — progress bar with estimated time remaining.

**Until-success** — spinner with attempt counter and elapsed time.

**Verbose** (`--progress -v`) — adds a scrolling output area showing the current command's output.

## JSON output (--json)

```json
{
  "command": "echo hello",
  "runs": [
    {"run":1,"exit_code":0,"duration_ms":22,"stdout":"hello\n","stderr":""},
    {"run":2,"exit_code":0,"duration_ms":18,"stdout":"hello\n","stderr":""}
  ],
  "total": 2,
  "successes": 2,
  "failures": 0,
  "total_duration_ms": 40
}
```

## Testing

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test -tags=integration ./test/integration/ -v

# All tests
go test ./... -count=1
go test -tags=integration ./test/integration/ -count=1
```

## License

MIT
