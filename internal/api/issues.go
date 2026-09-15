package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type IssueListParams struct {
	Limit       int
	Offset      int
	Sort        string
	Query       string
	Include     []string
	IssueIDs    []int
	AssigneeID  int
	DueDate     string
	StatusID    int
	PriorityID  int
	Subject     string
	TaskTypeID  int
	ProjectID   int
	SprintID    int
	AllStatuses bool
}

func (c *Client) ListIssues(ctx context.Context, params IssueListParams) (IssueListResponse, error) {
	if params.SprintID < 0 {
		return IssueListResponse{}, fmt.Errorf("sprint id must be positive")
	}
	if params.AllStatuses && params.StatusID != 0 {
		return IssueListResponse{}, fmt.Errorf("all statuses and status id cannot be combined")
	}
	query := url.Values{}
	hasFilter := false
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	if strings.TrimSpace(params.Sort) != "" {
		query.Set("sort", params.Sort)
	}
	if strings.TrimSpace(params.Query) != "" {
		query.Set("set_filter", "1")
		hasFilter = true
		query.Set("easy_query_q", params.Query)
	}
	if params.AssigneeID > 0 {
		query.Set("assigned_to_id", strconv.Itoa(params.AssigneeID))
		hasFilter = true
	}
	if strings.TrimSpace(params.DueDate) != "" {
		query.Set("due_date", params.DueDate)
		hasFilter = true
	}
	if params.StatusID > 0 {
		query.Set("status_id", strconv.Itoa(params.StatusID))
		hasFilter = true
	}
	if params.AllStatuses {
		query.Set("status_id", "*")
		hasFilter = true
	}
	if params.SprintID > 0 {
		query.Set("easy_sprint_id", strconv.Itoa(params.SprintID))
		if params.StatusID == 0 && !params.AllStatuses {
			query.Set("status_id", "o")
		}
		hasFilter = true
	}
	if params.PriorityID > 0 {
		query.Set("priority_id", strconv.Itoa(params.PriorityID))
		hasFilter = true
	}
	if strings.TrimSpace(params.Subject) != "" {
		query.Set("subject", params.Subject)
		hasFilter = true
	}
	if params.TaskTypeID > 0 {
		query.Set("tracker_id", strconv.Itoa(params.TaskTypeID))
		hasFilter = true
	}
	if params.ProjectID > 0 {
		query.Set("project_id", strconv.Itoa(params.ProjectID))
		hasFilter = true
	}
	if len(params.IssueIDs) > 0 {
		ids := make([]string, len(params.IssueIDs))
		for i, id := range params.IssueIDs {
			ids[i] = strconv.Itoa(id)
		}
		query.Set("issue_id", strings.Join(ids, "|"))
		hasFilter = true
	}
	if hasFilter {
		query.Set("set_filter", "1")
	}
	if len(params.Include) > 0 {
		query.Set("include", strings.Join(params.Include, ","))
	}

	var resp IssueListResponse
	if err := c.doJSON(ctx, "GET", "/issues.json", query, nil, &resp); err != nil {
		return IssueListResponse{}, err
	}
	if resp.raw == nil {
		return IssueListResponse{}, fmt.Errorf("empty issue list response")
	}
	if params.SprintID > 0 {
		for _, issue := range resp.Issues {
			if err := issue.verifySprintFilter(params.SprintID); err != nil {
				return IssueListResponse{}, err
			}
		}
	}
	return resp, nil
}

func (c *Client) GetIssue(ctx context.Context, id int, include []string) (IssueResponse, error) {
	if id == 0 {
		return IssueResponse{}, fmt.Errorf("missing issue id")
	}
	path := fmt.Sprintf("/issues/%d.json", id)
	var query url.Values
	if len(include) > 0 {
		query = url.Values{}
		query.Set("include", strings.Join(include, ","))
	}
	var resp IssueResponse
	if err := c.doJSON(ctx, "GET", path, query, nil, &resp); err != nil {
		return IssueResponse{}, err
	}
	if resp.Issue.ID != id {
		return IssueResponse{}, fmt.Errorf("invalid issue response: expected issue #%d, got #%d", id, resp.Issue.ID)
	}
	return resp, nil
}

func (c *Client) UploadAttachment(ctx context.Context, filename string, body io.Reader) (UploadResponse, error) {
	if c.APIKey == "" {
		return UploadResponse{}, fmt.Errorf("missing API key")
	}
	if strings.TrimSpace(filename) == "" {
		return UploadResponse{}, fmt.Errorf("missing attachment filename")
	}

	query := url.Values{}
	query.Set("filename", filename)
	urlValue := c.BaseURL + "/uploads.json?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlValue, body)
	if err != nil {
		return UploadResponse{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Redmine-API-Key", c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return UploadResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return UploadResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return UploadResponse{}, APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody)), URL: urlValue}
	}

	var uploadResp UploadResponse
	if len(respBody) == 0 {
		return uploadResp, nil
	}
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return UploadResponse{}, err
	}
	return uploadResp, nil
}

func (c *Client) CreateIssue(ctx context.Context, input IssueInput) (IssueResponse, error) {
	var resp IssueResponse
	request := IssueRequest{Issue: input}
	if err := c.doJSON(ctx, "POST", "/issues.json", nil, request, &resp); err != nil {
		return IssueResponse{}, err
	}
	if resp.Issue.ID == 0 {
		return IssueResponse{}, fmt.Errorf("empty issue create response")
	}
	return resp, nil
}

func (c *Client) UpdateIssue(ctx context.Context, id int, input IssueInput) (IssueResponse, error) {
	if id == 0 {
		return IssueResponse{}, fmt.Errorf("missing issue id")
	}
	path := fmt.Sprintf("/issues/%d.json", id)
	var resp IssueResponse
	automationSource := AutomationSourceEasy8CLI
	input.AutomationSource = &automationSource
	request := IssueRequest{Issue: input}
	if err := c.doJSON(ctx, "PUT", path, nil, request, &resp); err != nil {
		return IssueResponse{}, err
	}
	if resp.Issue.ID != 0 && resp.Issue.ID != id {
		return IssueResponse{}, fmt.Errorf("invalid issue update response: expected issue #%d, got #%d; the HTTP update may have succeeded partially; no rollback was performed", id, resp.Issue.ID)
	}
	return resp, nil
}
