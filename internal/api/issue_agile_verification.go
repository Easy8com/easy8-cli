package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// VerifyAgileFields checks only explicitly requested native agile fields. The
// Easy8 agile Swagger declares points as string, while the model is integer.
// Verification accepts those two integer representations without modifying raw
// response data or relaxing the types of any other known REST fields.
func (issue Issue) VerifyAgileFields(input IssueInput) error {
	if input.EasyStoryPoints != nil {
		actual := bytes.TrimSpace(issue.EasyStoryPoints)
		if len(actual) == 0 {
			return fmt.Errorf("easy_story_points is missing or inaccessible")
		}
		if input.EasyStoryPoints.Value == nil {
			if !bytes.Equal(actual, []byte("null")) {
				return fmt.Errorf("easy_story_points did not return the requested null")
			}
		} else {
			points, err := storyPointsInteger(actual)
			if err != nil || points != *input.EasyStoryPoints.Value {
				return fmt.Errorf("easy_story_points does not match the requested integer")
			}
		}
	}
	if input.EasySprintID != nil {
		actual := bytes.TrimSpace(issue.EasySprint)
		if len(actual) == 0 {
			return fmt.Errorf("easy_sprint is missing or inaccessible")
		}
		if input.EasySprintID.Value == nil {
			if !bytes.Equal(actual, []byte("null")) {
				return fmt.Errorf("easy_sprint did not return the requested null")
			}
		} else {
			id, err := sprintObjectID(actual)
			if err != nil || id != *input.EasySprintID.Value {
				return fmt.Errorf("easy_sprint does not match the requested sprint ID")
			}
		}
	}
	return nil
}

func storyPointsInteger(raw json.RawMessage) (int, error) {
	if bytes.Equal(raw, []byte("null")) {
		return 0, fmt.Errorf("points are null")
	}
	if len(raw) > 0 && raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, err
		}
		return strconv.Atoi(value)
	}
	var value int
	err := json.Unmarshal(raw, &value)
	return value, err
}

func sprintObjectID(raw json.RawMessage) (int, error) {
	var sprint struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(raw, &sprint); err != nil {
		return 0, err
	}
	if sprint.ID <= 0 {
		return 0, fmt.Errorf("missing positive sprint ID")
	}
	return sprint.ID, nil
}

func (issue Issue) verifySprintFilter(id int) error {
	actual, err := sprintObjectID(issue.EasySprint)
	if err != nil {
		return fmt.Errorf("cannot verify sprint filter for issue #%d: sprint metadata is missing, inaccessible or invalid", issue.ID)
	}
	if actual != id {
		return fmt.Errorf("sprint filter was not respected: issue #%d belongs to sprint #%d, requested #%d", issue.ID, actual, id)
	}
	return nil
}
