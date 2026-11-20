package main

import (
	"fmt"
	"io"
	"os"
)

// stdout and stderr are variables so tests can capture output.
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func main() {
	cfg, act, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "hostnamefetch: %s\n", err)
		os.Exit(1)
	}

	switch act {
	case actionHelp:
		fmt.Fprint(stdout, usageText)
		return
	case actionVersion:
		fmt.Fprintf(stdout, "hostnamefetch %s\n", version)
		return
	}

	env, info := collect()

	out, err := render(cfg, info, env)
	if err != nil {
		fmt.Fprintf(stderr, "hostnamefetch: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprint(stdout, out)
}
