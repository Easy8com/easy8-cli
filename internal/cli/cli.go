package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"easy8-cli/internal/api"
	"easy8-cli/internal/config"
)

// Version can be overridden at build time via -ldflags "-X easy8-cli/internal/cli.Version=..."
var Version = "0.1.8"

const setupBanner = `                                   ┌─────────┐
███████╗ █████╗ ███████╗██╗   ██╗  │ ███████ │
██╔════╝██╔══██╗██╔════╝╚██╗ ██╔╝  │██     ██│
█████╗  ███████║███████╗ ╚████╔╝   │ ███████ │
██╔══╝  ██╔══██║╚════██║  ╚██╔╝    │██     ██│
███████╗██║  ██║███████║   ██║     │ ███████ │
╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝     └─────────┘`

func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	switch args[0] {
	case "commands":
		return runCommands(args[1:])
	case "update":
		return runUpdate(args[1:])
	case "version", "--version", "-v":
		fmt.Println(Version)
		return 0
	case "help", "-h", "--help":
		printUsage()
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}
	if shouldRunStartupAutoUpdate(args[0]) {
		maybeRunAutoUpdate(cfg.AutoUpdate)
	}

	switch args[0] {
	case "issue":
		return runIssue(args[1:], cfg)
	case "pbi":
		return runPBI(args[1:], cfg)
	case "auth":
		return runAuth(args[1:], cfg)
	case "setup":
		return runSetup(args[1:], cfg)
	case "skill":
		return runSkill(args[1:], cfg)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", args[0])
		printUsage()
		return 2
	}
}

func runIssue(args []string, cfg config.Config) int {
	if len(args) == 0 {
		printIssueUsage()
		return 2
	}

	client := api.NewClient(cfg)

	switch args[0] {
	case "create":
		return runIssueCreate(args[1:], cfg, client)
	case "show":
		return runIssueShow(args[1:], cfg, client)
	case "list":
		return runIssueList(args[1:], cfg, client)
	case "search":
		return runIssueSearch(args[1:], cfg, client)
	case "update":
		return runIssueUpdate(args[1:], cfg, client)
	case "help", "-h", "--help":
		printIssueUsage()
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown issue command:", args[0])
		printIssueUsage()
		return 2
	}
}

func runSetup(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	baseURL := fs.String("base-url", "", "Easy8 base URL")
	apiKey := fs.String("api-key", "", "Easy8 API key")
	var projectID optionalInt
	var trackerID optionalInt
	var statusID optionalInt
	var priorityID optionalInt
	var authorID optionalInt
	var assignedToID optionalInt
	var autoUpdate optionalBool
	fs.Var(&projectID, "project-id", "Default project ID")
	fs.Var(&trackerID, "tracker-id", "Default tracker ID")
	fs.Var(&statusID, "status-id", "Default status ID")
	fs.Var(&priorityID, "priority-id", "Default priority ID")
	fs.Var(&authorID, "author-id", "Default author ID")
	fs.Var(&assignedToID, "assigned-to-id", "Default assigned-to ID")
	fs.Var(&autoUpdate, "autoupdate", "Enable automatic daily self-update")
	globalOut := fs.Bool("global", false, "Save to global config (~/.config/easy8/config.yaml)")
	localOut := fs.Bool("local", false, "Save to local config (.easy8.yaml)")
	nonInteractive := fs.Bool("non-interactive", false, "Do not prompt for missing values")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		return usageError(fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " ")))
	}
	if *globalOut && *localOut {
		return usageError(fmt.Errorf("--global and --local cannot be used together"))
	}

	if !*nonInteractive {
		fmt.Fprintln(os.Stdout, setupBanner)
		fmt.Fprintln(os.Stdout)
	}

	saveLocal := *localOut
	if !*globalOut && !*localOut && !*nonInteractive {
		fmt.Fprintln(os.Stdout, "Config scope:")
		fmt.Fprintln(os.Stdout, "  global -> ~/.config/easy8/config.yaml (all projects)")
		fmt.Fprintln(os.Stdout, "  local  -> ./.easy8.yaml (current project only; overrides global)")
		scope, err := promptScope(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		saveLocal = (scope == "local")
	}

	baseURLValue := strings.TrimSpace(*baseURL)
	apiKeyValue := strings.TrimSpace(*apiKey)

	if *nonInteractive {
		if baseURLValue == "" {
			return usageError(fmt.Errorf("--base-url is required in --non-interactive mode"))
		}
		if apiKeyValue == "" {
			return usageError(fmt.Errorf("--api-key is required in --non-interactive mode"))
		}
	} else {
		var err error
		baseURLValue, err = promptString(os.Stdin, os.Stdout, "Easy8 base URL", firstNonEmpty(baseURLValue, cfg.BaseURL))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		apiKeyValue, err = promptString(os.Stdin, os.Stdout, "Easy8 API key", firstNonEmpty(apiKeyValue, cfg.APIKey))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		if !autoUpdate.set {
			autoUpdateValue, err := promptBool(os.Stdin, os.Stdout, "Enable automatic daily updates", true)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return 1
			}
			autoUpdate.value = autoUpdateValue
			autoUpdate.set = true
		}
	}

	if strings.TrimSpace(baseURLValue) == "" {
		return usageError(fmt.Errorf("base URL is required"))
	}
	if strings.TrimSpace(apiKeyValue) == "" {
		return usageError(fmt.Errorf("API key is required"))
	}
	if err := validateBaseURL(baseURLValue); err != nil {
		return usageError(err)
	}

	defaults := cfg.Defaults
	if projectID.set {
		defaults.ProjectID = projectID.value
	}
	if trackerID.set {
		defaults.TrackerID = trackerID.value
	}
	if statusID.set {
		defaults.StatusID = statusID.value
	}
	if priorityID.set {
		defaults.PriorityID = priorityID.value
	}
	if authorID.set {
		defaults.AuthorID = authorID.value
	}
	if assignedToID.set {
		defaults.AssignedToID = assignedToID.value
	}
	autoUpdateValue := true
	if autoUpdate.set {
		autoUpdateValue = autoUpdate.value
	}

	newCfg := config.Config{
		BaseURL:    strings.TrimSpace(baseURLValue),
		APIKey:     strings.TrimSpace(apiKeyValue),
		AutoUpdate: autoUpdateValue,
		Defaults:   defaults,
	}

	var (
		path string
		err  error
	)
	if saveLocal {
		path, err = config.SaveLocal(newCfg)
	} else {
		path, err = config.SaveGlobal(newCfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Configuration saved to %s\n", path)
	return 0
}

func runIssueCreate(args []string, cfg config.Config, client *api.Client) int {
	fs := flag.NewFlagSet("issue create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	subject := fs.String("subject", "", "Issue subject (required)")
	description := fs.String("description", "", "Issue description")
	projectID := fs.Int("project-id", cfg.Defaults.ProjectID, "Project ID")
	trackerID := fs.Int("tracker-id", cfg.Defaults.TrackerID, "Tracker ID")
	statusID := fs.Int("status-id", cfg.Defaults.StatusID, "Status ID")
	priorityID := fs.Int("priority-id", cfg.Defaults.PriorityID, "Priority ID")
	var parentID optionalInt
	authorID := fs.Int("author-id", cfg.Defaults.AuthorID, "Author ID")
	assignedToID := fs.Int("assigned-to-id", cfg.Defaults.AssignedToID, "Assigned to user ID")
	startDate := fs.String("start-date", "", "Start date (YYYY-MM-DD)")
	dueDate := fs.String("due-date", "", "Due date (YYYY-MM-DD)")
	var doneRatio optionalInt
	fs.Var(&parentID, "parent-id", "Parent issue ID")
	fs.Var(&doneRatio, "done-ratio", "Done ratio (0-100)")
	fields := issueFieldFlags{}
	fields.register(fs)
	attachmentArgs := issueAttachmentArgs{}
	fs.Var(&issueAttachmentPathValue{args: &attachmentArgs}, "attachment", "Attachment file path (repeatable)")
	fs.Var(&issueAttachmentDescriptionValue{args: &attachmentArgs}, "attachment-description", "Description for the preceding --attachment")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if err := requireString("subject", *subject); err != nil {
		return usageError(err)
	}
	if err := requireInt("project-id", *projectID); err != nil {
		return usageError(err)
	}
	if err := requireInt("tracker-id", *trackerID); err != nil {
		return usageError(err)
	}
	if err := requireInt("status-id", *statusID); err != nil {
		return usageError(err)
	}
	if err := requireInt("priority-id", *priorityID); err != nil {
		return usageError(err)
	}
	if err := requireInt("author-id", *authorID); err != nil {
		return usageError(err)
	}
	if err := requireInt("assigned-to-id", *assignedToID); err != nil {
		return usageError(err)
	}

	input := api.IssueInput{
		Subject:      stringPtr(*subject),
		ProjectID:    intPtr(*projectID),
		TrackerID:    intPtr(*trackerID),
		StatusID:     intPtr(*statusID),
		PriorityID:   intPtr(*priorityID),
		AuthorID:     intPtr(*authorID),
		AssignedToID: intPtr(*assignedToID),
	}
	if err := fields.apply(&input); err != nil {
		return usageError(err)
	}
	if parentID.set {
		if parentID.value <= 0 {
			return usageError(fmt.Errorf("--parent-id must be greater than 0"))
		}
		input.ParentIssueID = intPtr(parentID.value)
	}
	if descriptionHTML := formatCKEditorHTML(*description); descriptionHTML != "" {
		input.Description = stringPtr(descriptionHTML)
	}
	if strings.TrimSpace(*startDate) != "" {
		input.StartDate = stringPtr(*startDate)
	}
	if strings.TrimSpace(*dueDate) != "" {
		input.DueDate = stringPtr(*dueDate)
	}
	if doneRatio.set {
		if doneRatio.value < 0 || doneRatio.value > 100 {
			return usageError(fmt.Errorf("--done-ratio must be between 0 and 100"))
		}
		input.DoneRatio = intPtr(doneRatio.value)
	}
	if len(attachmentArgs.items) > 0 {
		uploads, err := prepareIssueUploads(context.Background(), client, attachmentArgs.items)
		if err != nil {
			return apiError(err)
		}
		input.Uploads = uploads
	}

	resp, err := client.CreateIssue(context.Background(), input)
	if err != nil {
		return apiError(err)
	}
	if err := resp.Issue.VerifyAgileFields(input); err != nil {
		return apiError(fmt.Errorf("issue #%d create could not be verified: %v; the HTTP create may have succeeded partially; no rollback was performed", resp.Issue.ID, err))
	}

	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("Issue #%d created", resp.Issue.ID)
		breadcrumbs := []outputBreadcrumb{
			{Action: "show", Cmd: fmt.Sprintf("easy8 issue show %d", resp.Issue.ID), Description: "Show issue detail"},
			{Action: "update", Cmd: fmt.Sprintf("easy8 issue update %d --status-id <id>", resp.Issue.ID), Description: "Update issue"},
		}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputIssues([]api.Issue{resp.Issue})
}

func runIssueShow(args []string, cfg config.Config, client *api.Client) int {
	normalizedArgs, positionalID, hasPositionalID, err := normalizeIDArgs(args)
	if err != nil {
		return usageError(err)
	}
	explicitID, hasExplicitID, err := extractExplicitIDArg(args)
	if err != nil {
		return usageError(err)
	}

	fs := flag.NewFlagSet("issue show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	id := fs.Int("id", 0, "Issue ID (required)")
	include := fs.String("include", "", "Include fields (comma-separated, e.g. journals,attachments)")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if hasPositionalID && hasExplicitID && positionalID != explicitID {
		return usageError(fmt.Errorf("positional id %d does not match --id %d", positionalID, explicitID))
	}

	if *id == 0 {
		return usageError(fmt.Errorf("id is required (use '<id>' or --id)"))
	}

	var includes []string
	if strings.TrimSpace(*include) != "" {
		includes = splitComma(*include)
	}

	resp, err := client.GetIssue(context.Background(), *id, includes)
	if err != nil {
		return apiError(err)
	}

	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("Issue #%d: %s", resp.Issue.ID, resp.Issue.Subject)
		breadcrumbs := []outputBreadcrumb{
			{Action: "update", Cmd: fmt.Sprintf("easy8 issue update %d --status-id <id>", resp.Issue.ID), Description: "Update issue"},
			{Action: "list", Cmd: "easy8 issue list", Description: "List issues"},
			{Action: "search", Cmd: "easy8 issue search --q \"text\"", Description: "Search issues"},
		}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputIssueDetail(resp.Issue)
}

func runIssueList(args []string, cfg config.Config, client *api.Client) int {
	fs := flag.NewFlagSet("issue list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	limit := fs.Int("limit", 25, "Limit (max 100)")
	offset := fs.Int("offset", 0, "Offset")
	sort := fs.String("sort", "", "Sort expression")
	query := fs.String("q", "", "Free-text query (easy_query_q)")
	var sprintID optionalInt
	fs.Var(&sprintID, "sprint-id", "Native sprint ID filter (positive integer)")
	allStatuses := fs.Bool("all-statuses", false, "Include open and closed tasks")
	include := fs.String("include", "", "Include fields (comma-separated)")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	params := api.IssueListParams{
		Limit:       *limit,
		Offset:      *offset,
		Sort:        strings.TrimSpace(*sort),
		Query:       strings.TrimSpace(*query),
		SprintID:    sprintID.value,
		AllStatuses: *allStatuses,
	}
	if sprintID.set && sprintID.value <= 0 {
		return usageError(fmt.Errorf("--sprint-id must be greater than 0"))
	}
	if strings.TrimSpace(*include) != "" {
		params.Include = splitComma(*include)
	}

	resp, err := client.ListIssues(context.Background(), params)
	if err != nil {
		return apiError(err)
	}
	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("%d issues", len(resp.Issues))
		breadcrumbs := []outputBreadcrumb{{Action: "show", Cmd: "easy8 issue show <id>", Description: "Show issue detail"}}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputIssues(resp.Issues)
}

func runIssueSearch(args []string, cfg config.Config, client *api.Client) int {
	fs := flag.NewFlagSet("issue search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	query := fs.String("q", "", "Search query")
	limit := fs.Int("limit", 25, "Limit (max 100)")
	offset := fs.Int("offset", 0, "Offset")
	sort := fs.String("sort", "", "Sort expression")
	include := fs.String("include", "", "Include fields (comma-separated)")
	var assigneeID optionalInt
	var statusID optionalInt
	var priorityID optionalInt
	var taskTypeID optionalInt
	var projectID optionalInt
	var sprintID optionalInt
	fs.Var(&sprintID, "sprint-id", "Native sprint ID filter (positive integer)")
	allStatuses := fs.Bool("all-statuses", false, "Include open and closed tasks; conflicts with --status and --status-id")
	fs.Var(&assigneeID, "assignee-id", "Assignee user ID")
	fs.Var(&statusID, "status-id", "Status ID")
	fs.Var(&priorityID, "priority-id", "Priority ID")
	fs.Var(&taskTypeID, "task-type-id", "Task type (tracker) ID")
	fs.Var(&projectID, "project-id", "Project ID")
	var dueDate string
	var subject string
	var assignee string
	var status string
	var priority string
	var taskType string
	var project string
	fs.StringVar(&dueDate, "due-date", "", "Due date (YYYY-MM-DD)")
	fs.StringVar(&subject, "subject", "", "Subject filter")
	fs.StringVar(&assignee, "assignee", "", "Assignee login or name")
	fs.StringVar(&status, "status", "", "Status name")
	fs.StringVar(&priority, "priority", "", "Priority name")
	fs.StringVar(&taskType, "task-type", "", "Task type (tracker) name")
	fs.StringVar(&project, "project", "", "Project name")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if sprintID.set && sprintID.value <= 0 {
		return usageError(fmt.Errorf("--sprint-id must be greater than 0"))
	}
	if *allStatuses && (statusID.set || strings.TrimSpace(status) != "") {
		return usageError(fmt.Errorf("--all-statuses cannot be combined with --status or --status-id"))
	}
	resolvedAssigneeID, err := resolveAssigneeID(context.Background(), client, assigneeID, assignee)
	if err != nil {
		return usageError(err)
	}
	resolvedStatusID, err := resolveStatusID(context.Background(), client, statusID, status)
	if err != nil {
		return usageError(err)
	}
	resolvedPriorityID, err := resolvePriorityID(context.Background(), client, priorityID, priority)
	if err != nil {
		return usageError(err)
	}
	resolvedTaskTypeID, err := resolveTaskTypeID(context.Background(), client, taskTypeID, taskType)
	if err != nil {
		return usageError(err)
	}
	resolvedProjectID, err := resolveProjectID(context.Background(), client, projectID, project)
	if err != nil {
		return usageError(err)
	}

	queryValue := strings.TrimSpace(*query)
	if queryValue == "" && !sprintID.set && !*allStatuses && resolvedAssigneeID == 0 && resolvedStatusID == 0 && resolvedPriorityID == 0 && resolvedTaskTypeID == 0 && resolvedProjectID == 0 && strings.TrimSpace(dueDate) == "" && strings.TrimSpace(subject) == "" {
		return usageError(fmt.Errorf("at least one filter is required (e.g. --q, --status, --assignee)"))
	}

	params := api.IssueListParams{
		Limit:       *limit,
		Offset:      *offset,
		Sort:        strings.TrimSpace(*sort),
		Query:       queryValue,
		DueDate:     strings.TrimSpace(dueDate),
		Subject:     strings.TrimSpace(subject),
		AssigneeID:  resolvedAssigneeID,
		StatusID:    resolvedStatusID,
		PriorityID:  resolvedPriorityID,
		TaskTypeID:  resolvedTaskTypeID,
		ProjectID:   resolvedProjectID,
		SprintID:    sprintID.value,
		AllStatuses: *allStatuses,
	}
	if strings.TrimSpace(*include) != "" {
		params.Include = splitComma(*include)
	}

	resp, err := client.ListIssues(context.Background(), params)
	if err != nil {
		return apiError(err)
	}
	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("%d issues matched", len(resp.Issues))
		breadcrumbs := []outputBreadcrumb{{Action: "show", Cmd: "easy8 issue show <id>", Description: "Show issue detail"}}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputIssues(resp.Issues)
}

func runIssueUpdate(args []string, cfg config.Config, client *api.Client) int {
	normalizedArgs, positionalID, hasPositionalID, err := normalizeIDArgs(args)
	if err != nil {
		return usageError(err)
	}
	explicitID, hasExplicitID, err := extractExplicitIDArg(args)
	if err != nil {
		return usageError(err)
	}

	fs := flag.NewFlagSet("issue update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	id := fs.Int("id", 0, "Issue ID (required)")
	subject := fs.String("subject", "", "Issue subject")
	description := fs.String("description", "", "Issue description")
	var statusID optionalInt
	var priorityID optionalInt
	var parentID optionalInt
	var assignedToID optionalInt
	var doneRatio optionalInt
	fs.Var(&statusID, "status-id", "Status ID")
	fs.Var(&priorityID, "priority-id", "Priority ID")
	fs.Var(&parentID, "parent-id", "Parent issue ID")
	fs.Var(&assignedToID, "assigned-to-id", "Assigned to user ID")
	fs.Var(&doneRatio, "done-ratio", "Done ratio (0-100)")
	fields := issueFieldFlags{}
	fields.register(fs)
	notes := fs.String("notes", "", "Notes (journal entry)")
	attachmentArgs := issueAttachmentArgs{}
	fs.Var(&issueAttachmentPathValue{args: &attachmentArgs}, "attachment", "Attachment file path (repeatable)")
	fs.Var(&issueAttachmentDescriptionValue{args: &attachmentArgs}, "attachment-description", "Description for the preceding --attachment")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if hasPositionalID && hasExplicitID && positionalID != explicitID {
		return usageError(fmt.Errorf("positional id %d does not match --id %d", positionalID, explicitID))
	}

	if *id == 0 {
		return usageError(fmt.Errorf("id is required (use '<id>' or --id)"))
	}

	input := api.IssueInput{}
	if err := fields.apply(&input); err != nil {
		return usageError(err)
	}
	if strings.TrimSpace(*subject) != "" {
		input.Subject = stringPtr(*subject)
	}
	if descriptionHTML := formatCKEditorHTML(*description); descriptionHTML != "" {
		input.Description = stringPtr(descriptionHTML)
	}
	if statusID.set {
		input.StatusID = intPtr(statusID.value)
	}
	if priorityID.set {
		input.PriorityID = intPtr(priorityID.value)
	}
	if parentID.set {
		if parentID.value <= 0 {
			return usageError(fmt.Errorf("--parent-id must be greater than 0"))
		}
		input.ParentIssueID = intPtr(parentID.value)
	}
	if assignedToID.set {
		input.AssignedToID = intPtr(assignedToID.value)
	}
	if doneRatio.set {
		if doneRatio.value < 0 || doneRatio.value > 100 {
			return usageError(fmt.Errorf("--done-ratio must be between 0 and 100"))
		}
		input.DoneRatio = intPtr(doneRatio.value)
	}
	if notesHTML := formatCKEditorHTML(*notes); notesHTML != "" {
		input.Notes = stringPtr(notesHTML)
	}
	if len(attachmentArgs.items) > 0 {
		uploads, err := prepareIssueUploads(context.Background(), client, attachmentArgs.items)
		if err != nil {
			return apiError(err)
		}
		input.Uploads = uploads
	}

	resp, err := client.UpdateIssue(context.Background(), *id, input)
	if err != nil {
		return apiError(err)
	}
	// Redmine PUT typically returns 200 with empty body.
	// Fetch the updated issue so the user sees the current state.
	if resp.Issue.ID == 0 {
		getResp, getErr := client.GetIssue(context.Background(), *id, nil)
		if getErr != nil {
			return apiError(fmt.Errorf("issue #%d update readback failed: %v; the HTTP update may have succeeded partially; no rollback was performed", *id, getErr))
		}
		resp = getResp
	}
	if err := resp.Issue.VerifyAgileFields(input); err != nil {
		return apiError(fmt.Errorf("issue #%d update could not be verified: %v; the HTTP update may have succeeded partially; no rollback was performed", *id, err))
	}
	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("Issue #%d updated", resp.Issue.ID)
		breadcrumbs := []outputBreadcrumb{
			{Action: "show", Cmd: fmt.Sprintf("easy8 issue show %d", resp.Issue.ID), Description: "Show updated issue"},
			{Action: "list", Cmd: "easy8 issue list", Description: "List issues"},
		}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputIssues([]api.Issue{resp.Issue})
}

// --- PBI commands ---

func runPBI(args []string, cfg config.Config) int {
	if len(args) == 0 {
		printPBIUsage()
		return 2
	}

	client := api.NewClient(cfg)

	switch args[0] {
	case "list":
		return runPBIList(args[1:], cfg, client)
	case "show":
		return runPBIShow(args[1:], cfg, client)
	case "update":
		return runPBIUpdate(args[1:], cfg, client)
	case "help", "-h", "--help":
		printPBIUsage()
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown pbi command:", args[0])
		printPBIUsage()
		return 2
	}
}

func runPBIList(args []string, cfg config.Config, client *api.Client) int {
	fs := flag.NewFlagSet("pbi list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	limit := fs.Int("limit", 25, "Max items")
	offset := fs.Int("offset", 0, "Offset")
	sort := fs.String("sort", "", "Sort (e.g. updated_at:desc)")
	query := fs.String("q", "", "Fulltext search")
	status := fs.String("status", "", "Filter by status (to_do, realization, done, deleted)")
	authorID := fs.Int("author-id", 0, "Filter by author ID")
	boardID := fs.Int("board-id", 0, "Filter by board ID")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	params := api.PBIListParams{
		Limit:    *limit,
		Offset:   *offset,
		Sort:     *sort,
		Query:    *query,
		Status:   *status,
		AuthorID: *authorID,
		BoardID:  *boardID,
	}

	resp, err := client.ListPBIs(context.Background(), params)
	if err != nil {
		return apiError(err)
	}

	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("%d PBIs", len(resp.PBIs))
		breadcrumbs := []outputBreadcrumb{{Action: "show", Cmd: "easy8 pbi show <id>", Description: "Show PBI detail"}}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputPBIs(resp.PBIs)
}

func runPBIShow(args []string, cfg config.Config, client *api.Client) int {
	normalizedArgs, positionalID, hasPositionalID, err := normalizeIDArgs(args)
	if err != nil {
		return usageError(err)
	}
	explicitID, hasExplicitID, err := extractExplicitIDArg(args)
	if err != nil {
		return usageError(err)
	}

	fs := flag.NewFlagSet("pbi show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	id := fs.Int("id", 0, "PBI ID (required)")
	jsonOut := fs.Bool("json", false, "JSON output")
	quietOut := fs.Bool("quiet", false, "Raw JSON output without envelope")

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if hasPositionalID && hasExplicitID && positionalID != explicitID {
		return usageError(fmt.Errorf("positional id %d does not match --id %d", positionalID, explicitID))
	}

	if *id == 0 {
		return usageError(fmt.Errorf("id is required (use '<id>' or --id)"))
	}

	resp, err := client.GetPBI(context.Background(), *id)
	if err != nil {
		return apiError(err)
	}

	// Fetch full issue details if PBI has linked issues
	var issues []api.Issue
	if len(resp.PBI.Issues) > 0 {
		ids := make([]int, len(resp.PBI.Issues))
		for i, issue := range resp.PBI.Issues {
			ids[i] = issue.ID
		}
		issueResp, err := client.ListIssues(context.Background(), api.IssueListParams{
			IssueIDs: ids,
			Limit:    len(ids),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch issue details: %v\n", err)
		} else {
			issues = issueResp.Issues
		}
	}

	if *quietOut {
		return outputJSON(resp)
	}
	if *jsonOut {
		summary := fmt.Sprintf("PBI #%d: %s", resp.PBI.ID, resp.PBI.Name)
		breadcrumbs := []outputBreadcrumb{
			{Action: "update", Cmd: fmt.Sprintf("easy8 pbi update %d --status done", resp.PBI.ID), Description: "Update PBI"},
			{Action: "list", Cmd: "easy8 pbi list", Description: "List PBIs"},
		}
		return outputJSONEnvelope(resp, summary, breadcrumbs, nil)
	}
	return outputPBIDetail(resp.PBI, issues)
}

func runPBIUpdate(args []string, cfg config.Config, client *api.Client) int {
	normalizedArgs, positionalID, hasPositionalID, err := normalizeIDArgs(args)
	if err != nil {
		return usageError(err)
	}
	explicitID, hasExplicitID, err := extractExplicitIDArg(args)
	if err != nil {
		return usageError(err)
	}

	fs := flag.NewFlagSet("pbi update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	id := fs.Int("id", 0, "PBI ID (required)")
	name := fs.String("name", "", "New name")
	description := fs.String("description", "", "New description")
	status := fs.String("status", "", "New status (to_do, realization, done, deleted)")
	estimate := fs.String("estimate", "", "New estimate")
	jsonOut := fs.Bool("json", false, "JSON envelope output")
	quietOut := fs.Bool("quiet", false, "Raw JSON data output")

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if err := validateMachineFlags(*jsonOut, *quietOut); err != nil {
		return usageError(err)
	}

	if hasPositionalID && hasExplicitID && positionalID != explicitID {
		return usageError(fmt.Errorf("positional id %d does not match --id %d", positionalID, explicitID))
	}

	if *id == 0 {
		return usageError(fmt.Errorf("id is required (use '<id>' or --id)"))
	}

	var input api.PBIInput
	if strings.TrimSpace(*name) != "" {
		input.Name = name
	}
	if descriptionHTML := formatCKEditorHTML(*description); descriptionHTML != "" {
		input.Description = stringPtr(descriptionHTML)
	}
	if strings.TrimSpace(*status) != "" {
		input.Status = status
	}
	if strings.TrimSpace(*estimate) != "" {
		input.Estimate = estimate
	}

	err = client.UpdatePBI(context.Background(), *id, input)
	if err != nil {
		return apiError(err)
	}

	result := map[string]any{
		"id":      *id,
		"updated": true,
	}
	breadcrumbs := []outputBreadcrumb{
		{Action: "show", Cmd: fmt.Sprintf("easy8 pbi show %d", *id), Description: "Show PBI detail"},
		{Action: "list", Cmd: "easy8 pbi list", Description: "List PBIs"},
	}
	if *quietOut {
		return outputJSON(result)
	}
	if *jsonOut {
		return outputJSONEnvelope(result, fmt.Sprintf("PBI #%d updated", *id), breadcrumbs, nil)
	}

	fmt.Fprintf(os.Stdout, "PBI #%d updated.\n", *id)
	return 0
}

func usageError(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 2
}

func requireString(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}

func requireInt(name string, value int) error {
	if value == 0 {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}

func printUsage() {
	lines := []string{
		"easy8-cli",
		"",
		"Usage:",
		"  easy8 issue <command> [flags]",
		"  easy8 pbi <command> [flags]",
		"  easy8 auth <command> [flags]",
		"  easy8 setup [flags]",
		"  easy8 skill [command] [flags]",
		"  easy8 commands [flags]",
		"  easy8 update [flags]",
		"  easy8 version",
		"",
		"Commands:",
		"  issue create   Create a new issue",
		"  issue show     Show issue detail",
		"  issue list     List issues",
		"  issue search   Fulltext search",
		"  issue update   Update an issue",
		"  pbi list       List product backlog items",
		"  pbi show       Show PBI detail",
		"  pbi update     Update a PBI",
		"  auth status    Show authentication status",
		"  auth login     Save API key",
		"  auth logout    Remove API key",
		"  setup          Configure base URL and API key",
		"  skill          Print/install skill file",
		"  commands       List command catalog",
		"  update         Update easy8 from GitHub Releases",
		"  version        Print version",
		"",
		"Use 'easy8 issue --help', 'easy8 pbi --help', 'easy8 auth --help', 'easy8 setup --help', 'easy8 skill --help', or 'easy8 update --help' for details.",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func printIssueUsage() {
	lines := []string{
		"easy8 issue",
		"",
		"Usage:",
		"  easy8 issue create [flags]",
		"  easy8 issue show <id> [flags]",
		"  easy8 issue list [flags]",
		"  easy8 issue search [flags]",
		"  easy8 issue update <id> [flags]",
		"",
		"Examples:",
		"  easy8 issue show 123",
		"  easy8 issue show 123 --include journals,attachments --json",
		"  easy8 issue show 123 --quiet",
		"  easy8 issue show --id 123",
		"  easy8 issue list --limit 10",
		"  easy8 issue list --sprint-id 34 --all-statuses --quiet",
		"  easy8 issue search --q \"onboarding\"",
		"  easy8 issue search --q \"petr\" --assignee-id 51 --status-id 2 --priority-id 3",
		"  easy8 issue search --q \"petr\" --assignee \"Alice Doe\" --status \"New\" --priority \"High\" --task-type \"Task\" --project \"Project A\"",
		"  easy8 issue create --subject \"Fix login\" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --parent-id 100",
		"  easy8 issue create --subject \"Fix login\" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --attachment ./spec.pdf --attachment-description \"Specification\"",
		"  easy8 issue update 123 --status-id 5",
		"  easy8 issue update 123 --parent-id 100",
		"  easy8 issue update 123 --sprint-id 34 --story-points 5",
		"  easy8 issue update 123 --clear-sprint --clear-story-points",
		`  easy8 issue update 123 --custom-fields '[{"id":7,"value":["a","b"]}]'`,
		"  easy8 issue update 123 --attachment ./error.log",
		"  easy8 issue update 123 --attachment ./screenshot.png --attachment-description \"Failure screenshot\"",
		"  easy8 issue update --id 123 --status-id 5",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func printPBIUsage() {
	lines := []string{
		"easy8 pbi",
		"",
		"Usage:",
		"  easy8 pbi list [flags]",
		"  easy8 pbi show <id> [flags]",
		"  easy8 pbi update <id> [flags]",
		"",
		"Examples:",
		"  easy8 pbi list --limit 10",
		"  easy8 pbi list --status to_do --board-id 17",
		"  easy8 pbi list --q \"design\" --author-id 51",
		"  easy8 pbi show 42",
		"  easy8 pbi show 42 --json",
		"  easy8 pbi show 42 --quiet",
		"  easy8 pbi show --id 42",
		"  easy8 pbi update 42 --status done",
		"  easy8 pbi update --id 42 --status done",
		"  easy8 pbi update --id 42 --name \"New name\" --estimate 5",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

type optionalInt struct {
	set   bool
	value int
}

type optionalBool struct {
	set   bool
	value bool
}

type issueAttachmentInput struct {
	Path           string
	Description    string
	descriptionSet bool
}

type issueAttachmentArgs struct {
	items []issueAttachmentInput
}

type issueAttachmentPathValue struct {
	args *issueAttachmentArgs
}

type issueAttachmentDescriptionValue struct {
	args *issueAttachmentArgs
}

func (flagValue *optionalInt) String() string {
	if !flagValue.set {
		return ""
	}
	return fmt.Sprintf("%d", flagValue.value)
}

func (flagValue *optionalBool) String() string {
	if !flagValue.set {
		return ""
	}
	return strconv.FormatBool(flagValue.value)
}

func (flagValue *optionalBool) IsBoolFlag() bool {
	return true
}

func (value *issueAttachmentPathValue) String() string {
	if value == nil || value.args == nil || len(value.args.items) == 0 {
		return ""
	}
	paths := make([]string, 0, len(value.args.items))
	for _, item := range value.args.items {
		paths = append(paths, item.Path)
	}
	return strings.Join(paths, ",")
}

func (value *issueAttachmentPathValue) Set(raw string) error {
	if value == nil || value.args == nil {
		return fmt.Errorf("attachment flags are not initialized")
	}
	path := strings.TrimSpace(raw)
	if path == "" {
		return fmt.Errorf("attachment path cannot be empty")
	}
	value.args.items = append(value.args.items, issueAttachmentInput{Path: path})
	return nil
}

func (value *issueAttachmentDescriptionValue) String() string {
	if value == nil || value.args == nil || len(value.args.items) == 0 {
		return ""
	}
	descriptions := make([]string, 0, len(value.args.items))
	for _, item := range value.args.items {
		if item.Description == "" {
			continue
		}
		descriptions = append(descriptions, item.Description)
	}
	return strings.Join(descriptions, ",")
}

func (value *issueAttachmentDescriptionValue) Set(raw string) error {
	if value == nil || value.args == nil {
		return fmt.Errorf("attachment flags are not initialized")
	}
	if len(value.args.items) == 0 {
		return fmt.Errorf("--attachment-description requires a preceding --attachment")
	}
	last := &value.args.items[len(value.args.items)-1]
	if last.descriptionSet {
		return fmt.Errorf("--attachment-description already set for %s", last.Path)
	}
	last.Description = raw
	last.descriptionSet = true
	return nil
}

func (flagValue *optionalInt) Set(value string) error {
	parsed, err := parseInt(value)
	if err != nil {
		return err
	}
	flagValue.value = parsed
	flagValue.set = true
	return nil
}

func (flagValue *optionalBool) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid bool: %s", value)
	}
	flagValue.value = parsed
	flagValue.set = true
	return nil
}

func parseInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid int: %s", value)
	}
	return parsed, nil
}

func promptString(input io.Reader, output io.Writer, label, defaultValue string) (string, error) {
	prompt := label
	if strings.TrimSpace(defaultValue) != "" {
		prompt = fmt.Sprintf("%s [%s]", label, strings.TrimSpace(defaultValue))
	}
	if _, err := fmt.Fprintf(output, "%s: ", prompt); err != nil {
		return "", err
	}
	line, err := readPromptLine(input)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = strings.TrimSpace(defaultValue)
	}
	if value == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}

func promptScope(input io.Reader, output io.Writer) (string, error) {
	if _, err := fmt.Fprint(output, "Where to save config? [global/local] (default: global): "); err != nil {
		return "", err
	}
	line, err := readPromptLine(input)
	if err != nil {
		return "", err
	}
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return "global", nil
	}
	switch value {
	case "g", "global":
		return "global", nil
	case "l", "local":
		return "local", nil
	default:
		return "", fmt.Errorf("invalid scope: %s (use global or local)", value)
	}
}

func promptBool(input io.Reader, output io.Writer, label string, defaultValue bool) (bool, error) {
	defaultLabel := "y/N"
	if defaultValue {
		defaultLabel = "Y/n"
	}
	if _, err := fmt.Fprintf(output, "%s? [%s]: ", label, defaultLabel); err != nil {
		return false, err
	}
	line, err := readPromptLine(input)
	if err != nil {
		return false, err
	}
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "y", "yes", "true", "1", "on":
		return true, nil
	case "n", "no", "false", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value for %s: %s (use yes or no)", strings.ToLower(label), value)
	}
}

func readPromptLine(input io.Reader) (string, error) {
	var builder strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := input.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return builder.String(), nil
			}
			if buf[0] != '\r' {
				builder.WriteByte(buf[0])
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return builder.String(), nil
			}
			return "", err
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateBaseURL(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("base URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("base URL must start with http:// or https://")
	}
	if parsed.Host == "" {
		return fmt.Errorf("base URL must include a host")
	}
	return nil
}

func normalizeIDArgs(args []string) ([]string, int, bool, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args, 0, false, nil
	}

	id, err := parseInt(args[0])
	if err != nil {
		return nil, 0, false, fmt.Errorf("invalid id: %s", args[0])
	}

	normalized := make([]string, 0, len(args)+1)
	normalized = append(normalized, "--id", args[0])
	normalized = append(normalized, args[1:]...)
	return normalized, id, true, nil
}

func extractExplicitIDArg(args []string) (int, bool, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--id" || arg == "-id":
			if i+1 >= len(args) {
				return 0, false, fmt.Errorf("--id requires a value")
			}
			id, err := parseInt(args[i+1])
			if err != nil {
				return 0, false, fmt.Errorf("invalid id: %s", args[i+1])
			}
			return id, true, nil
		case strings.HasPrefix(arg, "--id="):
			value := strings.TrimPrefix(arg, "--id=")
			id, err := parseInt(value)
			if err != nil {
				return 0, false, fmt.Errorf("invalid id: %s", value)
			}
			return id, true, nil
		case strings.HasPrefix(arg, "-id="):
			value := strings.TrimPrefix(arg, "-id=")
			id, err := parseInt(value)
			if err != nil {
				return 0, false, fmt.Errorf("invalid id: %s", value)
			}
			return id, true, nil
		}
	}

	return 0, false, nil
}

func splitComma(input string) []string {
	parts := strings.Split(input, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func prepareIssueUploads(ctx context.Context, client *api.Client, attachments []issueAttachmentInput) ([]api.IssueUpload, error) {
	if err := validateIssueAttachments(attachments); err != nil {
		return nil, err
	}

	uploads := make([]api.IssueUpload, 0, len(attachments))
	for _, attachment := range attachments {
		upload, err := uploadIssueAttachment(ctx, client, attachment)
		if err != nil {
			return nil, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

func validateIssueAttachments(attachments []issueAttachmentInput) error {
	for _, attachment := range attachments {
		file, err := os.Open(attachment.Path)
		if err != nil {
			return fmt.Errorf("attachment %q: %w", attachment.Path, err)
		}
		info, statErr := file.Stat()
		closeErr := file.Close()
		if statErr != nil {
			return fmt.Errorf("attachment %q: %w", attachment.Path, statErr)
		}
		if closeErr != nil {
			return fmt.Errorf("attachment %q: %w", attachment.Path, closeErr)
		}
		if info.IsDir() {
			return fmt.Errorf("attachment %q: is a directory", attachment.Path)
		}
	}
	return nil
}

func uploadIssueAttachment(ctx context.Context, client *api.Client, attachment issueAttachmentInput) (api.IssueUpload, error) {
	file, err := os.Open(attachment.Path)
	if err != nil {
		return api.IssueUpload{}, fmt.Errorf("attachment %q: %w", attachment.Path, err)
	}
	defer file.Close()

	filename := filepath.Base(attachment.Path)
	uploadResp, err := client.UploadAttachment(ctx, filename, file)
	if err != nil {
		return api.IssueUpload{}, fmt.Errorf("upload attachment %q: %w", attachment.Path, err)
	}

	upload := api.IssueUpload{
		Token:    uploadResp.Upload.Token,
		Filename: filename,
	}
	if description := strings.TrimSpace(attachment.Description); description != "" {
		upload.Description = description
	}
	return upload, nil
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func apiError(err error) int {
	var apiErr api.APIError
	if errors.As(err, &apiErr) {
		fmt.Fprintf(os.Stderr, "api error %d: %s\n", apiErr.StatusCode, apiErr.Body)
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}

type nameID struct {
	ID   int
	Name string
}

func resolveAssigneeID(ctx context.Context, client *api.Client, id optionalInt, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		if id.set {
			return id.value, nil
		}
		return 0, nil
	}

	users, err := client.ListUsers(ctx)
	if err != nil {
		return 0, err
	}

	needle := normalizeName(name)
	var matches []api.User
	for _, user := range users {
		if matchesUser(user, needle) {
			matches = append(matches, user)
		}
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("assignee not found: %s", name)
	}
	if len(matches) > 1 {
		return 0, fmt.Errorf("assignee matches multiple users: %s", name)
	}
	match := matches[0]
	if id.set && id.value != match.ID {
		return 0, fmt.Errorf("assignee-id does not match assignee name")
	}
	return match.ID, nil
}

func resolveStatusID(ctx context.Context, client *api.Client, id optionalInt, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		if id.set {
			return id.value, nil
		}
		return 0, nil
	}
	items, err := client.ListIssueStatuses(ctx)
	if err != nil {
		return 0, err
	}
	return resolveNameID(id, name, toNameIDsStatus(items), "status")
}

func resolvePriorityID(ctx context.Context, client *api.Client, id optionalInt, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		if id.set {
			return id.value, nil
		}
		return 0, nil
	}
	items, err := client.ListIssuePriorities(ctx)
	if err != nil {
		return 0, err
	}
	return resolveNameID(id, name, toNameIDsPriority(items), "priority")
}

func resolveTaskTypeID(ctx context.Context, client *api.Client, id optionalInt, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		if id.set {
			return id.value, nil
		}
		return 0, nil
	}
	items, err := client.ListTrackers(ctx)
	if err != nil {
		return 0, err
	}
	return resolveNameID(id, name, toNameIDsTracker(items), "task-type")
}

func resolveProjectID(ctx context.Context, client *api.Client, id optionalInt, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		if id.set {
			return id.value, nil
		}
		return 0, nil
	}
	items, err := client.ListProjects(ctx)
	if err != nil {
		return 0, err
	}
	return resolveNameID(id, name, toNameIDsProject(items), "project")
}

func resolveNameID(id optionalInt, name string, items []nameID, label string) (int, error) {
	needle := normalizeName(name)
	var matches []nameID
	for _, item := range items {
		if normalizeName(item.Name) == needle {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("%s not found: %s", label, name)
	}
	if len(matches) > 1 {
		return 0, fmt.Errorf("%s matches multiple entries: %s", label, name)
	}
	match := matches[0]
	if id.set && id.value != match.ID {
		return 0, fmt.Errorf("%s-id does not match %s name", label, label)
	}
	return match.ID, nil
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func matchesUser(user api.User, needle string) bool {
	if normalizeName(user.Login) == needle {
		return true
	}
	full := strings.TrimSpace(user.Firstname + " " + user.Lastname)
	if normalizeName(full) == needle {
		return true
	}
	return false
}

func toNameIDsStatus(items []api.IssueStatus) []nameID {
	result := make([]nameID, 0, len(items))
	for _, item := range items {
		result = append(result, nameID{ID: item.ID, Name: item.Name})
	}
	return result
}

func toNameIDsPriority(items []api.IssuePriority) []nameID {
	result := make([]nameID, 0, len(items))
	for _, item := range items {
		result = append(result, nameID{ID: item.ID, Name: item.Name})
	}
	return result
}

func toNameIDsTracker(items []api.Tracker) []nameID {
	result := make([]nameID, 0, len(items))
	for _, item := range items {
		result = append(result, nameID{ID: item.ID, Name: item.Name})
	}
	return result
}

func toNameIDsProject(items []api.Project) []nameID {
	result := make([]nameID, 0, len(items))
	for _, item := range items {
		result = append(result, nameID{ID: item.ID, Name: item.Name})
	}
	return result
}
