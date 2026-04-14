package main

import (
	"fmt"
	"io"
	"strings"
)

const (
	logoRingColor = "\x1b[38;2;247;111;83m" // #F76F53
	colorReset    = "\x1b[0m"
)

const logo = `
     .-----------------------.
   .'                         '.
  /                             \
 |      @@@@@@@@@@@@@@@@@@@      |
 |    @@@@@@@@@@@@@@@@@@@@@@@    |
 |   @@@@@@@@         @@@@@@@@   |
 |   @@@@@    @@@@@@@    @@@@@   |
 |   @@@@   @@@     @@@   @@@@   |
 |   @@@@  @@@   @   @@@  @@@@   |
 |   @@@@  @@@  @@@  @@@  @@@@   |
 |   @@@@  @@@   @   @@@  @@@@   |
 |   @@@@   @@@     @@@   @@@@   |
 |   @@@@@    @@@@@@@    @@@@@   |
 |   @@@@@@@@         @@@@@@@@   |
 |    @@@@@@@@@@@@@@@@@@@@@@@    |
 |      @@@@@@@@@@@@@@@@@@@      |
  \                             /
   '.                         .'
     '-----------------------'
`

func renderLogo(w io.Writer) {
	coloredLogo := strings.ReplaceAll(logo, "@", logoRingColor+"@"+colorReset)
	fmt.Fprintln(w, coloredLogo)
}

func renderHelp(w io.Writer) {
	renderLogo(w)
	fmt.Fprintln(w, "ZenSync backs up and restores Zen Browser profile data on macOS.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  zensync --export")
	fmt.Fprintln(w, "  zensync --import <zip-path>")
	fmt.Fprintln(w, "  zensync --help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  zensync")
	fmt.Fprintln(w, "  zensync --export")
	fmt.Fprintln(w, "  zensync --import ~/Downloads/zen_backup_YYYY-MM-DD_HH-MM-SS.zip")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Note: run as normal user (do not use sudo).")
}

func printSection(w io.Writer, title string) {
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("-", len(title)))
}

func printInfo(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "  - "+format+"\n", args...)
}

func printSuccess(w io.Writer, message string) {
	fmt.Fprintf(w, "\nSuccess: %s\n", message)
}

func printPath(w io.Writer, label, path string) {
	fmt.Fprintf(w, "  %s: %s\n", label, path)
}
