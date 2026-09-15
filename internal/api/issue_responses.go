package api

import (
	"encoding/json"
	"fmt"
)

// Keep the complete API document, including unknown nested fields and original
// JSON number/string/null types. Typed fields are a read-only human-output view.
func (response *IssueResponse) UnmarshalJSON(data []byte) error {
	type view IssueResponse
	var decoded view
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Issue.ID <= 0 {
		return fmt.Errorf("invalid issue response: missing positive issue id")
	}
	*response = IssueResponse(decoded)
	response.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (response IssueResponse) MarshalJSON() ([]byte, error) {
	if response.raw != nil {
		return response.raw, nil
	}
	type view IssueResponse
	return json.Marshal(view(response))
}

func (response *IssueListResponse) UnmarshalJSON(data []byte) error {
	type view IssueListResponse
	var decoded view
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if _, ok := fields["issues"]; !ok {
		return fmt.Errorf("invalid issue list response: missing issues")
	}
	if decoded.Issues == nil {
		return fmt.Errorf("invalid issue list response: issues must be an array, not null")
	}
	for index, issue := range decoded.Issues {
		if issue.ID <= 0 {
			return fmt.Errorf("invalid issue list response: entry %d must contain a positive issue id", index)
		}
	}
	*response = IssueListResponse(decoded)
	response.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (response IssueListResponse) MarshalJSON() ([]byte, error) {
	if response.raw != nil {
		return response.raw, nil
	}
	type view IssueListResponse
	return json.Marshal(view(response))
}
