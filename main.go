package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	showHelp = flag.Bool("help", false, "Show help and exit")
	showH    = flag.Bool("h", false, "Show help (shorthand)")
)

func usage() {
	fmt.Fprintf(os.Stderr, `NetSpawner — local blockchain network launcher

Usage:
  netspawner [flags] <command>

Commands:
  resume   Resume network from the same point
  reset    Reset and start the network from init (progress drop)
  help     Show this help

Flags:
  -h, -help   Show help and exit

Examples:
  netspawner resume
  netspawner reset
  netspawner -h
`)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *showHelp || *showH {
		usage()
		return
	}

	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	cmd := strings.ToLower(flag.Arg(0))
	if cmd == "help" {
		usage()
		return
	}

	var err error
	switch cmd {
	case "resume":
		err = resumeNetwork()
	case "reset":
		err = resetNetwork()
	default:
		err = fmt.Errorf("unknown command: %q", cmd)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func resumeNetwork() error {
	fmt.Println("Resuming network from the same point...")
	return nil
}

func resetNetwork() error {
	fmt.Println("Resetting network and starting from scratch...")
	return nil
}

func notImplemented(feature string) error {
	return errors.New(feature + " not implemented")
}
