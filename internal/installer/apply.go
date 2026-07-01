package installer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type ApplyOptions struct {
	LegacyScript string
}

func Apply(ctx context.Context, opts ApplyOptions) error {
	if opts.LegacyScript == "" {
		opts.LegacyScript = "./scripts/setup_system_legacy.sh"
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	start := time.Now()
	emitEvent("run_started", map[string]string{
		"vps":         cfg.Values["vps"],
		"base_domain": cfg.Values["base_domain"],
	})
	emitStage(Stages[0])

	if err := runLegacyInstaller(ctx, opts.LegacyScript); err != nil {
		emitEvent("run_failed", map[string]string{"duration": time.Since(start).String()})
		return err
	}

	emitEvent("run_completed", map[string]string{"duration": time.Since(start).String()})
	return nil
}

func runLegacyInstaller(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	lines := make(chan string)
	var scanners sync.WaitGroup
	scanners.Add(2)
	go scanOutput(stdout, lines, &scanners)
	go scanOutput(stderr, lines, &scanners)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		scanners.Wait()
		close(lines)
	}()

	for line := range lines {
		fmt.Println(line)
		stageFromLine(line)
	}

	if err := <-done; err != nil {
		return err
	}
	return nil
}

func scanOutput(r io.Reader, lines chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
}
