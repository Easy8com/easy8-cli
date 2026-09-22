package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"easy8-cli/internal/api"
	"easy8-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func TestRunNoArgs(t *testing.T) {
	setTestHome(t)

	code := Run([]string{})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func TestVersionCommand(t *testing.T) {
	setTestHome(t)

	stdout, _, code := captureRun(t, []string{"version"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout, "0.1.9") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	setTestHome(t)

	code := Run([]string{"nope"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func TestSkillCommandPrintsEmbedded(t *testing.T) {
	setTestHome(t)

	stdout, stderr, code := captureRun(t, []string{"skill"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "# Easy8 CLI") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestSkillCommandIncludesInstallGuidance(t *testing.T) {
	setTestHome(t)

	stdout, stderr, code := captureRun(t, []string{"skill"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "curl -fsSL https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.sh | bash") {
		t.Fatalf("expected linux install guidance in skill output")
	}
	if !strings.Contains(stdout, "irm https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.ps1 | iex") {
		t.Fatalf("expected windows install guidance in skill output")
	}
	if strings.Contains(stdout, "go run ./cmd/easy8") {
		t.Fatalf("did not expect go run fallback in skill output")
	}
}

func TestSkillInstallLocalOpenCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	setWorkingDir(t, project)

	stdout, stderr, code := captureRun(t, []string{"skill", "install", "--target", "opencode", "--local"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Skill installed") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	path := filepath.Join(project, ".opencode", "skills", "easy8-cli", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if !strings.Contains(string(data), "name: easy8-cli") {
		t.Fatalf("unexpected skill content")
	}
}

func TestSkillListQuiet(t *testing.T) {
	setTestHome(t)

	stdout, stderr, code := captureRun(t, []string{"skill", "list", "--quiet"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}

	var list []skillInfo
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("json error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 bundled skill, got %+v", list)
	}
	if list[0].Name != "easy8-cli" {
		t.Fatalf("unexpected skill list: %+v", list)
	}
}

func TestSkillSyncGlobalOpenCode(t *testing.T) {
	setTestHome(t)
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	stdout, stderr, code := captureRun(t, []string{"skill", "sync", "--target", "opencode"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Synced 1 skills") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	for _, name := range []string{"easy8-cli"} {
		path := filepath.Join(configRoot, "opencode", "skills", name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read synced skill %s: %v", name, err)
		}
		if !strings.Contains(string(data), "name: "+name) {
			t.Fatalf("unexpected content for %s", name)
		}
	}
}

func TestSkillSyncLocalOpenCode(t *testing.T) {
	setTestHome(t)
	project := t.TempDir()
	setWorkingDir(t, project)

	stdout, stderr, code := captureRun(t, []string{"skill", "sync", "--target", "opencode", "--local"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Synced 1 skills") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	path := filepath.Join(project, ".opencode", "skills", "easy8-cli", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read synced skill: %v", err)
	}
	if !strings.Contains(string(data), "name: easy8-cli") {
		t.Fatalf("unexpected skill content")
	}
}

func TestSkillSyncDryRunDoesNotWrite(t *testing.T) {
	setTestHome(t)
	project := t.TempDir()
	setWorkingDir(t, project)

	stdout, stderr, code := captureRun(t, []string{"skill", "sync", "--target", "opencode", "--local", "--dry-run"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Would sync 1 skills") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	path := filepath.Join(project, ".opencode", "skills", "easy8-cli", "SKILL.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run not to write %s, err=%v", path, err)
	}
}

func TestCommandsJSONOutput(t *testing.T) {
	setTestHome(t)

	stdout, stderr, code := captureRun(t, []string{"commands", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var cmds []commandInfo
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &cmds); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if len(cmds) == 0 {
		t.Fatalf("expected command catalog")
	}
	foundIssue := false
	for _, cmd := range cmds {
		if cmd.Name == "easy8 issue" {
			foundIssue = true
			break
		}
	}
	if !foundIssue {
		t.Fatalf("expected issue command in catalog")
	}
}

func TestAuthLoginStatusLogoutFlow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")

	stdout, stderr, code := captureRun(t, []string{"auth", "status", "--quiet"})
	if code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, stderr)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status json error: %v", err)
	}
	if status["authenticated"] != false {
		t.Fatalf("expected unauthenticated status: %+v", status)
	}

	_, stderr, code = captureRun(t, []string{"auth", "login", "--api-key", "secret-123", "--global", "--quiet"})
	if code != 0 {
		t.Fatalf("login code=%d stderr=%s", code, stderr)
	}

	stdout, stderr, code = captureRun(t, []string{"auth", "status", "--quiet"})
	if code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status json error: %v", err)
	}
	if status["authenticated"] != true {
		t.Fatalf("expected authenticated status: %+v", status)
	}

	_, stderr, code = captureRun(t, []string{"auth", "logout", "--global", "--quiet"})
	if code != 0 {
		t.Fatalf("logout code=%d stderr=%s", code, stderr)
	}

	stdout, stderr, code = captureRun(t, []string{"auth", "status", "--quiet"})
	if code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status json error: %v", err)
	}
	if status["authenticated"] != false {
		t.Fatalf("expected unauthenticated status after logout: %+v", status)
	}
}

func TestSetupNonInteractiveGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")

	stdout, stderr, code := captureRun(t, []string{"setup", "--non-interactive", "--global", "--base-url", "https://example.com", "--api-key", "abc", "--project-id", "10", "--autoupdate"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Configuration saved to") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	path := filepath.Join(home, ".config", "easy8", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if saved.BaseURL != "https://example.com" {
		t.Fatalf("base_url = %q", saved.BaseURL)
	}
	if saved.APIKey != "abc" {
		t.Fatalf("api_key = %q", saved.APIKey)
	}
	if saved.Defaults.ProjectID != 10 {
		t.Fatalf("project_id = %d", saved.Defaults.ProjectID)
	}
	if !saved.AutoUpdate {
		t.Fatalf("autoupdate = false")
	}
}

func TestSetupNonInteractiveLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")
	t.Setenv("EASY8_AUTOUPDATE", "true")

	project := t.TempDir()
	setWorkingDir(t, project)

	stdout, stderr, code := captureRun(t, []string{"setup", "--non-interactive", "--local", "--base-url", "https://local.example.com", "--api-key", "local-key", "--autoupdate=false"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Configuration saved to") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}

	path := filepath.Join(project, ".easy8.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if saved.BaseURL != "https://local.example.com" {
		t.Fatalf("base_url = %q", saved.BaseURL)
	}
	if saved.APIKey != "local-key" {
		t.Fatalf("api_key = %q", saved.APIKey)
	}
	if saved.AutoUpdate {
		t.Fatalf("autoupdate = true")
	}
}

func TestSetupNonInteractiveDefaultsAutoupdateTrue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")
	t.Setenv("EASY8_AUTOUPDATE", "")

	_, stderr, code := captureRun(t, []string{"setup", "--non-interactive", "--global", "--base-url", "https://example.com", "--api-key", "abc"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}

	path := filepath.Join(home, ".config", "easy8", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !saved.AutoUpdate {
		t.Fatalf("autoupdate = false")
	}
}

func TestSetupInteractivePromptsAutoupdateDefaultYes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")
	t.Setenv("EASY8_AUTOUPDATE", "")

	input := "\nhttps://example.com\nabc\n\n"
	stdout, stderr, code := captureRunWithInput(t, []string{"setup"}, input)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s stdout=%s", code, stderr, stdout)
	}
	if !strings.Contains(stdout, "Enable automatic daily updates? [Y/n]") {
		t.Fatalf("expected autoupdate prompt, stdout=%s", stdout)
	}

	path := filepath.Join(home, ".config", "easy8", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !saved.AutoUpdate {
		t.Fatalf("autoupdate = false")
	}
}

func TestSetupInteractiveAutoupdateNo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EASY8_BASE_URL", "")
	t.Setenv("EASY8_API_KEY", "")
	t.Setenv("EASY8_AUTOUPDATE", "")

	project := t.TempDir()
	setWorkingDir(t, project)

	input := "https://example.com\nabc\nno\n"
	stdout, stderr, code := captureRunWithInput(t, []string{"setup", "--local"}, input)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s stdout=%s", code, stderr, stdout)
	}

	path := filepath.Join(project, ".easy8.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if saved.AutoUpdate {
		t.Fatalf("autoupdate = true")
	}
}

func TestSetupNonInteractiveValidation(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"setup", "--non-interactive", "--base-url", "https://example.com"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "--api-key is required in --non-interactive mode") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestSetupTargetConflict(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"setup", "--non-interactive", "--global", "--local", "--base-url", "https://example.com", "--api-key", "abc"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "--global and --local cannot be used together") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestPromptScopeDefaultsToGlobal(t *testing.T) {
	output := &bytes.Buffer{}
	scope, err := promptScope(strings.NewReader("\n"), output)
	if err != nil {
		t.Fatalf("promptScope error: %v", err)
	}
	if scope != "global" {
		t.Fatalf("scope = %q", scope)
	}
}

func TestPromptScopeAcceptsLocal(t *testing.T) {
	output := &bytes.Buffer{}
	scope, err := promptScope(strings.NewReader("local\n"), output)
	if err != nil {
		t.Fatalf("promptScope error: %v", err)
	}
	if scope != "local" {
		t.Fatalf("scope = %q", scope)
	}
}

func TestPromptScopeRejectsInvalid(t *testing.T) {
	output := &bytes.Buffer{}
	_, err := promptScope(strings.NewReader("xxx\n"), output)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPromptBoolDefaultsToYes(t *testing.T) {
	output := &bytes.Buffer{}
	value, err := promptBool(strings.NewReader("\n"), output, "Enable automatic daily updates", true)
	if err != nil {
		t.Fatalf("promptBool error: %v", err)
	}
	if !value {
		t.Fatalf("value = false")
	}
	if !strings.Contains(output.String(), "[Y/n]") {
		t.Fatalf("unexpected prompt: %s", output.String())
	}
}

func TestPromptBoolAcceptsNo(t *testing.T) {
	output := &bytes.Buffer{}
	value, err := promptBool(strings.NewReader("no\n"), output, "Enable automatic daily updates", true)
	if err != nil {
		t.Fatalf("promptBool error: %v", err)
	}
	if value {
		t.Fatalf("value = true")
	}
}

func TestPromptBoolRejectsInvalid(t *testing.T) {
	output := &bytes.Buffer{}
	_, err := promptBool(strings.NewReader("maybe\n"), output, "Enable automatic daily updates", true)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueCreateDoneRatioInvalid(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	args := []string{"issue", "create", "--subject", "Test", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2", "--done-ratio", "150"}
	_, stderr, code := captureRun(t, args)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "done-ratio must be between 0 and 100") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueUpdateDoneRatioInvalid(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	_, stderr, code := captureRun(t, []string{"issue", "update", "--id", "101", "--done-ratio", "-5"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "done-ratio must be between 0 and 100") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueCreateMissingSubject(t *testing.T) {
	setTestHome(t)

	code := Run([]string{"issue", "create", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func TestIssueCreateParentIDInvalid(t *testing.T) {
	setTestHome(t)

	args := []string{"issue", "create", "--subject", "Test", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2", "--parent-id", "0"}
	_, stderr, code := captureRun(t, args)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "parent-id must be greater than 0") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueUpdateParentIDInvalid(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"issue", "update", "101", "--parent-id", "0"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "parent-id must be greater than 0") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueSearchMissingQuery(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"issue", "search"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "at least one filter is required") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueListTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "list"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Subject") || !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestIssueListJSONOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "list", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueListResponse
	env := decodeEnvelope(t, stdout)
	if !env.OK {
		t.Fatalf("expected ok envelope")
	}
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if len(resp.Issues) != 1 || resp.Issues[0].ID != 101 {
		t.Fatalf("unexpected issues: %+v", resp.Issues)
	}
}

func TestIssueListQuietOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "list", "--quiet"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueListResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("quiet json error: %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("unexpected issues: %+v", resp.Issues)
	}
}

func TestIssueListJSONQuietConflict(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	_, stderr, code := captureRun(t, []string{"issue", "list", "--json", "--quiet"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "--json and --quiet cannot be used together") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueCreateJSONOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	args := []string{"issue", "create", "--subject", "New task", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2", "--json"}
	stdout, stderr, code := captureRun(t, args)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if resp.Issue.ID != 202 {
		t.Fatalf("unexpected issue id: %d", resp.Issue.ID)
	}
}

func TestIssueCreateOptionalDefaults(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		defaults config.Defaults
		env      map[string]string
		want     map[string]string
	}{
		{name: "omitted fields use server defaults"},
		{
			name: "explicit status only",
			args: []string{"--status-id", "2"},
			want: map[string]string{"status_id": "2"},
		},
		{
			name: "explicit priority only",
			args: []string{"--priority-id", "3"},
			want: map[string]string{"priority_id": "3"},
		},
		{
			name: "explicit author only",
			args: []string{"--author-id", "4"},
			want: map[string]string{"author_id": "4"},
		},
		{
			name: "explicit fields",
			args: []string{"--status-id", "2", "--priority-id", "3", "--author-id", "4"},
			want: map[string]string{"status_id": "2", "priority_id": "3", "author_id": "4"},
		},
		{
			name:     "config defaults",
			defaults: config.Defaults{StatusID: 5, PriorityID: 6, AuthorID: 7},
			want:     map[string]string{"status_id": "5", "priority_id": "6", "author_id": "7"},
		},
		{
			name:     "environment overrides config",
			defaults: config.Defaults{StatusID: 5, PriorityID: 6, AuthorID: 7},
			env: map[string]string{
				"EASY8_DEFAULT_STATUS_ID":   "8",
				"EASY8_DEFAULT_PRIORITY_ID": "9",
				"EASY8_DEFAULT_AUTHOR_ID":   "10",
			},
			want: map[string]string{"status_id": "8", "priority_id": "9", "author_id": "10"},
		},
		{
			name:     "flags override config and environment",
			args:     []string{"--status-id", "2", "--priority-id", "3", "--author-id", "4"},
			defaults: config.Defaults{StatusID: 5, PriorityID: 6, AuthorID: 7},
			env: map[string]string{
				"EASY8_DEFAULT_STATUS_ID":   "8",
				"EASY8_DEFAULT_PRIORITY_ID": "9",
				"EASY8_DEFAULT_AUTHOR_ID":   "10",
			},
			want: map[string]string{"status_id": "2", "priority_id": "3", "author_id": "4"},
		},
		{
			name:     "zero flags omit configured fields",
			args:     []string{"--status-id", "0", "--priority-id", "0", "--author-id", "0"},
			defaults: config.Defaults{StatusID: 5, PriorityID: 6, AuthorID: 7},
		},
		{
			name:     "zero environment omits configured fields",
			defaults: config.Defaults{StatusID: 5, PriorityID: 6, AuthorID: 7},
			env: map[string]string{
				"EASY8_DEFAULT_STATUS_ID":   "0",
				"EASY8_DEFAULT_PRIORITY_ID": "0",
				"EASY8_DEFAULT_AUTHOR_ID":   "0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := make(chan map[string]json.RawMessage, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/issues.json" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var request struct {
					Issue map[string]json.RawMessage `json:"issue"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- request.Issue
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"issue":{"id":202,"subject":"Update net-smtp gem"}}`))
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			setWorkingDir(t, t.TempDir())
			t.Setenv("EASY8_AUTOUPDATE", "false")
			for _, name := range []string{"EASY8_DEFAULT_STATUS_ID", "EASY8_DEFAULT_PRIORITY_ID", "EASY8_DEFAULT_AUTHOR_ID"} {
				t.Setenv(name, "")
			}
			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			if _, err := config.SaveGlobal(config.Config{Defaults: tt.defaults}); err != nil {
				t.Fatalf("save config: %v", err)
			}

			args := []string{"issue", "create", "--subject", "Update net-smtp gem", "--project-id", "35", "--tracker-id", "8", "--assigned-to-id", "6", "--quiet"}
			stdout, stderr, code := captureRun(t, append(args, tt.args...))
			if code != 0 {
				t.Fatalf("code = %d stderr=%s", code, stderr)
			}
			var response api.IssueResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil || response.Issue.ID != 202 {
				t.Fatalf("unexpected response: %s (error: %v)", stdout, err)
			}
			select {
			case issue := <-requests:
				for name, want := range map[string]string{"subject": `"Update net-smtp gem"`, "project_id": "35", "tracker_id": "8", "assigned_to_id": "6"} {
					if got := string(issue[name]); got != want {
						t.Errorf("%s = %s, want %s", name, got, want)
					}
				}
				for _, name := range []string{"status_id", "priority_id", "author_id"} {
					got, present := issue[name]
					want, expected := tt.want[name]
					if present != expected || (expected && string(got) != want) {
						t.Errorf("%s = %s (present=%t), want %s (present=%t)", name, got, present, want, expected)
					}
				}
			default:
				t.Fatal("expected create request")
			}
		})
	}
}

func TestIssueCreateServerValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":["Status cannot be blank"]}`))
	}))
	defer server.Close()
	setTestEnv(t, server.URL)
	setWorkingDir(t, t.TempDir())
	t.Setenv("EASY8_AUTOUPDATE", "false")
	for _, name := range []string{"EASY8_DEFAULT_STATUS_ID", "EASY8_DEFAULT_PRIORITY_ID", "EASY8_DEFAULT_AUTHOR_ID"} {
		t.Setenv(name, "")
	}

	_, stderr, code := captureRun(t, []string{"issue", "create", "--subject", "Update net-smtp gem", "--project-id", "35", "--tracker-id", "8", "--assigned-to-id", "6"})
	if code != 1 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stderr, "422") || !strings.Contains(stderr, "Status cannot be blank") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueCreateWithParentSendsParentIssueID(t *testing.T) {
	var seen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/issues.json" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		seen = true
		var request api.IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if request.Issue.ParentIssueID == nil || *request.Issue.ParentIssueID != 100 {
			t.Fatalf("parent_issue_id = %v", request.Issue.ParentIssueID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issue":{"id":202,"subject":"New task","parent":{"id":100,"name":"Parent task"}}}`))
	}))
	defer server.Close()
	setTestEnv(t, server.URL)

	args := []string{"issue", "create", "--subject", "New task", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2", "--parent-id", "100", "--quiet"}
	stdout, stderr, code := captureRun(t, args)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !seen {
		t.Fatal("expected create request")
	}
	var resp api.IssueResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("quiet json error: %v", err)
	}
	if resp.Issue.Parent == nil || resp.Issue.Parent.ID != 100 {
		t.Fatalf("parent = %+v", resp.Issue.Parent)
	}
}

func TestIssueCreateWithMultipleAttachmentsAndOptionalDescriptions(t *testing.T) {
	firstPath := mustWriteTestFile(t, "spec.pdf", "spec-content")
	secondPath := mustWriteTestFile(t, "build.log", "build output")
	expectedUploads := []struct {
		Filename    string
		Description string
		Body        string
		Token       string
	}{
		{Filename: "spec.pdf", Description: "Specification", Body: "spec-content", Token: "token-1"},
		{Filename: "build.log", Description: "", Body: "build output", Token: "token-2"},
	}

	uploadCalls := 0
	createSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/uploads.json":
			if r.Method != http.MethodPost {
				t.Fatalf("upload method = %s", r.Method)
			}
			if uploadCalls >= len(expectedUploads) {
				t.Fatalf("unexpected extra upload call")
			}
			expected := expectedUploads[uploadCalls]
			if r.URL.Query().Get("filename") != expected.Filename {
				t.Fatalf("upload filename = %s", r.URL.Query().Get("filename"))
			}
			if r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Fatalf("upload content-type = %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			if string(body) != expected.Body {
				t.Fatalf("upload body = %q", string(body))
			}
			uploadCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"upload":{"token":"` + expected.Token + `"}}`))
		case r.URL.Path == "/issues.json":
			if r.Method != http.MethodPost {
				t.Fatalf("create method = %s", r.Method)
			}
			createSeen = true
			var request api.IssueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(request.Issue.Uploads) != len(expectedUploads) {
				t.Fatalf("uploads count = %d", len(request.Issue.Uploads))
			}
			for i, upload := range request.Issue.Uploads {
				expected := expectedUploads[i]
				if upload.Token != expected.Token {
					t.Fatalf("upload token[%d] = %q", i, upload.Token)
				}
				if upload.Filename != expected.Filename {
					t.Fatalf("upload filename[%d] = %q", i, upload.Filename)
				}
				if upload.Description != expected.Description {
					t.Fatalf("upload description[%d] = %q", i, upload.Description)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issue":{"id":202,"subject":"New task"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setTestEnv(t, server.URL)

	args := []string{
		"issue", "create",
		"--subject", "New task",
		"--project-id", "1",
		"--tracker-id", "1",
		"--status-id", "1",
		"--priority-id", "1",
		"--author-id", "1",
		"--assigned-to-id", "2",
		"--attachment", firstPath,
		"--attachment-description", "Specification",
		"--attachment", secondPath,
	}
	stdout, stderr, code := captureRun(t, args)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "New task") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if uploadCalls != len(expectedUploads) {
		t.Fatalf("upload calls = %d", uploadCalls)
	}
	if !createSeen {
		t.Fatal("expected create request")
	}
}

func TestIssueUpdateTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "update", "--id", "101", "--status-id", "2"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestIssueUpdateWithParentSendsParentIssueID(t *testing.T) {
	var seen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/issues/101.json" || r.Method != http.MethodPut {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		seen = true
		var request api.IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if request.Issue.ParentIssueID == nil || *request.Issue.ParentIssueID != 100 {
			t.Fatalf("parent_issue_id = %v", request.Issue.ParentIssueID)
		}
		if request.Issue.AutomationSource == nil || *request.Issue.AutomationSource != api.AutomationSourceEasy8CLI {
			t.Fatalf("automation_source = %v", request.Issue.AutomationSource)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issue":{"id":101,"subject":"Fix onboarding","parent":{"id":100,"name":"Parent task"}}}`))
	}))
	defer server.Close()
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "update", "101", "--parent-id", "100", "--quiet"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !seen {
		t.Fatal("expected update request")
	}
	var resp api.IssueResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("quiet json error: %v", err)
	}
	if resp.Issue.Parent == nil || resp.Issue.Parent.ID != 100 {
		t.Fatalf("parent = %+v", resp.Issue.Parent)
	}
}

func TestIssueUpdateWithNotesRefetch(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "update", "--id", "101", "--notes", "a test note"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	// After PUT (empty body), the CLI re-fetches via GET and displays the issue.
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("expected re-fetched issue in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "101") {
		t.Fatalf("expected issue ID in stdout: %s", stdout)
	}
}

func TestIssueUpdateWithPositionalID(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "update", "101", "--status-id", "2"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestIssueUpdateWithAttachmentOnlyNoNotes(t *testing.T) {
	attachmentPath := mustWriteTestFile(t, "error.log", "runtime failure")
	uploadSeen := false
	putSeen := false
	getSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/uploads.json":
			uploadSeen = true
			if r.Method != http.MethodPost {
				t.Fatalf("upload method = %s", r.Method)
			}
			if r.URL.Query().Get("filename") != "error.log" {
				t.Fatalf("upload filename = %s", r.URL.Query().Get("filename"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			if string(body) != "runtime failure" {
				t.Fatalf("upload body = %q", string(body))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"upload":{"token":"upload-token"}}`))
		case r.URL.Path == "/issues/101.json" && r.Method == http.MethodPut:
			putSeen = true
			var request api.IssueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if request.Issue.Notes != nil {
				t.Fatalf("expected notes to be omitted, got %q", *request.Issue.Notes)
			}
			if request.Issue.AutomationSource == nil || *request.Issue.AutomationSource != api.AutomationSourceEasy8CLI {
				t.Fatalf("automation_source = %v", request.Issue.AutomationSource)
			}
			if len(request.Issue.Uploads) != 1 {
				t.Fatalf("uploads count = %d", len(request.Issue.Uploads))
			}
			upload := request.Issue.Uploads[0]
			if upload.Token != "upload-token" {
				t.Fatalf("upload token = %q", upload.Token)
			}
			if upload.Filename != "error.log" {
				t.Fatalf("upload filename = %q", upload.Filename)
			}
			if upload.Description != "" {
				t.Fatalf("upload description = %q", upload.Description)
			}
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/issues/101.json" && r.Method == http.MethodGet:
			getSeen = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issue":{"id":101,"subject":"Fix onboarding","status":{"id":1,"name":"New"},"assigned_to":{"id":2,"name":"Alice"},"updated_on":"2024-01-15"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "update", "--id", "101", "--attachment", attachmentPath})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !uploadSeen || !putSeen || !getSeen {
		t.Fatalf("expected upload=%v put=%v get=%v", uploadSeen, putSeen, getSeen)
	}
}

func TestIssueUpdateWithAttachmentDescription(t *testing.T) {
	attachmentPath := mustWriteTestFile(t, "screenshot.png", "png-binary")
	putSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/uploads.json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"upload":{"token":"upload-token"}}`))
		case r.URL.Path == "/issues/101.json" && r.Method == http.MethodPut:
			putSeen = true
			var request api.IssueRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(request.Issue.Uploads) != 1 {
				t.Fatalf("uploads count = %d", len(request.Issue.Uploads))
			}
			if request.Issue.Uploads[0].Description != "Failure screenshot" {
				t.Fatalf("upload description = %q", request.Issue.Uploads[0].Description)
			}
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/issues/101.json" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issue":{"id":101,"subject":"Fix onboarding","status":{"id":1,"name":"New"},"assigned_to":{"id":2,"name":"Alice"},"updated_on":"2024-01-15"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setTestEnv(t, server.URL)

	_, stderr, code := captureRun(t, []string{"issue", "update", "--id", "101", "--attachment", attachmentPath, "--attachment-description", "Failure screenshot"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !putSeen {
		t.Fatal("expected update request")
	}
}

func TestIssueAttachmentDescriptionWithoutAttachment(t *testing.T) {
	setTestHome(t)

	args := []string{
		"issue", "create",
		"--subject", "New task",
		"--project-id", "1",
		"--tracker-id", "1",
		"--status-id", "1",
		"--priority-id", "1",
		"--author-id", "1",
		"--assigned-to-id", "2",
		"--attachment-description", "Specification",
	}
	_, stderr, code := captureRun(t, args)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "requires a preceding --attachment") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueAttachmentMissingFile(t *testing.T) {
	setTestHome(t)

	args := []string{
		"issue", "create",
		"--subject", "New task",
		"--project-id", "1",
		"--tracker-id", "1",
		"--status-id", "1",
		"--priority-id", "1",
		"--author-id", "1",
		"--assigned-to-id", "2",
		"--attachment", filepath.Join(t.TempDir(), "missing.txt"),
	}
	_, stderr, code := captureRun(t, args)
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "attachment") || !strings.Contains(stderr, "no such file or directory") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueAttachmentDuplicateDescription(t *testing.T) {
	setTestHome(t)

	args := []string{
		"issue", "create",
		"--subject", "New task",
		"--project-id", "1",
		"--tracker-id", "1",
		"--status-id", "1",
		"--priority-id", "1",
		"--author-id", "1",
		"--assigned-to-id", "2",
		"--attachment", "spec.pdf",
		"--attachment-description", "First",
		"--attachment-description", "Second",
	}
	_, stderr, code := captureRun(t, args)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "already set") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueShowTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "show", "--id", "101"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("expected subject in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Alice") {
		t.Fatalf("expected assignee in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Parent task") {
		t.Fatalf("expected parent in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "30%") {
		t.Fatalf("expected done ratio in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Onboarding flow needs fixing") {
		t.Fatalf("expected description in stdout: %s", stdout)
	}
}

func TestIssueShowWithPositionalID(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "show", "101", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if resp.Issue.ID != 101 {
		t.Fatalf("issue id = %d", resp.Issue.ID)
	}
}

func TestIssueShowJSONOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "show", "--id", "101", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if resp.Issue.ID != 101 {
		t.Fatalf("issue id = %d", resp.Issue.ID)
	}
	if resp.Issue.Description != "Onboarding flow needs fixing" {
		t.Fatalf("description = %q", resp.Issue.Description)
	}
}

func TestIssueShowMissingID(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"issue", "show"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "id is required (use '<id>' or --id)") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueShowPositionalAndFlagIDConflict(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"issue", "show", "101", "--id", "102"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "positional id 101 does not match --id 102") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueShowWithJournalsAndAttachments(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "show", "--id", "101", "--include", "journals,attachments"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	// Basic fields still present
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("expected subject in stdout: %s", stdout)
	}
	// Journals section
	if !strings.Contains(stdout, "Journals:") {
		t.Fatalf("expected Journals section in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Bob") {
		t.Fatalf("expected journal author Bob in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Started work on this") {
		t.Fatalf("expected journal notes in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "status_id") {
		t.Fatalf("expected journal detail name in stdout: %s", stdout)
	}
	// Attachments section
	if !strings.Contains(stdout, "Attachments:") {
		t.Fatalf("expected Attachments section in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "design.pdf") {
		t.Fatalf("expected attachment filename in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Design doc") {
		t.Fatalf("expected attachment description in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "200.0 KB") {
		t.Fatalf("expected attachment filesize in stdout: %s", stdout)
	}
}

func TestIssueShowWithJournalsJSON(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "show", "--id", "101", "--include", "journals,attachments", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.IssueResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if len(resp.Issue.Journals) != 2 {
		t.Fatalf("journals count = %d", len(resp.Issue.Journals))
	}
	if resp.Issue.Journals[0].Notes != "Started work on this" {
		t.Fatalf("journal notes = %q", resp.Issue.Journals[0].Notes)
	}
	if len(resp.Issue.Journals[1].Details) != 1 {
		t.Fatalf("journal details count = %d", len(resp.Issue.Journals[1].Details))
	}
	if len(resp.Issue.Attachments) != 1 {
		t.Fatalf("attachments count = %d", len(resp.Issue.Attachments))
	}
	if resp.Issue.Attachments[0].Filename != "design.pdf" {
		t.Fatalf("attachment filename = %q", resp.Issue.Attachments[0].Filename)
	}
}

func TestIssueSearchTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "search", "--q", "onboarding"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestIssueListAPIError(t *testing.T) {
	server := newErrorServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"issue", "list"})
	if code != 1 {
		t.Fatalf("code = %d stdout=%s", code, stdout)
	}
	if !strings.Contains(stderr, "api error 500") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestIssueSearchNameFilters(t *testing.T) {
	server := newLookupServer(t)
	setTestEnv(t, server.URL)

	args := []string{
		"issue", "search", "--q", "petr",
		"--assignee", "Alice Doe",
		"--status", "New",
		"--priority", "High",
		"--task-type", "Task",
		"--project", "Project A",
	}
	stdout, stderr, code := captureRun(t, args)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Fix onboarding") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestIssueSearchNameConflict(t *testing.T) {
	server := newLookupServer(t)
	setTestEnv(t, server.URL)

	args := []string{"issue", "search", "--q", "petr", "--status-id", "1", "--status", "New"}
	_, stderr, code := captureRun(t, args)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "status-id does not match status name") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

// --- PBI tests ---

func TestPBIListTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "list", "--limit", "5"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "PBI Alpha") {
		t.Fatalf("expected name in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "to_do") {
		t.Fatalf("expected status in stdout: %s", stdout)
	}
}

func TestPBIListJSONOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "list", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.PBIListResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if len(resp.PBIs) != 1 || resp.PBIs[0].ID != 1 {
		t.Fatalf("unexpected: %+v", resp)
	}
}

func TestPBIShowTableOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "show", "--id", "1"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "PBI Alpha") {
		t.Fatalf("expected name: %s", stdout)
	}
	if !strings.Contains(stdout, "Design") {
		t.Fatalf("expected board: %s", stdout)
	}
	if !strings.Contains(stdout, "Full description") {
		t.Fatalf("expected description: %s", stdout)
	}
	// Issue details should include status, assignee, done ratio (fetched via batch)
	if !strings.Contains(stdout, "Task 100") {
		t.Fatalf("expected issue subject: %s", stdout)
	}
	if !strings.Contains(stdout, "New") {
		t.Fatalf("expected issue status: %s", stdout)
	}
	if !strings.Contains(stdout, "Alice") {
		t.Fatalf("expected issue assignee: %s", stdout)
	}
	if !strings.Contains(stdout, "50%") {
		t.Fatalf("expected issue done ratio: %s", stdout)
	}
	if !strings.Contains(stdout, "Review") {
		t.Fatalf("expected sticky note: %s", stdout)
	}
}

func TestPBIShowWithPositionalID(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "show", "1", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.PBIResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if resp.PBI.ID != 1 {
		t.Fatalf("id = %d", resp.PBI.ID)
	}
}

func TestPBIShowJSONOutput(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "show", "--id", "1", "--json"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	var resp api.PBIResponse
	env := decodeEnvelope(t, stdout)
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		t.Fatalf("json data error: %v", err)
	}
	if resp.PBI.ID != 1 {
		t.Fatalf("id = %d", resp.PBI.ID)
	}
}

func TestPBIShowMissingID(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"pbi", "show"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "id is required (use '<id>' or --id)") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestPBIUpdateSuccess(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "update", "--id", "1", "--status", "done"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "PBI #1 updated") {
		t.Fatalf("expected update message: %s", stdout)
	}
}

func TestPBIUpdateWithPositionalID(t *testing.T) {
	server := newTestServer(t)
	setTestEnv(t, server.URL)

	stdout, stderr, code := captureRun(t, []string{"pbi", "update", "1", "--status", "done"})
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "PBI #1 updated") {
		t.Fatalf("expected update message: %s", stdout)
	}
}

func TestPBIUpdateMissingID(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"pbi", "update", "--status", "done"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "id is required (use '<id>' or --id)") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestPBIUpdatePositionalAndFlagIDConflict(t *testing.T) {
	setTestHome(t)

	_, stderr, code := captureRun(t, []string{"pbi", "update", "1", "--id", "2", "--status", "done"})
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr, "positional id 1 does not match --id 2") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func mustWriteTestFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func setTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func setWorkingDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
}

func setTestEnv(t *testing.T, baseURL string) {
	t.Helper()
	setTestHome(t)
	t.Setenv("EASY8_BASE_URL", baseURL)
	t.Setenv("EASY8_API_KEY", "test-key")
}

func captureRun(t *testing.T, args []string) (string, string, int) {
	return captureRunWithInput(t, args, "")
}

func captureRunWithInput(t *testing.T, args []string, input string) (string, string, int) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	if input != "" {
		inputFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
		if err != nil {
			t.Fatalf("create stdin: %v", err)
		}
		if _, err := inputFile.WriteString(input); err != nil {
			t.Fatalf("write stdin: %v", err)
		}
		if _, err := inputFile.Seek(0, 0); err != nil {
			t.Fatalf("seek stdin: %v", err)
		}
		defer inputFile.Close()
		os.Stdin = inputFile
	}
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	code := Run(args)

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	if _, err := stdout.ReadFrom(stdoutReader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if _, err := stderr.ReadFrom(stderrReader); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	_ = stdoutReader.Close()
	_ = stderrReader.Close()

	return stdout.String(), stderr.String(), code
}

type testEnvelope struct {
	OK          bool               `json:"ok"`
	Data        json.RawMessage    `json:"data"`
	Summary     string             `json:"summary,omitempty"`
	Breadcrumbs []outputBreadcrumb `json:"breadcrumbs,omitempty"`
	Context     map[string]any     `json:"context,omitempty"`
}

func decodeEnvelope(t *testing.T, raw string) testEnvelope {
	t.Helper()
	var env testEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("envelope json error: %v", err)
	}
	if len(env.Data) == 0 {
		t.Fatalf("envelope missing data: %s", raw)
	}
	return env
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.NewServeMux()
	handler.HandleFunc("/issues.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			// If issue_id filter is set, return matching issues with full details
			if issueID := r.URL.Query().Get("issue_id"); issueID == "100" {
				_, _ = w.Write([]byte(`{"issues":[{"id":100,"subject":"Task 100","status":{"id":1,"name":"New"},"assigned_to":{"id":2,"name":"Alice"},"done_ratio":50,"updated_on":"2024-01-10"}],"total_count":1,"offset":0,"limit":25}`))
				return
			}
			_, _ = w.Write([]byte("{\"issues\":[{\"id\":101,\"subject\":\"Fix onboarding\",\"status\":{\"id\":1,\"name\":\"New\"},\"assigned_to\":{\"id\":2,\"name\":\"Alice\"},\"updated_on\":\"2024-01-01\"}],\"total_count\":1,\"offset\":0,\"limit\":25}"))
			return
		}
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{\"issue\":{\"id\":202,\"subject\":\"New task\"}}"))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	handler.HandleFunc("/issues/101.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			inc := r.URL.Query().Get("include")
			if strings.Contains(inc, "journals") || strings.Contains(inc, "attachments") {
				_, _ = w.Write([]byte(`{"issue":{"id":101,"subject":"Fix onboarding","description":"Onboarding flow needs fixing","status":{"id":1,"name":"New"},"priority":{"id":4,"name":"Normal"},"parent":{"id":100,"name":"Parent task"},"project":{"id":5,"name":"Alpha"},"tracker":{"id":7,"name":"Task"},"author":{"id":1,"name":"Bob"},"assigned_to":{"id":2,"name":"Alice"},"done_ratio":30,"start_date":"2024-01-01","due_date":"2024-02-01","created_on":"2024-01-01","updated_on":"2024-01-15","journals":[{"id":500,"user":{"id":1,"name":"Bob"},"notes":"Started work on this","created_on":"2024-01-02","private_notes":false,"details":[]},{"id":501,"user":{"id":2,"name":"Alice"},"notes":"","created_on":"2024-01-03","private_notes":false,"details":[{"property":"attr","name":"status_id","old_value":"1","new_value":"2"}]}],"attachments":[{"id":300,"filename":"design.pdf","filesize":204800,"content_type":"application/pdf","description":"Design doc","version":1,"content_url":"https://example.com/att/300","author":{"id":1,"name":"Bob"},"created_on":"2024-01-01"}]}}`))
			} else {
				_, _ = w.Write([]byte(`{"issue":{"id":101,"subject":"Fix onboarding","description":"Onboarding flow needs fixing","status":{"id":1,"name":"New"},"priority":{"id":4,"name":"Normal"},"parent":{"id":100,"name":"Parent task"},"project":{"id":5,"name":"Alpha"},"tracker":{"id":7,"name":"Task"},"author":{"id":1,"name":"Bob"},"assigned_to":{"id":2,"name":"Alice"},"done_ratio":30,"start_date":"2024-01-01","due_date":"2024-02-01","created_on":"2024-01-01","updated_on":"2024-01-15"}}`))
			}
			return
		}
		if r.Method == http.MethodPut {
			// Real Redmine returns 200 with empty body on successful PUT.
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	handler.HandleFunc("/easy_product_backlog_items.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"easy_product_backlog_items":[{"id":1,"name":"PBI Alpha","status":"to_do","estimate":"3","author":{"id":1,"name":"Alice"},"updated_at":"2024-01-01"}],"total_count":1,"offset":0,"limit":25}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	handler.HandleFunc("/easy_product_backlog_items/1.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"easy_product_backlog_item":{"id":1,"name":"PBI Alpha","description":"Full description","estimate":"3","estimate_float":3.0,"status":"to_do","author":{"id":1,"name":"Alice"},"easy_product_backlog_board":{"id":17,"name":"Design"},"issues":[{"id":100,"subject":"Task 100"}],"easy_sticky_notes":[{"id":10,"name":"Review","status":"done"}],"created_at":"2024-01-01","updated_at":"2024-01-15"}}`))
			return
		}
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	return httptest.NewServer(handler)
}

func newLookupServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.NewServeMux()
	handler.HandleFunc("/users.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"users\":[{\"id\":11,\"login\":\"alice\",\"firstname\":\"Alice\",\"lastname\":\"Doe\"}],\"total_count\":1,\"offset\":0,\"limit\":100}"))
	})
	handler.HandleFunc("/issue_statuses.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"issue_statuses\":[{\"id\":2,\"name\":\"New\"}]}"))
	})
	handler.HandleFunc("/enumerations/issue_priorities.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"issue_priorities\":[{\"id\":3,\"name\":\"High\"}]}"))
	})
	handler.HandleFunc("/trackers.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"trackers\":[{\"id\":4,\"name\":\"Task\"}]}"))
	})
	handler.HandleFunc("/projects.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"projects\":[{\"id\":5,\"name\":\"Project A\"}],\"total_count\":1,\"offset\":0,\"limit\":100}"))
	})
	handler.HandleFunc("/issues.json", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("assigned_to_id") != "11" {
			t.Fatalf("assigned_to_id = %s", query.Get("assigned_to_id"))
		}
		if query.Get("status_id") != "2" {
			t.Fatalf("status_id = %s", query.Get("status_id"))
		}
		if query.Get("priority_id") != "3" {
			t.Fatalf("priority_id = %s", query.Get("priority_id"))
		}
		if query.Get("tracker_id") != "4" {
			t.Fatalf("tracker_id = %s", query.Get("tracker_id"))
		}
		if query.Get("project_id") != "5" {
			t.Fatalf("project_id = %s", query.Get("project_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"issues\":[{\"id\":101,\"subject\":\"Fix onboarding\",\"status\":{\"id\":1,\"name\":\"New\"},\"assigned_to\":{\"id\":2,\"name\":\"Alice\"},\"updated_on\":\"2024-01-01\"}],\"total_count\":1,\"offset\":0,\"limit\":25}"))
	})

	return httptest.NewServer(handler)
}

func newErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := http.NewServeMux()
	handler.HandleFunc("/issues.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	return httptest.NewServer(handler)
}
