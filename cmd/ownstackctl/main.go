package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/getownstack/ownstack-cluster-template/internal/installer"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "apply":
		fs := flag.NewFlagSet("apply", flag.ExitOnError)
		legacyScript := fs.String("legacy-script", "./scripts/setup_system_legacy.sh", "path to the legacy shell installer")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return installer.Apply(context.Background(), installer.ApplyOptions{LegacyScript: *legacyScript})
	case "plan":
		return installer.Plan(os.Stdout)
	case "doctor":
		return installer.Doctor(os.Stdout)
	case "status":
		return installer.Status(os.Stdout)
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "ownstackctl manages the customer-owned Ownstack cluster bootstrap.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  ownstackctl plan")
	fmt.Fprintln(os.Stderr, "  ownstackctl doctor")
	fmt.Fprintln(os.Stderr, "  ownstackctl apply [--legacy-script ./scripts/setup_system_legacy.sh]")
	fmt.Fprintln(os.Stderr, "  ownstackctl status")
}
