// stop-hook is the Claude Code Stop hook binary.
// It reads the hook payload from stdin, appends the transcript path to the queue,
// and then spawns `lajan run --last` as a detached background process so the
// analysis happens automatically without blocking Claude Code.
//
// Claude Code Stop hook input (stdin):
//
//	{
//	  "hook_event_name": "Stop",
//	  "session_id": "...",
//	  "transcript_path": "/path/to/session.jsonl",
//	  "stop_hook_active": true
//	}
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/ariary/claude-lajan/internal/config"
)

type stopInput struct {
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	StopHookActive bool   `json:"stop_hook_active"`
}

func main() {
	if !config.IsEnabled() {
		os.Exit(0)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop-hook: read stdin: %v\n", err)
		os.Exit(1)
	}

	var input stopInput
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(os.Stderr, "stop-hook: parse input: %v\n", err)
		os.Exit(1)
	}

	if input.TranscriptPath == "" {
		fmt.Fprintln(os.Stderr, "stop-hook: no transcript_path in hook input")
		os.Exit(1)
	}

	if err := appendToQueue(input.TranscriptPath); err != nil {
		fmt.Fprintf(os.Stderr, "stop-hook: queue: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "stop-hook: queued %s — launching lajan in background\n", input.TranscriptPath)

	if err := spawnReviewer(); err != nil {
		// Non-fatal: the session is queued, user can run manually.
		fmt.Fprintf(os.Stderr, "stop-hook: could not spawn lajan automatically: %v\n", err)
		fmt.Fprintln(os.Stderr, "stop-hook: run `lajan run` manually to process the queue.")
	}
	// Exit 0 immediately — Claude Code does not wait for the background process.
}

// spawnReviewer starts `lajan run --last` as a detached background process.
// The new process is in its own process group (Setpgid) so it survives after
// the hook exits, and output is redirected to a log file.
func spawnReviewer() error {
	lajanBin := filepath.Join(config.BinDir(), "lajan")
	if _, err := os.Stat(lajanBin); err != nil {
		return fmt.Errorf("lajan binary not found at %s — run `lajan install` first", lajanBin)
	}

	logFile := filepath.Join(config.ReviewerDir(), "lajan.log")
	lf, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}

	cmd := exec.Command(lajanBin, "run", "--last")
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.Stdin = nil
	// Detach from parent process group so the process outlives the hook.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		lf.Close()
		return fmt.Errorf("start lajan: %w", err)
	}

	// Disown — we do not call cmd.Wait(). The process runs independently.
	go func() { _ = cmd.Wait() }()
	lf.Close()
	return nil
}

func appendToQueue(path string) error {
	queueFile := config.QueueFile()
	if err := os.MkdirAll(filepath.Dir(queueFile), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(queueFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, path)
	return err
}
