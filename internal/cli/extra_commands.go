package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"easy8-cli/internal/config"
	"easy8-cli/skills"
)

type commandInfo struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Flags       []flagInfo    `json:"flags,omitempty"`
	Subcommands []commandInfo `json:"subcommands,omitempty"`
}

type flagInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type skillInfo struct {
	Name string `json:"name"`
}

type skillSyncItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Action string `json:"action"`
}

type skillSyncResult struct {
	Target string          `json:"target"`
	Scope  string          `json:"scope"`
	Root   string          `json:"root"`
	DryRun bool            `json:"dry_run"`
	Skills []skillSyncItem `json:"skills"`
}

func runSkill(args []string, cfg config.Config) int {
	if len(args) == 0 {
		_, _ = os.Stdout.Write(skills.Content)
		if len(skills.Content) == 0 || skills.Content[len(skills.Content)-1] != '\n' {
			fmt.Fprintln(os.Stdout)
		}
		return 0
	}

	switch args[0] {
	case "install":
		return runSkillInstall(args[1:])
	case "list":
		return runSkillList(args[1:])
	case "sync":
		return runSkillSync(args[1:])
	case "help", "-h", "--help":
		printSkillUsage()
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown skill command:", args[0])
		printSkillUsage()
		return 2
	}
}

func runSkillInstall(args []string) int {
	fs := flag.NewFlagSet("skill install", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	target := fs.String("target", "opencode", "Install target: opencode, claude, codex")
	pathFlag := fs.String("path", "", "Custom output path (file or directory)")
	global := fs.Bool("global", false, "Install into global agent directory")
	local := fs.Bool("local", false, "Install into local project directory")
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}
	if *global && *local {
		return usageError(fmt.Errorf("--global and --local cannot be used together"))
	}

	destination, scope, err := resolveSkillPath(strings.ToLower(strings.TrimSpace(*target)), strings.TrimSpace(*pathFlag), *global, *local)
	if err != nil {
		return usageError(err)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if err := os.WriteFile(destination, skills.Content, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	data := map[string]any{
		"target": strings.ToLower(strings.TrimSpace(*target)),
		"scope":  scope,
		"path":   destination,
	}
	breadcrumbs := []outputBreadcrumb{
		{Action: "show", Cmd: "easy8 skill", Description: "Print installed skill"},
	}
	if *quietOut {
		return outputJSON(data)
	}
	if *jsonOut {
		return outputJSONEnvelope(data, "Skill installed", breadcrumbs, nil)
	}

	fmt.Fprintf(os.Stdout, "Skill installed to %s\n", destination)
	return 0
}

func runSkillList(args []string) int {
	fs := flag.NewFlagSet("skill list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	list := make([]skillInfo, 0, len(skills.Names))
	for _, name := range skills.Names {
		list = append(list, skillInfo{Name: name})
	}

	if *quietOut {
		return outputJSON(list)
	}
	if *jsonOut {
		return outputJSONEnvelope(list, fmt.Sprintf("%d bundled skills", len(list)), nil, nil)
	}

	for _, item := range list {
		fmt.Fprintln(os.Stdout, item.Name)
	}
	return 0
}

func runSkillSync(args []string) int {
	fs := flag.NewFlagSet("skill sync", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	target := fs.String("target", "opencode", "Install target: opencode, claude, codex")
	pathFlag := fs.String("path", "", "Custom output directory")
	global := fs.Bool("global", false, "Install into global agent directory")
	local := fs.Bool("local", false, "Install into local project directory")
	dryRun := fs.Bool("dry-run", false, "Show planned writes without changing files")
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}
	if *global && *local {
		return usageError(fmt.Errorf("--global and --local cannot be used together"))
	}

	targetName := strings.ToLower(strings.TrimSpace(*target))
	root, scope, err := resolveSkillRoot(targetName, strings.TrimSpace(*pathFlag), *global, *local)
	if err != nil {
		return usageError(err)
	}

	items := make([]skillSyncItem, 0, len(skills.Names))
	for _, name := range skills.Names {
		content, err := skills.Read(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}

		destination := filepath.Join(root, name, "SKILL.md")
		action := "written"
		if *dryRun {
			action = "would-write"
		} else {
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return 1
			}
			if err := os.WriteFile(destination, content, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return 1
			}
		}

		items = append(items, skillSyncItem{Name: name, Path: destination, Action: action})
	}

	result := skillSyncResult{
		Target: targetName,
		Scope:  scope,
		Root:   root,
		DryRun: *dryRun,
		Skills: items,
	}
	if *quietOut {
		return outputJSON(result)
	}
	if *jsonOut {
		summary := fmt.Sprintf("%d skills synced", len(items))
		if *dryRun {
			summary = fmt.Sprintf("%d skills would be synced", len(items))
		}
		return outputJSONEnvelope(result, summary, nil, nil)
	}

	if *dryRun {
		fmt.Fprintf(os.Stdout, "Would sync %d skills to %s\n", len(items), root)
	} else {
		fmt.Fprintf(os.Stdout, "Synced %d skills to %s\n", len(items), root)
	}
	for _, item := range items {
		fmt.Fprintf(os.Stdout, "  %s\t%s\n", item.Name, item.Path)
	}
	return 0
}

func resolveSkillPath(target, customPath string, global, local bool) (string, string, error) {
	return resolveSkillFilePath(target, customPath, global, local, skills.PrimaryName)
}

func resolveSkillFilePath(target, customPath string, global, local bool, skillName string) (string, string, error) {
	if customPath != "" {
		if strings.HasSuffix(strings.ToLower(customPath), ".md") {
			return customPath, "custom", nil
		}
		return filepath.Join(customPath, skillName, "SKILL.md"), "custom", nil
	}
	root, scope, err := resolveSkillRoot(target, customPath, global, local)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(root, skillName, "SKILL.md"), scope, nil
}

func resolveSkillRoot(target, customPath string, global, local bool) (string, string, error) {
	if customPath != "" {
		if strings.HasSuffix(strings.ToLower(customPath), ".md") {
			return "", "", fmt.Errorf("--path must be a directory for skill sync")
		}
		return customPath, "custom", nil
	}

	scope := "global"
	if local {
		scope = "local"
	}
	if !global && !local {
		scope = "global"
	}

	if scope == "global" {
		switch target {
		case "opencode":
			root, err := defaultOpenCodeSkillsRoot()
			return root, scope, err
		case "claude":
			home, err := os.UserHomeDir()
			if err != nil {
				return "", "", err
			}
			return filepath.Join(home, ".claude", "skills"), scope, nil
		case "codex":
			home, err := os.UserHomeDir()
			if err != nil {
				return "", "", err
			}
			return filepath.Join(home, ".codex", "skills"), scope, nil
		default:
			return "", "", fmt.Errorf("unsupported --target: %s", target)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	switch target {
	case "opencode":
		return filepath.Join(wd, ".opencode", "skills"), scope, nil
	case "claude":
		return filepath.Join(wd, ".claude", "skills"), scope, nil
	case "codex":
		return filepath.Join(wd, ".codex", "skills"), scope, nil
	default:
		return "", "", fmt.Errorf("unsupported --target: %s", target)
	}
}

func defaultOpenCodeSkillsRoot() (string, error) {
	if runtime.GOOS == "windows" {
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, "opencode", "skills"), nil
		}
	}
	if xdgConfig := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfig != "" {
		return filepath.Join(xdgConfig, "opencode", "skills"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opencode", "skills"), nil
}

func printSkillUsage() {
	lines := []string{
		"easy8 skill",
		"",
		"Usage:",
		"  easy8 skill",
		"  easy8 skill install [flags]",
		"  easy8 skill list [flags]",
		"  easy8 skill sync [flags]",
		"",
		"Examples:",
		"  easy8 skill",
		"  easy8 skill install --target opencode",
		"  easy8 skill list",
		"  easy8 skill sync --target opencode",
		"  easy8 skill sync --target opencode --local",
		"  easy8 skill sync --target opencode --dry-run",
		"  easy8 skill install --target claude",
		"  easy8 skill install --target codex --local",
		"  easy8 skill install --path ./custom/skills",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func runCommands(args []string) int {
	fs := flag.NewFlagSet("commands", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	catalog := buildCommandCatalog()
	if *quietOut {
		return outputJSON(catalog)
	}
	if *jsonOut {
		return outputJSONEnvelope(catalog, fmt.Sprintf("%d command groups", len(catalog)), nil, nil)
	}

	for _, cmd := range catalog {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", cmd.Name, cmd.Description)
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(os.Stdout, "  %s\t%s\n", sub.Name, sub.Description)
		}
	}
	return 0
}

func buildCommandCatalog() []commandInfo {
	return []commandInfo{
		{
			Name:        "easy8 issue",
			Description: "Issue operations",
			Subcommands: []commandInfo{
				{Name: "easy8 issue create", Description: "Create issue", Flags: issuePlanningWriteFlags()},
				{Name: "easy8 issue show", Description: "Show issue detail"},
				{Name: "easy8 issue list", Description: "List issues", Flags: issuePlanningFilterFlags()},
				{Name: "easy8 issue search", Description: "Search issues", Flags: issuePlanningFilterFlags()},
				{Name: "easy8 issue update", Description: "Update issue", Flags: issuePlanningWriteFlags()},
			},
		},
		{
			Name:        "easy8 pbi",
			Description: "PBI operations",
			Subcommands: []commandInfo{
				{Name: "easy8 pbi list", Description: "List PBIs"},
				{Name: "easy8 pbi show", Description: "Show PBI detail"},
				{Name: "easy8 pbi update", Description: "Update PBI"},
			},
		},
		{
			Name:        "easy8 auth",
			Description: "Authentication helpers",
			Subcommands: []commandInfo{
				{Name: "easy8 auth status", Description: "Show auth status"},
				{Name: "easy8 auth login", Description: "Save API key"},
				{Name: "easy8 auth logout", Description: "Remove API key"},
			},
		},
		{
			Name:        "easy8 setup",
			Description: "Configure base URL and API key",
		},
		{
			Name:        "easy8 skill",
			Description: "Print/install skill file",
			Subcommands: []commandInfo{
				{Name: "easy8 skill install", Description: "Install primary skill for coding agents"},
				{Name: "easy8 skill list", Description: "List bundled skills"},
				{Name: "easy8 skill sync", Description: "Sync bundled skills for coding agents"},
			},
		},
		{
			Name:        "easy8 commands",
			Description: "List command catalog",
		},
		{
			Name:        "easy8 update",
			Description: "Update easy8 from GitHub Releases",
		},
		{
			Name:        "easy8 version",
			Description: "Show version",
		},
	}
}

func issuePlanningWriteFlags() []flagInfo {
	return []flagInfo{
		{Name: "--sprint-id", Description: "Native sprint ID (positive integer)"},
		{Name: "--story-points", Description: "Native story points (integer, including zero)"},
		{Name: "--clear-sprint", Description: "Remove sprint with explicit null; conflicts with --sprint-id"},
		{Name: "--clear-story-points", Description: "Clear points with explicit null; conflicts with --story-points"},
		{Name: "--custom-fields", Description: "JSON array of {id, value} custom field updates"},
	}
}

func issuePlanningFilterFlags() []flagInfo {
	return []flagInfo{
		{Name: "--sprint-id", Description: "Filter by native sprint ID"},
		{Name: "--all-statuses", Description: "Include open and closed tasks"},
	}
}

func runAuth(args []string, cfg config.Config) int {
	if len(args) == 0 {
		printAuthUsage()
		return 2
	}

	switch args[0] {
	case "status":
		return runAuthStatus(args[1:], cfg)
	case "login":
		return runAuthLogin(args[1:], cfg)
	case "logout":
		return runAuthLogout(args[1:], cfg)
	case "help", "-h", "--help":
		printAuthUsage()
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown auth command:", args[0])
		printAuthUsage()
		return 2
	}
}

func runAuthStatus(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("auth status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	status := map[string]any{
		"authenticated": strings.TrimSpace(cfg.APIKey) != "",
		"base_url":      cfg.BaseURL,
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		status["api_key_masked"] = maskSecret(cfg.APIKey)
	}

	if *quietOut {
		return outputJSON(status)
	}
	if *jsonOut {
		summary := "Not authenticated"
		if strings.TrimSpace(cfg.APIKey) != "" {
			summary = "Authenticated"
		}
		return outputJSONEnvelope(status, summary, nil, nil)
	}

	fmt.Fprintf(os.Stdout, "Authenticated: %t\n", strings.TrimSpace(cfg.APIKey) != "")
	fmt.Fprintf(os.Stdout, "Base URL: %s\n", cfg.BaseURL)
	return 0
}

func runAuthLogin(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	apiKey := fs.String("api-key", "", "API key to save")
	local := fs.Bool("local", false, "Save into local .easy8.yaml")
	global := fs.Bool("global", false, "Save into global config")
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}
	if *global && *local {
		return usageError(fmt.Errorf("--global and --local cannot be used together"))
	}

	key := strings.TrimSpace(*apiKey)
	if key == "" && fs.NArg() == 1 {
		key = strings.TrimSpace(fs.Arg(0))
	}
	if key == "" {
		return usageError(fmt.Errorf("API key is required (use --api-key or positional token)"))
	}

	cfg.APIKey = key
	var (
		path  string
		err   error
		scope = "global"
	)
	if *local {
		scope = "local"
		path, err = config.SaveLocal(cfg)
	} else {
		path, err = config.SaveGlobal(cfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}

	data := map[string]any{
		"authenticated": true,
		"scope":         scope,
		"path":          path,
	}
	breadcrumbs := []outputBreadcrumb{{Action: "status", Cmd: "easy8 auth status --json", Description: "Check auth status"}}
	if *quietOut {
		return outputJSON(data)
	}
	if *jsonOut {
		return outputJSONEnvelope(data, "API key saved", breadcrumbs, nil)
	}

	fmt.Fprintf(os.Stdout, "API key saved to %s\n", path)
	return 0
}

func runAuthLogout(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("auth logout", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	local := fs.Bool("local", false, "Clear API key in local .easy8.yaml")
	global := fs.Bool("global", false, "Clear API key in global config")
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}
	if *global && *local {
		return usageError(fmt.Errorf("--global and --local cannot be used together"))
	}

	cfg.APIKey = ""
	var (
		path  string
		err   error
		scope = "global"
	)
	if *local {
		scope = "local"
		path, err = config.SaveLocal(cfg)
	} else {
		path, err = config.SaveGlobal(cfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}

	data := map[string]any{
		"authenticated": false,
		"scope":         scope,
		"path":          path,
	}
	breadcrumbs := []outputBreadcrumb{{Action: "login", Cmd: "easy8 auth login --api-key <key>", Description: "Save API key"}}
	if *quietOut {
		return outputJSON(data)
	}
	if *jsonOut {
		return outputJSONEnvelope(data, "API key removed", breadcrumbs, nil)
	}

	fmt.Fprintf(os.Stdout, "API key removed from %s\n", path)
	return 0
}

func printAuthUsage() {
	lines := []string{
		"easy8 auth",
		"",
		"Usage:",
		"  easy8 auth status [flags]",
		"  easy8 auth login [flags] [token]",
		"  easy8 auth logout [flags]",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func validateMachineFlags(jsonOut, quietOut bool) error {
	if jsonOut && quietOut {
		return fmt.Errorf("--json and --quiet cannot be used together")
	}
	return nil
}

func maskSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) <= 6 {
		return strings.Repeat("*", len(trimmed))
	}
	return trimmed[:3] + strings.Repeat("*", len(trimmed)-6) + trimmed[len(trimmed)-3:]
}
