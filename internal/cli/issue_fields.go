package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"unicode"

	"easy8-cli/internal/api"
)

type issueFieldFlags struct {
	sprintID         optionalInt
	storyPoints      optionalInt
	clearSprint      bool
	clearStoryPoints bool
	customFields     customFieldsFlag
}

func (fields *issueFieldFlags) register(fs *flag.FlagSet) {
	fs.Var(&fields.sprintID, "sprint-id", "Native sprint ID (positive integer)")
	fs.Var(&fields.storyPoints, "story-points", "Native story points (integer; zero is preserved)")
	fs.BoolVar(&fields.clearSprint, "clear-sprint", false, "Remove the sprint (send explicit null)")
	fs.BoolVar(&fields.clearStoryPoints, "clear-story-points", false, "Clear story points (send explicit null)")
	fs.Var(&fields.customFields, "custom-fields", `Custom field JSON array, e.g. [{"id":7,"value":"text"}]`)
}

func (fields issueFieldFlags) apply(input *api.IssueInput) error {
	if fields.sprintID.set && fields.clearSprint {
		return fmt.Errorf("--sprint-id and --clear-sprint cannot be used together")
	}
	if fields.storyPoints.set && fields.clearStoryPoints {
		return fmt.Errorf("--story-points and --clear-story-points cannot be used together")
	}
	if fields.sprintID.set {
		if fields.sprintID.value <= 0 {
			return fmt.Errorf("--sprint-id must be greater than 0")
		}
		input.EasySprintID = &api.NullableInt{Value: intPtr(fields.sprintID.value)}
	} else if fields.clearSprint {
		input.EasySprintID = &api.NullableInt{}
	}
	if fields.storyPoints.set {
		input.EasyStoryPoints = &api.NullableInt{Value: intPtr(fields.storyPoints.value)}
	} else if fields.clearStoryPoints {
		input.EasyStoryPoints = &api.NullableInt{}
	}
	if fields.customFields.set {
		input.CustomFields = &fields.customFields.values
	}
	return nil
}

type customFieldsFlag struct {
	set    bool
	values []api.CustomFieldInput
}

func (fields *customFieldsFlag) String() string {
	if !fields.set {
		return ""
	}
	data, _ := json.Marshal(fields.values)
	return string(data)
}

func (fields *customFieldsFlag) Set(value string) error {
	if fields.set {
		return fmt.Errorf("--custom-fields can only be supplied once")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &entries); err != nil {
		return fmt.Errorf("--custom-fields must be a JSON array of {id, value} objects: %w", err)
	}
	if entries == nil {
		return fmt.Errorf("--custom-fields must be an array, not null")
	}
	result := make([]api.CustomFieldInput, 0, len(entries))
	seen := make(map[int]bool)
	for _, entry := range entries {
		var id int
		if err := json.Unmarshal(entry["id"], &id); err != nil || id <= 0 {
			return fmt.Errorf("each custom field requires a positive integer id")
		}
		if len(entry) != 2 || entry["value"] == nil {
			return fmt.Errorf("each custom field must contain exactly id and value")
		}
		if seen[id] {
			return fmt.Errorf("duplicate custom field id %d", id)
		}
		seen[id] = true
		result = append(result, api.CustomFieldInput{ID: id, Value: entry["value"]})
	}
	fields.set = true
	fields.values = result
	return nil
}

// Human output unwraps strings; structured and null values stay legible JSON.
func jsonValueLabel(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	var text string
	if string(value) != "null" && json.Unmarshal(value, &text) == nil {
		return escapeFieldControls(text)
	}
	var compact bytes.Buffer
	if json.Compact(&compact, value) == nil {
		return escapeFieldControls(compact.String())
	}
	return escapeFieldControls(string(value))
}

func sprintLabel(value json.RawMessage) string {
	var sprint api.NamedRef
	if json.Unmarshal(value, &sprint) == nil && (sprint.ID != 0 || sprint.Name != "") {
		return escapeFieldControls(parentLabel(&sprint))
	}
	return jsonValueLabel(value)
}

func customFieldsLabel(fields []api.CustomField) string {
	labels := make([]string, 0, len(fields))
	for _, field := range fields {
		labels = append(labels, fmt.Sprintf("%s (#%d): %s", escapeFieldControls(field.Name), field.ID, jsonValueLabel(field.Value)))
	}
	return strings.Join(labels, "; ")
}

// New plugin/custom field columns must not inject table rows or terminal escapes.
func escapeFieldControls(text string) string {
	var escaped strings.Builder
	for _, char := range text {
		switch char {
		case '\n':
			escaped.WriteString(`\n`)
		case '\r':
			escaped.WriteString(`\r`)
		case '\t':
			escaped.WriteString(`\t`)
		default:
			if unicode.IsControl(char) {
				fmt.Fprintf(&escaped, `\u%04x`, char)
			} else {
				escaped.WriteRune(char)
			}
		}
	}
	return escaped.String()
}
