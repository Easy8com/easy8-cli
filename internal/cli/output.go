package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"easy8-cli/internal/api"
)

type outputBreadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

type outputEnvelope struct {
	OK          bool               `json:"ok"`
	Data        any                `json:"data,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Breadcrumbs []outputBreadcrumb `json:"breadcrumbs,omitempty"`
	Context     map[string]any     `json:"context,omitempty"`
}

func outputJSON(value any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		return 1
	}
	return 0
}

func outputJSONEnvelope(data any, summary string, breadcrumbs []outputBreadcrumb, context map[string]any) int {
	env := outputEnvelope{
		OK:          true,
		Data:        data,
		Summary:     summary,
		Breadcrumbs: breadcrumbs,
		Context:     context,
	}
	return outputJSON(env)
}

func outputIssues(issues []api.Issue) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSubject\tStatus\tAssignee\tUpdated\tSprint\tStory points\tCustom fields")
	for _, issue := range issues {
		status := nameOrEmpty(issue.Status)
		assignee := nameOrEmpty(issue.AssignedTo)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", issue.ID, issue.Subject, status, assignee, issue.UpdatedOn, sprintLabel(issue.EasySprint), jsonValueLabel(issue.EasyStoryPoints), customFieldsLabel(issue.CustomFields))
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		return 1
	}
	return 0
}

func outputIssueDetail(issue api.Issue) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", issue.ID)
	fmt.Fprintf(w, "Subject:\t%s\n", issue.Subject)
	fmt.Fprintf(w, "Project:\t%s\n", nameOrEmpty(issue.Project))
	fmt.Fprintf(w, "Tracker:\t%s\n", nameOrEmpty(issue.Tracker))
	fmt.Fprintf(w, "Status:\t%s\n", nameOrEmpty(issue.Status))
	fmt.Fprintf(w, "Priority:\t%s\n", nameOrEmpty(issue.Priority))
	if parent := parentLabel(issue.Parent); parent != "" {
		fmt.Fprintf(w, "Parent:\t%s\n", parent)
	}
	fmt.Fprintf(w, "Author:\t%s\n", nameOrEmpty(issue.Author))
	fmt.Fprintf(w, "Assignee:\t%s\n", nameOrEmpty(issue.AssignedTo))
	fmt.Fprintf(w, "Start date:\t%s\n", issue.StartDate)
	fmt.Fprintf(w, "Due date:\t%s\n", issue.DueDate)
	fmt.Fprintf(w, "Done:\t%d%%\n", issue.DoneRatio)
	fmt.Fprintf(w, "Created:\t%s\n", issue.CreatedOn)
	fmt.Fprintf(w, "Updated:\t%s\n", issue.UpdatedOn)
	if len(issue.EasySprint) > 0 {
		fmt.Fprintf(w, "Sprint:\t%s\n", sprintLabel(issue.EasySprint))
	}
	if len(issue.EasyStoryPoints) > 0 {
		fmt.Fprintf(w, "Story points:\t%s\n", jsonValueLabel(issue.EasyStoryPoints))
	}
	if len(issue.CustomFields) > 0 {
		fmt.Fprintln(w, "Custom fields:")
		for _, field := range issue.CustomFields {
			fmt.Fprintf(w, "  %s (#%d):\t%s\n", escapeFieldControls(field.Name), field.ID, jsonValueLabel(field.Value))
		}
	}
	if issue.Description != "" {
		fmt.Fprintf(w, "\nDescription:\n%s\n", issue.Description)
	}
	if len(issue.Attachments) > 0 {
		fmt.Fprintf(w, "\nAttachments:\n")
		for _, att := range issue.Attachments {
			author := nameOrEmpty(att.Author)
			label := att.Filename
			if att.Description != "" {
				label = fmt.Sprintf("%s (%s)", att.Filename, truncate(att.Description, 80))
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", label, formatFilesize(att.Filesize), author, att.CreatedOn)
		}
	}
	if len(issue.Journals) > 0 {
		fmt.Fprintf(w, "\nJournals:\n")
		for _, j := range issue.Journals {
			author := nameOrEmpty(j.User)
			fmt.Fprintf(w, "  #%d\t%s\t%s\n", j.ID, author, j.CreatedOn)
			for _, d := range j.Details {
				fmt.Fprintf(w, "    %s:\t%s -> %s\n", d.Name, truncate(d.OldValue, 40), truncate(d.NewValue, 40))
			}
			if j.Notes != "" {
				fmt.Fprintf(w, "    %s\n", truncate(j.Notes, 200))
			}
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		return 1
	}
	return 0
}

func outputPBIs(pbis []api.PBI) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tStatus\tEstimate\tAuthor\tUpdated")
	for _, pbi := range pbis {
		author := nameOrEmpty(pbi.Author)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", pbi.ID, pbi.Name, pbi.Status, pbi.Estimate, author, pbi.UpdatedAt)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		return 1
	}
	return 0
}

func outputPBIDetail(pbi api.PBI, issues []api.Issue) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%d\n", pbi.ID)
	fmt.Fprintf(w, "Name:\t%s\n", pbi.Name)
	fmt.Fprintf(w, "Status:\t%s\n", pbi.Status)
	fmt.Fprintf(w, "Estimate:\t%s\n", pbi.Estimate)
	fmt.Fprintf(w, "Board:\t%s\n", nameOrEmpty(pbi.Board))
	fmt.Fprintf(w, "Author:\t%s\n", nameOrEmpty(pbi.Author))
	fmt.Fprintf(w, "Created:\t%s\n", pbi.CreatedAt)
	fmt.Fprintf(w, "Updated:\t%s\n", pbi.UpdatedAt)
	if len(issues) > 0 {
		fmt.Fprintf(w, "\nIssues:\n")
		fmt.Fprintf(w, "  ID\tSubject\tStatus\tAssignee\tDone\n")
		for _, issue := range issues {
			status := nameOrEmpty(issue.Status)
			assignee := nameOrEmpty(issue.AssignedTo)
			fmt.Fprintf(w, "  #%d\t%s\t%s\t%s\t%d%%\n", issue.ID, issue.Subject, status, assignee, issue.DoneRatio)
		}
	} else if len(pbi.Issues) > 0 {
		// Fallback: show basic info if full details were not fetched
		fmt.Fprintf(w, "\nIssues:\n")
		for _, issue := range pbi.Issues {
			fmt.Fprintf(w, "  #%d\t%s\n", issue.ID, issue.Subject)
		}
	}
	if len(pbi.StickyNotes) > 0 {
		fmt.Fprintf(w, "\nSticky notes:\n")
		for _, note := range pbi.StickyNotes {
			fmt.Fprintf(w, "  %s\t[%s]\n", note.Name, note.Status)
		}
	}
	if pbi.Description != "" {
		fmt.Fprintf(w, "\nDescription:\n%s\n", pbi.Description)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "output error:", err)
		return 1
	}
	return 0
}

func nameOrEmpty(ref *api.NamedRef) string {
	if ref == nil {
		return ""
	}
	return ref.Name
}

func parentLabel(ref *api.NamedRef) string {
	if ref == nil {
		return ""
	}
	if ref.Name != "" && ref.ID != 0 {
		return fmt.Sprintf("#%d %s", ref.ID, ref.Name)
	}
	if ref.Name != "" {
		return ref.Name
	}
	if ref.ID != 0 {
		return fmt.Sprintf("#%d", ref.ID)
	}
	return ""
}

func formatFilesize(bytes int) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
