package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mephistophiles/repeat/internal/log"
	"github.com/Mephistophiles/repeat/internal/runner"
	"github.com/Mephistophiles/repeat/internal/summary"
	"github.com/Mephistophiles/repeat/internal/tui"
)

var buildVersion = "dev"

func main() {
	verbose := flag.Bool("v", false, "show command output in stdout (TUI: scroll area)")
	delay := flag.Duration("d", 0, "delay between runs (e.g. 1s, 500ms)")
	timeout := flag.Duration("t", 0, "timeout per run (e.g. 30s, 5m)")
	cont := flag.Bool("continue", false, "run all N times even on error")
	untilSuccess := flag.Bool("until-success", false, "run until command succeeds (exit 0)")
	progress := flag.Bool("progress", false, "show TUI with progress bar and ETA")
	jsonOut := flag.Bool("json", false, "output JSON summary at the end")
	versionFlag := flag.Bool("version", false, "show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: repeat [flags] <N> <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "       repeat [flags] --until-success <command> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Println("repeat", buildVersion)
		os.Exit(0)
	}

	if *untilSuccess && flag.NArg() >= 1 {
		if n, err := strconv.Atoi(flag.Arg(0)); err == nil && n >= 1 {
			fmt.Fprintln(os.Stderr, "error: --until-success is incompatible with <N>")
			os.Exit(2)
		}
	}

	if *untilSuccess && flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: --until-success requires a command")
		os.Exit(2)
	}
	if !*untilSuccess && flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "error: N and command required")
		os.Exit(2)
	}

	var total int
	var command string
	var cmdArgs []string

	if *untilSuccess {
		command = flag.Arg(0)
		cmdArgs = flag.Args()[1:]
	} else {
		n, err := strconv.Atoi(flag.Arg(0))
		if err != nil || n < 1 {
			fmt.Fprintf(os.Stderr, "error: N must be a positive integer, got %q\n", flag.Arg(0))
			os.Exit(2)
		}
		total = n
		command = flag.Arg(1)
		cmdArgs = flag.Args()[2:]
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	session, err := log.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	mode := tui.ModeNormal
	if *untilSuccess {
		mode = tui.ModeUntilSuccess
	}

	stopOnErr := !*cont && !*untilSuccess

	if *progress {
		runTUI(ctx, cancel, session, mode, total, command, cmdArgs, *timeout, *delay, *verbose, stopOnErr, *jsonOut)
	} else {
		exitCode := runSimple(ctx, session, total, command, cmdArgs, *timeout, *delay, *verbose, stopOnErr, *jsonOut, mode)
		os.Exit(exitCode)
	}
}

func runSimple(ctx context.Context, session *log.Session, total int, command string, args []string, timeout, delay time.Duration, verbose bool, stopOnErr bool, jsonOut bool, mode tui.Mode) int {
	var results []runner.Result

	for i := 1; mode == tui.ModeUntilSuccess || i <= total; i++ {
		if i > 1 && delay > 0 {
			select {
			case <-ctx.Done():
				return 130
			case <-time.After(delay):
			}
		}

		select {
		case <-ctx.Done():
			return 130
		default:
		}

		logFile, err := session.CreateRunFile(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}

		fullCmd := command
		if len(args) > 0 {
			fullCmd += " " + strings.Join(args, " ")
		}
		log.WriteHeader(logFile, i, total, fullCmd, time.Now())

		var onOut func(string)
		if verbose {
			onOut = func(line string) { fmt.Println(line) }
		}

		opts := runner.Opts{
			Index:    i,
			Command:  command,
			Args:     args,
			Timeout:  timeout,
			LogW:     logFile,
			LogWErr:  logFile,
			OnOutput: onOut,
		}

		res := runner.Run(ctx, opts)
		log.WriteFooter(logFile, res)
		logFile.Close()
		session.UpdateRunSymlink(i)
		results = append(results, res)

		fmt.Fprintf(os.Stderr, "%s  %s  %s\n", progressPrefix(res.RunIndex, total, mode), progressStatus(res), progressDuration(res.Duration))

		if verbose && res.Interrupted {
			fmt.Fprintln(os.Stderr, "interrupted")
		}

		if mode == tui.ModeUntilSuccess && res.OK() {
			break
		}

		if stopOnErr && !res.OK() {
			if jsonOut {
				printJSON(command, args, results)
			}
			return res.ExitStatus()
		}
	}

	if jsonOut {
		printJSON(command, args, results)
	}

	if mode == tui.ModeUntilSuccess {
		return 0
	}

	return runner.LastErrorExitCode(results)
}

func runTUI(ctx context.Context, cancel context.CancelFunc, session *log.Session, mode tui.Mode, total int, command string, args []string, timeout, delay time.Duration, verbose bool, stopOnErr bool, jsonOut bool) {
	cfg := tui.Config{
		Mode:        mode,
		Total:       total,
		Command:     command + " " + strings.Join(args, " "),
		Verbose:     verbose,
		Delay:       delay,
		StopOnError: stopOnErr,
	}

	model := tui.NewModel(cfg)

	go func() {
		defer func() {
			close(model.MsgChan)
		}()

		for i := 1; mode == tui.ModeUntilSuccess || i <= total; i++ {
			if i > 1 && delay > 0 {
				select {
				case <-ctx.Done():
					model.MsgChan <- tui.InterruptedMsg{}
					return
				case <-time.After(delay):
				}
			}

			select {
			case <-ctx.Done():
				model.MsgChan <- tui.InterruptedMsg{}
				return
			default:
			}

			logFile, err := session.CreateRunFile(i)
			if err != nil {
				model.MsgChan <- tui.InterruptedMsg{}
				return
			}

			fullCmd := command
			if len(args) > 0 {
				fullCmd += " " + strings.Join(args, " ")
			}
			log.WriteHeader(logFile, i, total, fullCmd, time.Now())

			model.MsgChan <- tui.RunStartedMsg{Index: i}

			var onOut func(string)
			if verbose {
				onOut = func(line string) {
					model.MsgChan <- tui.RunOutputMsg{Index: i, Line: line}
				}
			}

			opts := runner.Opts{
				Index:    i,
				Command:  command,
				Args:     args,
				Timeout:  timeout,
				LogW:     logFile,
				LogWErr:  logFile,
				OnOutput: onOut,
			}

			res := runner.Run(ctx, opts)
			log.WriteFooter(logFile, res)
			logFile.Close()
			session.UpdateRunSymlink(i)

			msg := tui.RunCompletedMsg{Result: res}
			model.MsgChan <- msg

			if mode == tui.ModeUntilSuccess && res.OK() {
				return
			}

			if stopOnErr && !res.OK() {
				return
			}
		}
	}()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	results := finalModel.(tui.Model).Results
	if jsonOut {
		printJSON(command, args, results)
	}

	os.Exit(runner.LastErrorExitCode(results))
}

func progressPrefix(index, total int, mode tui.Mode) string {
	if mode == tui.ModeUntilSuccess {
		return fmt.Sprintf("[%d]", index)
	}
	return fmt.Sprintf("[%d/%d]", index, total)
}

func progressStatus(r runner.Result) string {
	if r.TimedOut {
		return "timeout"
	}
	if r.Interrupted {
		return "interrupted"
	}
	if r.ExitCode != 0 {
		return fmt.Sprintf("exit %d", r.ExitCode)
	}
	return "ok"
}

func progressDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := d / time.Minute
		s := (d % time.Minute).Seconds()
		return fmt.Sprintf("%dm %.0fs", m, s)
	}
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	return fmt.Sprintf("%dh %dm", h, m)
}

func printJSON(command string, args []string, results []runner.Result) {
	fullCmd := command
	if len(args) > 0 {
		fullCmd += " " + strings.Join(args, " ")
	}
	out := summary.Build(fullCmd, results)
	summary.Write(os.Stdout, out)
}
