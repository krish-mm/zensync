package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

type App struct {
	out io.Writer
	err io.Writer
}

type cliOptions struct {
	showHelp   bool
	doExport   bool
	importPath string
}

func NewApp(out, err io.Writer) *App {
	return &App{
		out: out,
		err: err,
	}
}

func (a *App) Run(args []string) int {
	opts, err := parseCLIArgs(args)
	if err != nil {
		a.printError(err)
		return 1
	}

	if opts.showHelp {
		renderHelp(a.out)
		return 0
	}

	if runtime.GOOS != "darwin" {
		a.printError(errors.New("zensync currently supports macOS only"))
		return 1
	}

	if os.Geteuid() == 0 {
		a.printError(errors.New("do not run zensync with sudo/root; run it as your normal macOS user"))
		return 1
	}

	renderLogo(a.out)
	if opts.doExport {
		if err := exportZen(a.out); err != nil {
			a.printError(err)
			return 1
		}
		return 0
	}

	if err := importZen(a.out, opts.importPath); err != nil {
		a.printError(err)
		return 1
	}

	return 0
}

func (a *App) printError(err error) {
	fmt.Fprintf(a.err, "Error: %s\n", err)
	fmt.Fprintln(a.err, "Run `zensync --help` to see commands.")
}

func parseCLIArgs(args []string) (cliOptions, error) {
	if len(args) == 0 {
		return cliOptions{showHelp: true}, nil
	}

	if len(args) == 1 {
		switch args[0] {
		case "help", "commands":
			return cliOptions{showHelp: true}, nil
		}
	}

	fs := flag.NewFlagSet("zensync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	showHelp := fs.Bool("help", false, "Show help")
	showHelpShort := fs.Bool("h", false, "Show help")
	doExport := fs.Bool("export", false, "Export Zen Browser data to a zip in ~/Downloads")
	importPath := fs.String("import", "", "Import a Zen backup zip")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	if *showHelp || *showHelpShort {
		return cliOptions{showHelp: true}, nil
	}

	if fs.NArg() > 0 {
		return cliOptions{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	trimmedImportPath := strings.TrimSpace(*importPath)
	hasImport := trimmedImportPath != ""
	if *doExport == hasImport {
		return cliOptions{}, errors.New("choose exactly one command: --export or --import <zip-path>")
	}

	return cliOptions{
		doExport:   *doExport,
		importPath: trimmedImportPath,
	}, nil
}
