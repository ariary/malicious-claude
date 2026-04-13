package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// suggestedHookRecord tracks a hookify rule file written by lajan so it can be
// removed cleanly by RemoveSuggestedHooks / lajan reset --hooks.
type suggestedHookRecord struct {
	RulePath string `json:"rule_path"` // absolute path to the hookify .md file
}

func suggestedHooksFile() string {
	return filepath.Join(config.ReviewerDir(), "suggested-hooks.json")
}

func loadSuggestedHookRecords() []suggestedHookRecord {
	data, err := os.ReadFile(suggestedHooksFile())
	if err != nil {
		return nil
	}
	var records []suggestedHookRecord
	_ = json.Unmarshal(data, &records)
	return records
}

func saveSuggestedHookRecords(records []suggestedHookRecord) error {
	out, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(suggestedHooksFile(), append(out, '\n'), 0644)
}

// InjectSuggestedHooks writes each hook suggestion as a hookify rule file
// (.claude/hookify.{name}.local.md). Hookify handles registration automatically.
// All injected rules are tracked in suggested-hooks.json for clean removal.
func InjectSuggestedHooks(findings []debate.Finding, cwd string) ([]string, error) {
	var added []string
	records := loadSuggestedHookRecords()

	for _, f := range findings {
		if f.SuggestedHook == nil {
			continue
		}
		h := f.SuggestedHook

		var dir string
		if f.Scope == debate.ScopeProject {
			dir = filepath.Join(cwd, ".claude")
		} else {
			dir = config.ClaudeDir()
		}

		rulePath, err := writeHookifyRule(h, f.Text, dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [lajan] skipping hook suggestion: %v\n", err)
			continue
		}

		records = append(records, suggestedHookRecord{RulePath: rulePath})
		added = append(added, fmt.Sprintf("%s/%s → %s", h.Event, h.Action, filepath.Base(rulePath)))
	}

	if len(records) > 0 {
		_ = saveSuggestedHookRecords(records)
	}
	return added, nil
}

// writeHookifyRule validates the suggestion and writes a hookify rule file.
// Returns the path of the created file.
func writeHookifyRule(h *debate.HookSuggestion, description, dir string) (string, error) {
	validEvents := map[string]bool{"bash": true, "file": true, "prompt": true, "stop": true, "all": true}
	if !validEvents[h.Event] {
		return "", fmt.Errorf("invalid event %q (must be bash|file|prompt|stop)", h.Event)
	}
	if h.Action != "warn" && h.Action != "block" {
		return "", fmt.Errorf("invalid action %q (must be warn|block)", h.Action)
	}
	if _, err := regexp.Compile(h.Pattern); err != nil {
		return "", fmt.Errorf("invalid regex pattern: %v", err)
	}
	if strings.TrimSpace(h.Message) == "" {
		return "", fmt.Errorf("message is required")
	}

	slug := toSlug(description)
	if slug == "" {
		slug = toSlug(h.Pattern)
	}
	rulePath := filepath.Join(dir, fmt.Sprintf("hookify.%s.local.md", slug))

	// Already exists — overwrite with latest definition
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", slug)
	sb.WriteString("enabled: true\n")
	fmt.Fprintf(&sb, "event: %s\n", h.Event)
	fmt.Fprintf(&sb, "pattern: %s\n", h.Pattern)
	fmt.Fprintf(&sb, "action: %s\n", h.Action)
	if h.Field != "" {
		fmt.Fprintf(&sb, "field: %s\n", h.Field)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(h.Message + "\n")

	return rulePath, os.WriteFile(rulePath, []byte(sb.String()), 0644)
}

// toSlug converts a string to a lowercase hyphenated slug for use in filenames.
func toSlug(s string) string {
	var sb strings.Builder
	prevHyphen := true
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen && sb.Len() > 0 {
			sb.WriteRune('-')
			prevHyphen = true
		}
	}
	result := sb.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return strings.TrimRight(result, "-")
}

// RemoveSuggestedHooks deletes all hookify rule files written by lajan.
func RemoveSuggestedHooks() error {
	records := loadSuggestedHookRecords()
	if len(records) == 0 {
		return nil
	}
	for _, r := range records {
		_ = os.Remove(r.RulePath)
	}
	return os.Remove(suggestedHooksFile())
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

	// Parse existing findings and build a set of their plain texts for dedup.
	// parseBullets returns "**[type]** text" (without the leading "- ").
	existingLines := parseBullets(currentSection)
	existingTexts := make(map[string]bool, len(existingLines))
	for _, line := range existingLines {
		// Extract plain text from "**[type]** text"
		if idx := strings.Index(line, "** "); idx >= 0 {
			existingTexts[line[idx+3:]] = true
		}
	}

	// Build new section
	var sb strings.Builder
	sb.WriteString(prefix)
	if !strings.HasSuffix(prefix, "\n\n") {
		sb.WriteString("\n\n")
	}
	sb.WriteString(sectionStart + "\n")
	sb.WriteString(claudeMDSection)

	// Collect all lines: new findings first, then existing (newest→oldest order)
	var allLines []string
	for _, f := range newFindings {
		if !existingTexts[f.Text] {
			allLines = append(allLines, fmt.Sprintf("**[%s]** %s", f.Type, f.Text))
		}
	}
	for _, line := range existingLines {
		if strings.HasPrefix(line, "**[") {
			allLines = append(allLines, line)
		}
	}

	// Enforce rolling cap: keep only the most recent ClaudeMDMaxEntries entries
	if len(allLines) > config.ClaudeMDMaxEntries {
		allLines = allLines[:config.ClaudeMDMaxEntries]
	}

	for _, line := range allLines {
		fmt.Fprintf(&sb, "- %s\n", line)
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
// It also removes legacy hooks pointing to ~/.claude-reviewer/bin (old name).
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

	binDirs := config.AllBinDirs()
	for _, event := range []string{"Stop", "UserPromptSubmit", "PreToolUse"} {
		for _, dir := range binDirs {
			hooks[event] = removeReviewerHooks(hooks[event], dir)
		}
		// Remove the key entirely when the list is empty — avoids writing null to JSON.
		if list, _ := hooks[event].([]any); len(list) == 0 {
			delete(hooks, event)
		}
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
