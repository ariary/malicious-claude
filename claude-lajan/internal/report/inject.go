package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/debate"
)

const (
	claudeMDSection = "## Claude Session Learnings\n\n"
	sectionStart    = "<!-- claude-lajan:start -->"
	sectionEnd      = "<!-- claude-lajan:end -->"
)

// InjectFindings routes findings to the appropriate destinations:
// - scope:project → project memory file
// - scope:global  → ~/.claude/CLAUDE.md managed section
//
// If the "memory_inject" hook is disabled in user config, this function is a no-op.
func InjectFindings(findings []debate.Finding, projectCWD string) error {
	if !config.IsHookEnabled("memory_inject") {
		return nil
	}

	var projectFindings, globalFindings []debate.Finding
	for _, f := range findings {
		switch f.Scope {
		case debate.ScopeProject:
			projectFindings = append(projectFindings, f)
		case debate.ScopeGlobal:
			globalFindings = append(globalFindings, f)
		}
	}

	if len(projectFindings) > 0 {
		if err := injectProjectMemory(projectFindings, projectCWD); err != nil {
			return fmt.Errorf("project memory inject: %w", err)
		}
	}
	if len(globalFindings) > 0 {
		if err := injectGlobalCLAUDEmd(globalFindings); err != nil {
			return fmt.Errorf("global CLAUDE.md inject: %w", err)
		}
	}
	return nil
}

// injectProjectMemory writes project-scoped findings to the Claude Code project memory file.
func injectProjectMemory(findings []debate.Finding, cwd string) error {
	memFile := config.ProjectMemoryFile(cwd)
	if err := os.MkdirAll(filepath.Dir(memFile), 0755); err != nil {
		return err
	}

	existing, _ := os.ReadFile(memFile)
	content := buildMemoryFile(findings, string(existing))
	return os.WriteFile(memFile, []byte(content), 0644)
}

func buildMemoryFile(findings []debate.Finding, existing string) string {
	const header = "---\nname: Session Reviewer Learnings\ndescription: Learnings from adversarial session analysis — project-specific improvements\ntype: feedback\n---\n\n"

	var sb strings.Builder

	// Preserve existing content if it doesn't have our header
	if existing != "" && !strings.Contains(existing, "Session Reviewer Learnings") {
		sb.WriteString(existing)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(header)
	}

	sb.WriteString("## Project-Specific Learnings\n\n")
	for _, f := range findings {
		fmt.Fprintf(&sb, "- **[%s]** %s\n", f.Type, f.Text)
	}
	return sb.String()
}

// injectGlobalCLAUDEmd appends global findings to ~/.claude/CLAUDE.md under a managed section.
func injectGlobalCLAUDEmd(findings []debate.Finding) error {
	claudeMD := config.GlobalClaudeMD()
	existing, _ := os.ReadFile(claudeMD)
	content := mergeGlobalSection(string(existing), findings)
	return os.WriteFile(claudeMD, []byte(content), 0644)
}

func mergeGlobalSection(existing string, newFindings []debate.Finding) string {
	// Extract current managed section content
	start := strings.Index(existing, sectionStart)
	end := strings.Index(existing, sectionEnd)

	var prefix, currentSection string
	if start >= 0 && end >= 0 && end > start {
		prefix = existing[:start]
		currentSection = existing[start+len(sectionStart) : end]
	} else {
		prefix = existing
	}

	// Parse existing findings in the section
	existingFindings := parseBullets(currentSection)
	seen := map[string]bool{}
	for _, f := range existingFindings {
		seen[f] = true
	}

	// Build new section
	var sb strings.Builder
	sb.WriteString(prefix)
	if !strings.HasSuffix(prefix, "\n\n") {
		sb.WriteString("\n\n")
	}
	sb.WriteString(sectionStart + "\n")
	sb.WriteString(claudeMDSection)

	// Prepend new unique findings
	for _, f := range newFindings {
		if !seen[f.Text] {
			fmt.Fprintf(&sb, "- **[%s]** %s\n", f.Type, f.Text)
			seen[f.Text] = true
			existingFindings = append([]string{fmt.Sprintf("**[%s]** %s", f.Type, f.Text)}, existingFindings...)
		}
	}
	// Re-write existing
	for _, line := range existingFindings {
		if strings.HasPrefix(line, "**[") {
			// already formatted
			fmt.Fprintf(&sb, "- %s\n", line)
		}
	}
	sb.WriteString("\n" + sectionEnd + "\n")
	return sb.String()
}

func parseBullets(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			out = append(out, strings.TrimPrefix(line, "- "))
		}
	}
	return out
}

// ── Read helpers ──────────────────────────────────────────────────────────────

// ReadGlobalLearnings returns the bullet lines currently in the managed CLAUDE.md section.
func ReadGlobalLearnings() []string {
	data, err := os.ReadFile(config.GlobalClaudeMD())
	if err != nil {
		return nil
	}
	content := string(data)
	start := strings.Index(content, sectionStart)
	end := strings.Index(content, sectionEnd)
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	section := content[start+len(sectionStart) : end]
	return parseBullets(section)
}

// ReadProjectLearnings returns bullet lines from the project memory file for cwd.
func ReadProjectLearnings(cwd string) []string {
	data, err := os.ReadFile(config.ProjectMemoryFile(cwd))
	if err != nil {
		return nil
	}
	return parseBullets(string(data))
}

// ── Reset helpers ─────────────────────────────────────────────────────────────

// ResetGlobal removes the managed section from ~/.claude/CLAUDE.md.
func ResetGlobal() error {
	claudeMD := config.GlobalClaudeMD()
	data, err := os.ReadFile(claudeMD)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(data)
	start := strings.Index(content, sectionStart)
	end := strings.Index(content, sectionEnd)
	if start < 0 || end < 0 {
		return nil // nothing to remove
	}
	cleaned := strings.TrimRight(content[:start], "\n") + "\n"
	return os.WriteFile(claudeMD, []byte(cleaned), 0644)
}

// ResetProject removes the feedback_learnings.md memory file for a given project cwd.
func ResetProject(cwd string) error {
	memFile := config.ProjectMemoryFile(cwd)
	if err := os.Remove(memFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ResetDigest removes the rolling digest file.
func ResetDigest() error {
	if err := os.Remove(config.DigestFile()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// UninstallHooks removes the lajan hook entries from ~/.claude/settings.json.
func UninstallHooks() error {
	settingsPath := config.GlobalSettings()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}

	binDir := config.BinDir()
	for _, event := range []string{"Stop", "UserPromptSubmit"} {
		hooks[event] = removeReviewerHooks(hooks[event], binDir)
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0644)
}

func removeReviewerHooks(existing any, binDir string) []any {
	list, _ := existing.([]any)
	var filtered []any
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		hookList, _ := m["hooks"].([]any)
		isReviewer := false
		for _, h := range hookList {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if strings.HasPrefix(cmd, binDir) {
				isReviewer = true
				break
			}
		}
		if !isReviewer {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
