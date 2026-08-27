package launch

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type IntegrationSpec struct {
	Name           string
	DisplayName    string
	Description    string
	Aliases        []string
	CheckInstalled func() bool
	InstallURL     string
	InstallCmd     []string
}

var specs = []*IntegrationSpec{
	{Name: "claude", DisplayName: "Claude Code", Description: "Anthropic's coding tool with subagents", CheckInstalled: func() bool { _, err := exec.LookPath("claude"); return err == nil }, InstallURL: "https://code.claude.com/docs/en/quickstart"},
	{Name: "codex", DisplayName: "Codex", Description: "OpenAI's open-source coding agent", CheckInstalled: func() bool { _, err := exec.LookPath("codex"); return err == nil }, InstallURL: "https://developers.openai.com/codex/cli/", InstallCmd: []string{"npm", "install", "-g", "@openai/codex"}},
	{Name: "opencode", DisplayName: "OpenCode", Description: "Anomaly's open-source coding agent", CheckInstalled: func() bool { _, err := exec.LookPath("opencode"); return err == nil }, InstallURL: "https://opencode.ai"},
	{Name: "copilot", DisplayName: "Copilot CLI", Description: "GitHub's AI coding agent for the terminal", CheckInstalled: func() bool { _, err := exec.LookPath("copilot"); return err == nil }, InstallURL: "https://github.com/features/copilot/cli/"},
	{Name: "cline", DisplayName: "Cline", Description: "Autonomous coding agent with parallel execution", CheckInstalled: func() bool { _, err := exec.LookPath("cline"); return err == nil }, InstallCmd: []string{"npm", "install", "-g", "cline@latest"}},
	{Name: "droid", DisplayName: "Droid", Description: "Factory's coding agent across terminal and IDEs", CheckInstalled: func() bool { _, err := exec.LookPath("droid"); return err == nil }, InstallURL: "https://docs.factory.ai/cli/getting-started/quickstart"},
	{Name: "hermes", DisplayName: "Hermes Agent", Description: "Self-improving AI agent built by Nous Research", CheckInstalled: func() bool { _, err := exec.LookPath("hermes"); return err == nil }, InstallURL: "https://hermes-agent.nousresearch.com/docs/getting-started/installation/"},
	{Name: "hermes-desktop", DisplayName: "Hermes Desktop", Description: "Desktop app for Hermes Agent by Nous Research", CheckInstalled: func() bool { _, err := exec.LookPath("hermes"); return err == nil }, InstallURL: "https://hermes-agent.nousresearch.com/docs/getting-started/installation/"},
	{Name: "openclaw", DisplayName: "OpenClaw", Description: "Personal AI with 100+ skills", CheckInstalled: func() bool { _, err := exec.LookPath("openclaw"); return err == nil }, InstallURL: "https://docs.openclaw.ai"},
	{Name: "pi", DisplayName: "Pi", Description: "Minimal AI agent toolkit with plugin support", CheckInstalled: func() bool { _, err := exec.LookPath("pi"); return err == nil }, InstallCmd: []string{"npm", "install", "-g", "@earendil-works/pi-coding-agent@latest"}},
	{Name: "qwen", DisplayName: "Qwen Code", Description: "Qwen's AI coding agent with tool use", CheckInstalled: func() bool { _, err := exec.LookPath("qwen"); return err == nil }, InstallURL: "https://qwen.ai/qwencode"},
	{Name: "chatgpt", DisplayName: "ChatGPT", Description: "Complete work with ChatGPT", Aliases: []string{"codex-app", "codex-desktop", "codex-gui"}, CheckInstalled: checkCodexAppInstalled, InstallURL: "https://chatgpt.com/download"},
}

func checkCodexAppInstalled() bool {
	if _, err := os.Stat("/Applications/ChatGPT.app"); err == nil {
		return true
	}
	if _, err := exec.LookPath("chatgpt"); err == nil {
		return true
	}
	return false
}

var byName map[string]*IntegrationSpec

func init() {
	byName = make(map[string]*IntegrationSpec)
	for _, s := range specs {
		byName[strings.ToLower(s.Name)] = s
		for _, a := range s.Aliases {
			byName[strings.ToLower(a)] = s
		}
	}
}

func Lookup(name string) (*IntegrationSpec, error) {
	if s, ok := byName[strings.ToLower(name)]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("unknown integration: %s", name)
}

type IntegrationInfo struct {
	Name        string `json:"id"`
	DisplayName string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	Command     string `json:"command,omitempty"`
}

func ListInfos() []IntegrationInfo {
	order := []string{"claude", "codex", "openclaw", "opencode", "hermes", "hermes-desktop", "droid", "pi", "cline", "qwen", "copilot"}
	seen := map[string]bool{}
	var out []IntegrationInfo
	for _, n := range order {
		s, ok := byName[n]
		if !ok {
			continue
		}
		seen[s.Name] = true
		installed := false
		if s.CheckInstalled != nil {
			installed = s.CheckInstalled()
		}
		out = append(out, IntegrationInfo{Name: s.Name, DisplayName: s.DisplayName, Description: s.Description, Installed: installed, Command: "launchpad launch " + s.Name})
	}
	for _, s := range specs {
		if seen[s.Name] {
			continue
		}
		installed := false
		if s.CheckInstalled != nil {
			installed = s.CheckInstalled()
		}
		out = append(out, IntegrationInfo{Name: s.Name, DisplayName: s.DisplayName, Description: s.Description, Installed: installed, Command: "launchpad launch " + s.Name})
	}
	return out
}

func IsInstalled(name string) bool {
	s, err := Lookup(name)
	if err != nil {
		return false
	}
	if s.CheckInstalled == nil {
		return true
	}
	return s.CheckInstalled()
}
