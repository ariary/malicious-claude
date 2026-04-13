package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/debate"
	"github.com/ariary/claude-lajan/internal/report"
	"github.com/ariary/claude-lajan/internal/session"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "lajan",
		Short: "claude-lajan — Claude Code session analyser and self-improvement tool",
	}

	root.AddCommand(
		cmdRun(),
		cmdList(),
		cmdDigest(),
		cmdSummarize(),
		cmdReset(),
		cmdStatus(),
		cmdEnable(),
		cmdDisable(),
		cmdInstall(),
		cmdUninstall(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── run ──────────────────────────────────────────────────────────────────────

func cmdRun() *cobra.Command {
	var last bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Process queued sessions through the adversarial debate engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.AnthropicAPIKey() == "" {
				return fmt.Errorf("ANTHROPIC_API_KEY is not set")
			}

			if last {
				return processLast(dryRun)
			}
			return processQueue(dryRun)
		},
	}
	cmd.Flags().BoolVar(&last, "last", false, "Process only the most recent session from queue")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run debate but skip writing reports and injecting learnings")
	return cmd
}

func processQueue(dryRun bool) error {
	paths, err := readQueue()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Println("Queue is empty. Run a Claude Code session first, or check that the stop hook is installed.")
		return nil
	}
	fmt.Printf("Processing %d queued session(s)...\n", len(paths))
	var failed []string
	for _, p := range paths {
		if err := processSession(p, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "  error processing %s: %v\n", p, err)
			failed = append(failed, p)
		}
	}
	// Clear queue (rewrite with only failures)
	return writeQueue(failed)
}

func processLast(dryRun bool) error {
	paths, err := readQueue()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Println("Queue is empty.")
		return nil
	}
	p := paths[len(paths)-1]
	if err := processSession(p, dryRun); err != nil {
		return err
	}
	// Remove last from queue
	return writeQueue(paths[:len(paths)-1])
}

func processSession(jsonlPath string, dryRun bool) error {
	fmt.Printf("\n→ Analysing session: %s\n", filepath.Base(jsonlPath))

	s, err := session.LoadFromPath(jsonlPath)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	fmt.Printf("  Duration: %s | Turns: %d user / %d assistant | Tool calls: %d\n",
		s.EndTime.Sub(s.StartTime).Round(1e9),
		s.UserTurns, s.AssistantTurns, len(s.ToolCalls))

	userCfg, _ := config.LoadUserConfig()
	result, err := debate.RunN(context.Background(), s, userCfg.MaxDebateRounds)
	if err != nil {
		return fmt.Errorf("debate: %w", err)
	}

	if dryRun {
		fmt.Printf("\n[dry-run] %d findings (not written):\n", len(result.Findings))
		for _, f := range result.Findings {
			fmt.Printf("  [%s/%s] %s\n", f.Type, f.Scope, f.Text)
		}
		return nil
	}

	reportPath, err := report.Write(s, result)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	fmt.Printf("  Report written: %s\n", reportPath)

	if err := report.UpdateDigest(result.Findings); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: digest update failed: %v\n", err)
	}

	if err := report.InjectFindings(result.Findings, s.CWD); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: finding injection failed: %v\n", err)
	}

	fmt.Printf("  %d findings injected (%d project-scoped, %d global)\n",
		len(result.Findings), countScope(result.Findings, debate.ScopeProject), countScope(result.Findings, debate.ScopeGlobal))

	// Inject any hook suggestions from findings
	added, err := report.InjectSuggestedHooks(result.Findings, s.CWD)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  warning: hook injection failed: %v\n", err)
	}
	for _, h := range added {
		fmt.Printf("  → Hook added: %s\n", h)
	}

	return nil
}

func countScope(findings []debate.Finding, scope debate.Scope) int {
	n := 0
	for _, f := range findings {
		if f.Scope == scope {
			n++
		}
	}
	return n
}

// ── list ─────────────────────────────────────────────────────────────────────

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show sessions pending review in the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := readQueue()
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				fmt.Println("Queue is empty.")
				return nil
			}
			fmt.Printf("%d session(s) pending review:\n", len(paths))
			for _, p := range paths {
				fmt.Printf("  %s\n", p)
			}
			return nil
		},
	}
}

// ── digest ────────────────────────────────────────────────────────────────────

func cmdDigest() *cobra.Command {
	return &cobra.Command{
		Use:   "digest",
		Short: "Print the current rolling digest of learnings",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(config.DigestFile())
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No digest yet. Run `lajan run` after some sessions.")
					return nil
				}
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

// ── summarize ────────────────────────────────────────────────────────────────

func cmdSummarize() *cobra.Command {
	return &cobra.Command{
		Use:   "summarize",
		Short: "Show all learnings currently injected into Claude Code sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			printed := false

			// Global learnings (CLAUDE.md)
			global := report.ReadGlobalLearnings()
			if len(global) > 0 {
				printed = true
				fmt.Println("## Global learnings  (~/.claude/CLAUDE.md)")
				fmt.Println("   Injected into every Claude Code session on this machine.")
				for _, line := range global {
					fmt.Printf("  • %s\n", line)
				}
				fmt.Println()
			}

			// Project learnings (project memory)
			project := report.ReadProjectLearnings(cwd)
			if len(project) > 0 {
				printed = true
				fmt.Printf("## Project learnings  (%s)\n", cwd)
				fmt.Println("   Injected when Claude Code runs in this directory.")
				for _, line := range project {
					fmt.Printf("  • %s\n", line)
				}
				fmt.Println()
			}

			if !printed {
				fmt.Println("No learnings injected yet. Run `lajan run` after a Claude Code session.")
			}

			// Session evolution
			summaries := report.LoadRecentSummaries(5)
			if len(summaries) >= 2 {
				fmt.Println("## Session evolution  (last 5 sessions)")
				fmt.Println()
				fmt.Printf("%-18s%-11s%-10s%-10s%-7s%s\n", "DATE", "DURATION", "COST", "TOKENS", "TOOLS", "FAILED")
				for _, s := range summaries {
					fmt.Printf("%-18s%-11s%-10s%-10s%-7d%d\n",
						s.Date,
						s.DurationStr,
						fmt.Sprintf("$%.4f", s.EstimatedCostUSD),
						formatTokensComma(s.TotalTokens),
						s.ToolCallsTotal,
						s.ToolCallsFailed,
					)
				}
				fmt.Println()

				first := summaries[0]
				last := summaries[len(summaries)-1]
				fmt.Print("Trend: ")
				var parts []string
				if first.EstimatedCostUSD > 0 {
					costPct := (last.EstimatedCostUSD - first.EstimatedCostUSD) / first.EstimatedCostUSD * 100
					arrow := "↓"
					if costPct > 0 {
						arrow = "↑"
					}
					parts = append(parts, fmt.Sprintf("cost %s%.0f%%", arrow, math.Abs(costPct)))
				}
				if first.ToolCallsFailed > 0 {
					failPct := float64(last.ToolCallsFailed-first.ToolCallsFailed) / float64(first.ToolCallsFailed) * 100
					arrow := "↓"
					if failPct > 0 {
						arrow = "↑"
					}
					parts = append(parts, fmt.Sprintf("failed tools %s%.0f%%", arrow, math.Abs(failPct)))
				} else if last.ToolCallsFailed == 0 {
					parts = append(parts, "failed tools ↓100%")
				}
				fmt.Println(strings.Join(parts, " · "))
				fmt.Println()
			}

			return nil
		},
	}
}

// formatTokensComma formats an int64 with comma separators for the summarize output.
func formatTokensComma(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

// ── reset ─────────────────────────────────────────────────────────────────────

func cmdReset() *cobra.Command {
	var (
		global    bool
		project   bool
		digest    bool
		hooks     bool
		all       bool
		noConfirm bool
	)

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove injected learnings and/or hooks",
		Long: `Cleanly remove learnings injected by lajan without manually editing files.

Flags can be combined. With no flags, shows what would be removed (dry-run).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			if all {
				global, project, digest, hooks = true, true, true, true
			}

			// Nothing selected → show summary instead
			if !global && !project && !digest && !hooks {
				fmt.Println("Nothing selected. Use flags to choose what to remove:")
				fmt.Println()
				fmt.Println("  --global   Remove global learnings from ~/.claude/CLAUDE.md")
				fmt.Println("  --project  Remove project learnings from project memory (current dir)")
				fmt.Println("  --digest   Clear the rolling digest (~/.claude-lajan/digest.md)")
				fmt.Println("  --hooks    Uninstall lajan hooks from ~/.claude/settings.json")
				fmt.Println("  --all      All of the above")
				fmt.Println("\nCurrent state:")
				return runSubCommand([]string{"summarize"})
			}

			// Preview what will be removed
			fmt.Println("The following will be removed:")
			if global {
				lines := report.ReadGlobalLearnings()
				fmt.Printf("\n  Global CLAUDE.md section (%d entries):\n", len(lines))
				for _, l := range lines {
					fmt.Printf("    • %s\n", l)
				}
			}
			if project {
				lines := report.ReadProjectLearnings(cwd)
				fmt.Printf("\n  Project memory for %s (%d entries):\n", cwd, len(lines))
				for _, l := range lines {
					fmt.Printf("    • %s\n", l)
				}
			}
			if digest {
				fmt.Printf("\n  Digest file: %s\n", config.DigestFile())
			}
			if hooks {
				fmt.Printf("\n  Hooks from: %s\n", config.GlobalSettings())
			}

			if !noConfirm {
				fmt.Print("\nProceed? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if global {
				if err := report.ResetGlobal(); err != nil {
					return fmt.Errorf("reset global: %w", err)
				}
				fmt.Println("✓ Global CLAUDE.md section removed.")
			}
			if project {
				if err := report.ResetProject(cwd); err != nil {
					return fmt.Errorf("reset project: %w", err)
				}
				fmt.Printf("✓ Project memory removed (%s).\n", cwd)
			}
			if digest {
				if err := report.ResetDigest(); err != nil {
					return fmt.Errorf("reset digest: %w", err)
				}
				fmt.Println("✓ Digest cleared.")
			}
			if hooks {
				if err := report.UninstallHooks(); err != nil {
					return fmt.Errorf("uninstall hooks: %w", err)
				}
				if err := report.RemoveSuggestedHooks(); err != nil {
					fmt.Fprintf(os.Stderr, "  warning: remove suggested hooks: %v\n", err)
				}
				fmt.Println("✓ Hooks removed from settings.json.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Remove global learnings from ~/.claude/CLAUDE.md")
	cmd.Flags().BoolVar(&project, "project", false, "Remove project learnings from project memory (current dir)")
	cmd.Flags().BoolVar(&digest, "digest", false, "Clear the rolling digest")
	cmd.Flags().BoolVar(&hooks, "hooks", false, "Uninstall lajan hooks from ~/.claude/settings.json")
	cmd.Flags().BoolVar(&all, "all", false, "Remove everything (global + project + digest + hooks)")
	cmd.Flags().BoolVar(&noConfirm, "yes", false, "Skip confirmation prompt")
	return cmd
}

// runSubCommand re-executes a lajan sub-command in the same process.
func runSubCommand(args []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	c := exec.Command(self, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ── status ────────────────────────────────────────────────────────────────────

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current claude-lajan configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadUserConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			home, _ := os.UserHomeDir()
			tilde := func(p string) string {
				if len(p) >= len(home) && p[:len(home)] == home {
					return "~" + p[len(home):]
				}
				return p
			}

			fmt.Println("claude-lajan status")
			fmt.Println("───────────────────────────────────────")
			fmt.Printf("Enabled:           %v\n", cfg.Enabled)
			fmt.Printf("Max debate rounds: %d\n", cfg.MaxDebateRounds)
			fmt.Printf("Digest inject top: %d\n", cfg.DigestInjectTop)
			fmt.Println()
			fmt.Println("Injection:")
			fmt.Printf("  memory_inject:   %-6v (writes to project memory + CLAUDE.md)\n", cfg.Hooks.MemoryInject)
			fmt.Println()
			fmt.Println("Paths:")
			fmt.Printf("  Config:          %s\n", tilde(config.ConfigFile()))
			fmt.Printf("  Queue:           %s\n", tilde(config.QueueFile()))
			fmt.Printf("  Reports:         %s\n", tilde(config.ReportsDir())+"/")
			fmt.Printf("  Digest:          %s\n", tilde(config.DigestFile()))
			fmt.Printf("  Log:             %s\n", tilde(config.ReviewerDir())+"/lajan.log")
			fmt.Printf("  Binaries:        %s\n", tilde(config.BinDir())+"/")
			return nil
		},
	}
}

// ── enable / disable ──────────────────────────────────────────────────────────

func cmdEnable() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable claude-lajan (hooks will run normally)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadUserConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg.Enabled = true
			if err := config.SaveUserConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("claude-lajan enabled. Analysis will run after each Claude Code session.")
			return nil
		},
	}
}

func cmdDisable() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable claude-lajan (hooks exit immediately without processing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadUserConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg.Enabled = false
			if err := config.SaveUserConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("claude-lajan disabled. Hooks are still registered but will exit immediately.")
			return nil
		},
	}
}

// ── install ───────────────────────────────────────────────────────────────────

func cmdInstall() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Register lajan hooks in ~/.claude/settings.json",
		Long:  "Adds Stop hook pointing to ~/.claude-lajan/bin/. Run `make install` to also build and copy the binaries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return install()
		},
	}
}

func install() error {
	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Check binaries are present (built by Makefile)
	for _, bin := range []string{"stop-hook", "lajan"} {
		p := filepath.Join(binDir, bin)
		if _, err := os.Stat(p); err != nil {
			fmt.Printf("  warning: %s not found — run `make install` from the claude-lajan directory first\n", p)
		}
	}

	if err := patchSettings(binDir); err != nil {
		return fmt.Errorf("patch settings: %w", err)
	}

	if err := installZshCompletion(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: zsh completion not installed: %v\n", err)
	}

	fmt.Println("Hooks registered in ~/.claude/settings.json")
	fmt.Printf("  Stop hook: %s/stop-hook  (queues session + runs lajan in background)\n", binDir)
	fmt.Println("\nSessions will be reviewed automatically when Claude Code stops.")
	return nil
}

// ── uninstall ─────────────────────────────────────────────────────────────────

func cmdUninstall() *cobra.Command {
	var noConfirm bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all lajan traces from Claude Code and shell config",
		Long: `Cleanly removes everything lajan installed on the Claude Code side:
  • Stop hook from ~/.claude/settings.json
  • Hookify rule files written by lajan
  • Managed section from ~/.claude/CLAUDE.md
  • Rolling digest (~/.claude-lajan/digest.md)
  • Zsh completion line from ~/.zshrc

Binaries and ~/.claude-lajan/ are removed by 'make uninstall'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !noConfirm {
				fmt.Println("This will remove all lajan traces from ~/.claude/ and ~/.zshrc.")
				fmt.Print("Proceed? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := report.UninstallHooks(); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
			} else {
				fmt.Println("✓ Stop hook removed from ~/.claude/settings.json")
			}

			if err := report.RemoveSuggestedHooks(); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "  warning: remove suggested hooks: %v\n", err)
			} else {
				fmt.Println("✓ Hookify rule files removed")
			}

			if err := report.ResetGlobal(); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: reset CLAUDE.md: %v\n", err)
			} else {
				fmt.Println("✓ Global CLAUDE.md section removed")
			}

			if err := report.ResetDigest(); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "  warning: reset digest: %v\n", err)
			} else {
				fmt.Println("✓ Digest cleared")
			}

			if err := uninstallZshCompletion(); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: zsh completion: %v\n", err)
			} else {
				fmt.Println("✓ Zsh completion removed from ~/.zshrc")
			}

			fmt.Println("\nRun `make uninstall` from the claude-lajan repo to also remove binaries and ~/.claude-lajan/.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&noConfirm, "yes", false, "Skip confirmation prompt")
	return cmd
}

// installZshCompletion appends a source line to ~/.zshrc if not already present.
func installZshCompletion() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	zshrc := filepath.Join(home, ".zshrc")
	line := "source <(lajan completion zsh)  # added by claude-lajan"

	data, _ := os.ReadFile(zshrc)
	if strings.Contains(string(data), "lajan completion zsh") {
		return nil // already present
	}

	f, err := os.OpenFile(zshrc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", line)
	if err == nil {
		fmt.Println("  Zsh completion added to ~/.zshrc (run `source ~/.zshrc` to activate)")
	}
	return err
}

// uninstallZshCompletion removes the lajan completion line from ~/.zshrc.
func uninstallZshCompletion() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	zshrc := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(zshrc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(data)
	if !strings.Contains(content, "lajan completion zsh") {
		return nil // nothing to remove
	}
	var kept []string
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, "lajan completion zsh") {
			kept = append(kept, line)
		}
	}
	// Trim trailing blank lines added when we inserted ours, but keep a final newline
	cleaned := strings.TrimRight(strings.Join(kept, "\n"), "\n") + "\n"
	return os.WriteFile(zshrc, []byte(cleaned), 0644)
}

// patchSettings adds the lajan hooks to ~/.claude/settings.json.
func patchSettings(binDir string) error {
	settingsPath := config.GlobalSettings()
	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var settings map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
		}
	} else {
		settings = map[string]any{}
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	stopHookCmd := filepath.Join(binDir, "stop-hook")
	hooks["Stop"] = appendHookIfMissing(hooks["Stop"], stopHookCmd)
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0644)
}

func appendHookIfMissing(existing any, command string) []any {
	var list []any
	if existing != nil {
		list, _ = existing.([]any)
	}

	// Check if hook already registered
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hooks, _ := m["hooks"].([]any)
		for _, h := range hooks {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if hm["command"] == command {
				return list // already present
			}
		}
	}

	newEntry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": command,
			},
		},
	}
	return append(list, newEntry)
}

// ── queue helpers ─────────────────────────────────────────────────────────────

func readQueue() ([]string, error) {
	f, err := os.Open(config.QueueFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var paths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, scanner.Err()
}

func writeQueue(paths []string) error {
	if len(paths) == 0 {
		err := os.Remove(config.QueueFile())
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	f, err := os.Create(config.QueueFile())
	if err != nil {
		return err
	}
	defer f.Close()
	for _, p := range paths {
		fmt.Fprintln(f, p)
	}
	return nil
}
