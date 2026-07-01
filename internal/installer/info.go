package installer

import (
	"fmt"
	"io"
	"os/exec"
)

func Plan(w io.Writer) error {
	fmt.Fprintln(w, "Ownstack bootstrap plan:")
	for i, stage := range Stages {
		fmt.Fprintf(w, "%d. %s - %s\n", i+1, stage.Title, stage.Description)
	}
	return nil
}

func Doctor(w io.Writer) error {
	if _, err := LoadConfig(); err != nil {
		return err
	}

	fmt.Fprintln(w, "Environment contract: ok")
	for _, name := range []string{"bash", "curl"} {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("%s is required but was not found on PATH", name)
		}
		fmt.Fprintf(w, "%s: ok\n", name)
	}
	return nil
}

func Status(w io.Writer) error {
	fmt.Fprintln(w, "Status checks are not implemented yet.")
	fmt.Fprintln(w, "Next step: read Kubernetes, Harbor, Jenkins, and DNS state without mutating infrastructure.")
	return nil
}
