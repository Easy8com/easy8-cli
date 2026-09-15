package api

import "encoding/json"

const AutomationSourceEasy8CLI = "Easy8-CLI"

type NamedRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Issue struct {
	ID              int             `json:"id"`
	Subject         string          `json:"subject"`
	Description     string          `json:"description,omitempty"`
	DoneRatio       int             `json:"done_ratio,omitempty"`
	StartDate       string          `json:"start_date,omitempty"`
	DueDate         string          `json:"due_date,omitempty"`
	UpdatedOn       string          `json:"updated_on,omitempty"`
	CreatedOn       string          `json:"created_on,omitempty"`
	Project         *NamedRef       `json:"project,omitempty"`
	Tracker         *NamedRef       `json:"tracker,omitempty"`
	Status          *NamedRef       `json:"status,omitempty"`
	Priority        *NamedRef       `json:"priority,omitempty"`
	Parent          *NamedRef       `json:"parent,omitempty"`
	Author          *NamedRef       `json:"author,omitempty"`
	AssignedTo      *NamedRef       `json:"assigned_to,omitempty"`
	Journals        []Journal       `json:"journals,omitempty"`
	Attachments     []Attachment    `json:"attachments,omitempty"`
	EasySprint      json.RawMessage `json:"easy_sprint,omitempty"`
	EasyStoryPoints json.RawMessage `json:"easy_story_points,omitempty"`
	CustomFields    []CustomField   `json:"custom_fields,omitempty"`
}

type CustomField struct {
	ID    int             `json:"id"`
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

type CustomFieldInput struct {
	ID    int             `json:"id"`
	Value json.RawMessage `json:"value"`
}

// A nil *NullableInt omits a field; a nonnil wrapper with nil Value sends null.
type NullableInt struct {
	Value *int
}

func (value NullableInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(value.Value)
}

type Journal struct {
	ID           int             `json:"id"`
	User         *NamedRef       `json:"user,omitempty"`
	Notes        string          `json:"notes"`
	CreatedOn    string          `json:"created_on"`
	PrivateNotes bool            `json:"private_notes"`
	Details      []JournalDetail `json:"details"`
}

type JournalDetail struct {
	Property string `json:"property"`
	Name     string `json:"name"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

type Attachment struct {
	ID          int       `json:"id"`
	Filename    string    `json:"filename"`
	Filesize    int       `json:"filesize"`
	ContentType string    `json:"content_type"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	ContentURL  string    `json:"content_url"`
	Author      *NamedRef `json:"author,omitempty"`
	CreatedOn   string    `json:"created_on"`
}

type IssueUpload struct {
	Token       string `json:"token"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
}

type Upload struct {
	Token string `json:"token"`
}

type UploadResponse struct {
	Upload Upload `json:"upload"`
}

type IssueInput struct {
	Subject          *string             `json:"subject,omitempty"`
	ProjectID        *int                `json:"project_id,omitempty"`
	TrackerID        *int                `json:"tracker_id,omitempty"`
	StatusID         *int                `json:"status_id,omitempty"`
	PriorityID       *int                `json:"priority_id,omitempty"`
	ParentIssueID    *int                `json:"parent_issue_id,omitempty"`
	AuthorID         *int                `json:"author_id,omitempty"`
	AssignedToID     *int                `json:"assigned_to_id,omitempty"`
	Description      *string             `json:"description,omitempty"`
	StartDate        *string             `json:"start_date,omitempty"`
	DueDate          *string             `json:"due_date,omitempty"`
	DoneRatio        *int                `json:"done_ratio,omitempty"`
	Notes            *string             `json:"notes,omitempty"`
	AutomationSource *string             `json:"automation_source,omitempty"`
	Uploads          []IssueUpload       `json:"uploads,omitempty"`
	EasySprintID     *NullableInt        `json:"easy_sprint_id,omitempty"`
	EasyStoryPoints  *NullableInt        `json:"easy_story_points,omitempty"`
	CustomFields     *[]CustomFieldInput `json:"custom_fields,omitempty"`
}

type IssueRequest struct {
	Issue IssueInput `json:"issue"`
}

type IssueResponse struct {
	Issue Issue `json:"issue"`
	raw   json.RawMessage
}

type IssueListResponse struct {
	Issues     []Issue `json:"issues"`
	TotalCount int     `json:"total_count"`
	Offset     int     `json:"offset"`
	Limit      int     `json:"limit"`
	raw        json.RawMessage
}

type Tracker struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TrackerListResponse struct {
	Trackers []Tracker `json:"trackers"`
}

type IssueStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IssueStatusListResponse struct {
	IssueStatuses []IssueStatus `json:"issue_statuses"`
}

type IssuePriority struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IssuePriorityListResponse struct {
	IssuePriorities []IssuePriority `json:"issue_priorities"`
}

type User struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type UserListResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"total_count"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProjectListResponse struct {
	Projects   []Project `json:"projects"`
	TotalCount int       `json:"total_count"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
}
