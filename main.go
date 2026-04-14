package main

import "os"

func main() {
	app := NewApp(os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
