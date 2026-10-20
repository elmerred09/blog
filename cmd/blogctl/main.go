package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "blogctl:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: blogctl css")
	}
	switch args[0] {
	case "css":
		return runCSS(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
