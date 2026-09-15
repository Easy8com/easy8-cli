package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestAgileWriteResultsAreVerified(t *testing.T) {
	cases := []struct {
		name           string
		flags          []string
		responseFields string
		ok             bool
		field          string
	}{
		{"matching points number", []string{"--story-points", "5"}, `,"easy_story_points":5`, true, ""},
		{"matching points string", []string{"--story-points", "5"}, `,"easy_story_points":"5"`, true, ""},
		{"zero number", []string{"--story-points", "0"}, `,"easy_story_points":0`, true, ""},
		{"zero string", []string{"--story-points", "0"}, `,"easy_story_points":"0"`, true, ""},
		{"clear points", []string{"--clear-story-points"}, `,"easy_story_points":null`, true, ""},
		{"clear sprint", []string{"--clear-sprint"}, `,"easy_sprint":null`, true, ""},
		{"matching sprint", []string{"--sprint-id", "34"}, `,"easy_sprint":{"id":34,"name":"September","unknown":{"n":9007199254740993}}`, true, ""},
		{"ignored points", []string{"--story-points", "5"}, `,"easy_story_points":"3"`, false, "easy_story_points"},
		{"inaccessible points", []string{"--story-points", "5"}, ``, false, "easy_story_points"},
		{"null is not zero", []string{"--story-points", "0"}, `,"easy_story_points":null`, false, "easy_story_points"},
		{"false is not zero", []string{"--story-points", "0"}, `,"easy_story_points":false`, false, "easy_story_points"},
		{"fraction is not integer", []string{"--story-points", "5"}, `,"easy_story_points":5.5`, false, "easy_story_points"},
		{"fraction string", []string{"--story-points", "5"}, `,"easy_story_points":"5.0"`, false, "easy_story_points"},
		{"ignored clear points", []string{"--clear-story-points"}, `,"easy_story_points":0`, false, "easy_story_points"},
		{"absent clear points", []string{"--clear-story-points"}, ``, false, "easy_story_points"},
		{"empty string is not null", []string{"--clear-story-points"}, `,"easy_story_points":""`, false, "easy_story_points"},
		{"ignored sprint", []string{"--sprint-id", "34"}, `,"easy_sprint":{"id":35}`, false, "easy_sprint"},
		{"sprint null", []string{"--sprint-id", "34"}, `,"easy_sprint":null`, false, "easy_sprint"},
		{"sprint inaccessible", []string{"--sprint-id", "34"}, ``, false, "easy_sprint"},
		{"sprint missing id", []string{"--sprint-id", "34"}, `,"easy_sprint":{"name":"Hidden"}`, false, "easy_sprint"},
		{"sprint id wrong type", []string{"--sprint-id", "34"}, `,"easy_sprint":{"id":"34"}`, false, "easy_sprint"},
		{"ignored clear sprint", []string{"--clear-sprint"}, `,"easy_sprint":{"id":34}`, false, "easy_sprint"},
		{"absent clear sprint", []string{"--clear-sprint"}, ``, false, "easy_sprint"},
		{"partial update", []string{"--story-points", "5", "--sprint-id", "34"}, `,"easy_story_points":5,"easy_sprint":{"id":35}`, false, "easy_sprint"},
	}
	for _, action := range []string{"create", "update", "readback"} {
		for _, mode := range []string{"--quiet", "--json"} {
			for _, test := range cases {
				t.Run(action+mode+"/"+test.name, func(t *testing.T) {
					body := `{"issue":{"id":101,"subject":"Changed"` + test.responseFields + `},"unknown_top":[]}`
					writes := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost || r.Method == http.MethodPut {
							writes++
						}
						if action == "readback" && r.Method == http.MethodPut {
							w.WriteHeader(http.StatusNoContent)
							return
						}
						_, _ = io.WriteString(w, body)
					}))
					defer server.Close()
					setTestEnv(t, server.URL)
					args := []string{"issue", "update", "101", "--subject", "Changed"}
					if action == "create" {
						args = createIssueArgs()
					}
					args = append(args, test.flags...)
					stdout, stderr, code := captureRun(t, append(args, mode))
					if writes != 1 {
						t.Fatalf("expected one write, got %d", writes)
					}
					if test.ok {
						if code != 0 {
							t.Fatalf("code=%d stderr=%s", code, stderr)
						}
						if mode == "--json" {
							stdout = string(decodeEnvelope(t, stdout).Data)
						}
						assertSameJSON(t, body, stdout)
					} else {
						if code != 1 || stdout != "" {
							t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
						}
						for _, text := range []string{test.field, "could not be verified", "may have succeeded partially", "no rollback was performed"} {
							if !strings.Contains(stderr, text) {
								t.Errorf("missing %q in %s", text, stderr)
							}
						}
					}
				})
			}
		}
	}
}

func TestSprintFilterRejectsBroadenedResults(t *testing.T) {
	for _, field := range []string{``, `,"easy_sprint":null`, `,"easy_sprint":{}`, `,"easy_sprint":{"id":35}`, `,"easy_sprint_id":34`} {
		for _, command := range []string{"list", "search"} {
			t.Run(command+field, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.WriteString(w, `{"issues":[{"id":101,"easy_sprint":{"id":34}},{"id":102`+field+`}]}`)
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				stdout, stderr, code := captureRun(t, []string{"issue", command, "--sprint-id", "34", "--quiet"})
				if code != 1 || stdout != "" || !strings.Contains(stderr, "sprint filter") {
					t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
				}
			})
		}
	}
}

func TestSprintFilterStatusDefaults(t *testing.T) {
	cases := []struct {
		args     []string
		expected string
	}{
		{[]string{"issue", "list", "--sprint-id", "34"}, "o"},
		{[]string{"issue", "search", "--sprint-id", "34"}, "o"},
		{[]string{"issue", "list", "--sprint-id", "34", "--all-statuses"}, "*"},
		{[]string{"issue", "search", "--sprint-id", "34", "--status-id", "2"}, "2"},
		{[]string{"issue", "search", "--sprint-id", "34", "--status", "Closed"}, "2"},
		{[]string{"issue", "search", "--q", "Existing"}, ""},
	}
	for _, test := range cases {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/issue_statuses.json" {
					_, _ = io.WriteString(w, `{"issue_statuses":[{"id":2,"name":"Closed"}]}`)
					return
				}
				if got := r.URL.Query().Get("status_id"); got != test.expected {
					t.Errorf("status_id=%q want %q", got, test.expected)
				}
				_, _ = io.WriteString(w, `{"issues":[{"id":101,"easy_sprint":{"id":34}}]}`)
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			_, stderr, code := captureRun(t, append(test.args, "--quiet"))
			if code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
		})
	}
}

func TestIssueListRejectsInvalidEntries(t *testing.T) {
	for _, entries := range []string{`null`, `[null]`, `[{}]`, `[{"id":0}]`, `[{"id":-1}]`, `[{"id":101},null]`, `[{"id":101},{}]`} {
		t.Run(entries, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"issues":`+entries+`}`) }))
			defer server.Close()
			setTestEnv(t, server.URL)
			stdout, stderr, code := captureRun(t, []string{"issue", "list", "--quiet"})
			if code != 1 || stdout != "" || !strings.Contains(stderr, "invalid issue list response") {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}

func TestIssueIdentityMismatchNeverReportsSuccess(t *testing.T) {
	for _, action := range []string{"show", "update", "readback"} {
		for _, mode := range []string{"--quiet", "--json", ""} {
			t.Run(action+mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if action == "readback" && r.Method == http.MethodPut {
						w.WriteHeader(http.StatusOK)
						return
					}
					_, _ = io.WriteString(w, `{"issue":{"id":102}}`)
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				command := action
				if action == "readback" {
					command = "update"
				}
				args := []string{"issue", command, "101"}
				if mode != "" {
					args = append(args, mode)
				}
				stdout, stderr, code := captureRun(t, args)
				if code != 1 || stdout != "" || !strings.Contains(stderr, "expected issue #101, got #102") {
					t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
				}
				if action != "show" && !strings.Contains(stderr, "may have succeeded") {
					t.Errorf("missing partial write context: %s", stderr)
				}
				if action == "readback" && !strings.Contains(stderr, "readback failed") {
					t.Errorf("missing readback context: %s", stderr)
				}
			})
		}
	}
}

func TestExistingKnownIntegerTypeRemainsStrict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"issue":{"id":101,"done_ratio":"0"}}`)
	}))
	defer server.Close()
	setTestEnv(t, server.URL)
	stdout, stderr, code := captureRun(t, []string{"issue", "show", "101", "--quiet"})
	if code != 1 || stdout != "" || !strings.Contains(stderr, "done_ratio") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestHumanPluginFieldsEscapeControlsAndKeepMachineJSON(t *testing.T) {
	issue := `{"id":101,"subject":"Planning","easy_story_points":0,"easy_sprint":{"id":34,"name":"Sprint\n\t\u001b[31m"},"custom_fields":[{"id":7,"name":"Field\n\t\u001b","value":"Value\n\t\u001b"}]}`
	for _, command := range []string{"show", "list"} {
		t.Run(command, func(t *testing.T) {
			body := `{"issue":` + issue + `}`
			args := []string{"issue", command, "101"}
			if command == "list" {
				body = `{"issues":[` + issue + `]}`
				args = []string{"issue", command}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
			defer server.Close()
			setTestEnv(t, server.URL)
			stdout, stderr, code := captureRun(t, args)
			if code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
			for _, label := range []string{`Sprint\n\t\u001b[31m`, `Field\n\t\u001b`, `Value\n\t\u001b`} {
				if !strings.Contains(stdout, label) {
					t.Errorf("missing escaped %q: %s", label, stdout)
				}
			}
			if strings.ContainsAny(stdout, "\x1b\t") {
				t.Errorf("raw controls in output: %q", stdout)
			}
			if command == "list" && len(strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")) != 2 {
				t.Errorf("injected row: %q", stdout)
			}
			for _, mode := range []string{"--quiet", "--json"} {
				machine, stderr, code := captureRun(t, append(args, mode))
				if code != 0 {
					t.Fatalf("code=%d stderr=%s", code, stderr)
				}
				if mode == "--json" {
					machine = string(decodeEnvelope(t, machine).Data)
				}
				assertSameJSON(t, body, machine)
			}
		})
	}
}

func TestHumanStoryPointsZeroNullAndAbsent(t *testing.T) {
	for _, test := range []struct {
		field, label string
		present      bool
	}{
		{`,"easy_story_points":0`, "0", true},
		{`,"easy_story_points":"0"`, "0", true},
		{`,"easy_story_points":null`, "null", true},
		{"", "", false},
	} {
		for _, command := range []string{"show", "list"} {
			t.Run(command+test.field, func(t *testing.T) {
				issue := `{"id":101,"subject":"Planning","status":{"id":1,"name":"New"},"assigned_to":{"id":2,"name":"User"},"updated_on":"2026-09-15","easy_sprint":{"id":34,"name":"Sprint"},"custom_fields":[{"id":7,"name":"Field","value":"Value"}]` + test.field + `}`
				body := `{"issue":` + issue + `}`
				args := []string{"issue", command, "101"}
				if command == "list" {
					body = `{"issues":[` + issue + `]}`
					args = []string{"issue", command}
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
				defer server.Close()
				setTestEnv(t, server.URL)
				stdout, stderr, code := captureRun(t, args)
				if code != 0 {
					t.Fatalf("code=%d stderr=%s", code, stderr)
				}
				if command == "show" {
					match := regexp.MustCompile(`(?m)^Story points: +([^\n]*)$`).FindStringSubmatch(stdout)
					if test.present {
						if len(match) != 2 || strings.TrimSpace(match[1]) != test.label {
							t.Fatalf("wrong points line: %q", stdout)
						}
					} else if len(match) > 0 {
						t.Fatalf("absent points printed: %q", stdout)
					}
				} else {
					lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
					if len(lines) != 2 {
						t.Fatalf("wrong rows: %q", stdout)
					}
					start, end := strings.Index(lines[0], "Story points"), strings.Index(lines[0], "Custom fields")
					if start < 0 || end <= start || len(lines[1]) < end {
						t.Fatalf("invalid table positions: %q", stdout)
					}
					if got := strings.TrimSpace(lines[1][start:end]); got != test.label {
						t.Fatalf("actual points cell=%q want %q in %s", got, test.label, stdout)
					}
				}
			})
		}
	}
}

func TestEmptyIssueListRemainsValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, `{"issues":[],"total_count":0}`) }))
	defer server.Close()
	setTestEnv(t, server.URL)
	stdout, stderr, code := captureRun(t, []string{"issue", "list", "--quiet"})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	assertSameJSON(t, `{"issues":[],"total_count":0}`, stdout)
}
