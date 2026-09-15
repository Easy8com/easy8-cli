package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func createIssueArgs() []string {
	return []string{"issue", "create", "--subject", "Planning", "--project-id", "1", "--tracker-id", "1", "--status-id", "1", "--priority-id", "1", "--author-id", "1", "--assigned-to-id", "2"}
}

func assertSameJSON(t *testing.T, expected, actual string) {
	t.Helper()
	decode := func(value string) any {
		decoder := json.NewDecoder(strings.NewReader(value))
		decoder.UseNumber()
		var result any
		if err := decoder.Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v (%s)", err, value)
		}
		return result
	}
	if !reflect.DeepEqual(decode(expected), decode(actual)) {
		t.Fatalf("JSON changed\nexpected: %s\nactual: %s", expected, actual)
	}
}

func TestIssueMachineOutputPreservesCompleteAPIResponse(t *testing.T) {
	for _, points := range []string{`0`, `null`, `"8"`, `8`} {
		for _, mode := range []string{"--quiet", "--json"} {
			for _, command := range []string{"show", "list", "search", "create", "update"} {
				t.Run(command+mode+points, func(t *testing.T) {
					issue := fmt.Sprintf(`{"id":101,"subject":"Planning","description":null,"done_ratio":0,"easy_story_points":%s,"easy_sprint":{"id":34,"name":"September","plugin":{"zero":0,"nil":null,"array":[]}},"custom_fields":[{"id":7,"name":"Complex","value":[null,0,"0",false,{"n":9007199254740993}],"unknown":true}],"project":{"id":1,"name":"A","unknown":{"a":[1,2]}},"journals":[{"id":2,"details":[{"property":"attr","name":"x","old_value":null,"new_value":"0","extra":[{"id":3}]}],"unknown":false}],"attachments":[],"plugin":{"huge":9007199254740993,"nested":[{"x":null}],"bool":false}}`, points)
					body := `{"issue":` + issue + `,"unknown_top":{"array":[],"nil":null}}`
					args := []string{"issue", command, "101"}
					if command == "list" || command == "search" {
						body = `{"issues":[` + issue + `],"total_count":1,"offset":0,"limit":25,"unknown_top":{"cursor":null}}`
						args = []string{"issue", command, "--sprint-id", "34"}
					} else if command == "create" {
						args = createIssueArgs()
					}
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, body)
					}))
					defer server.Close()
					setTestEnv(t, server.URL)
					stdout, stderr, code := captureRun(t, append(args, mode))
					if code != 0 {
						t.Fatalf("code=%d stderr=%s", code, stderr)
					}
					if mode == "--json" {
						envelope := decodeEnvelope(t, stdout)
						if !envelope.OK {
							t.Fatal("envelope did not indicate success")
						}
						stdout = string(envelope.Data)
					}
					assertSameJSON(t, body, stdout)
				})
			}
		}
	}
}

func TestIssuePlanningWritesArePartialAndNullable(t *testing.T) {
	cases := []struct {
		name   string
		flags  []string
		fields string
	}{
		{"omitted", nil, `{}`},
		{"set", []string{"--sprint-id", "34", "--story-points", "5"}, `{"easy_sprint_id":34,"easy_story_points":5}`},
		{"zero", []string{"--story-points", "0"}, `{"easy_story_points":0}`},
		{"clear", []string{"--clear-sprint", "--clear-story-points"}, `{"easy_sprint_id":null,"easy_story_points":null}`},
		{"negative estimate", []string{"--story-points=-1"}, `{"easy_story_points":-1}`},
		{"custom fields", []string{"--custom-fields", `[{"id":7,"value":[null,0,"a",{"n":9007199254740993}]},{"id":8,"value":null}]`}, `{"custom_fields":[{"id":7,"value":[null,0,"a",{"n":9007199254740993}]},{"id":8,"value":null}]}`},
		{"empty custom fields", []string{"--custom-fields", `[]`}, `{"custom_fields":[]}`},
	}
	for _, command := range []string{"create", "update"} {
		for _, test := range cases {
			t.Run(command+"/"+test.name, func(t *testing.T) {
				var request map[string]map[string]json.RawMessage
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
					}
					result := map[string]any{"id": 101, "subject": "Planning"}
					if points, ok := request["issue"]["easy_story_points"]; ok {
						result["easy_story_points"] = points
					}
					if sprint, ok := request["issue"]["easy_sprint_id"]; ok {
						result["easy_sprint"] = nil
						if string(sprint) != "null" {
							result["easy_sprint"] = map[string]any{"id": sprint}
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"issue": result})
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				args := []string{"issue", "update", "101"}
				if command == "create" {
					args = createIssueArgs()
				}
				args = append(args, test.flags...)
				_, stderr, code := captureRun(t, append(args, "--quiet"))
				if code != 0 {
					t.Fatalf("code=%d stderr=%s", code, stderr)
				}
				fields := request["issue"]
				if command == "update" {
					assertSameJSON(t, `"Easy8-CLI"`, string(fields["automation_source"]))
					delete(fields, "automation_source")
				} else {
					for _, key := range []string{"subject", "project_id", "tracker_id", "status_id", "priority_id", "author_id", "assigned_to_id"} {
						if _, exists := fields[key]; !exists {
							t.Errorf("missing required create field %s", key)
						}
						delete(fields, key)
					}
				}
				body, err := json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				assertSameJSON(t, test.fields, string(body))
			})
		}
	}
}

func TestInvalidPlanningFlagsFailBeforeHTTP(t *testing.T) {
	flags := [][]string{
		{"--sprint-id", "1", "--clear-sprint"},
		{"--story-points", "0", "--clear-story-points"},
		{"--sprint-id", "0"}, {"--sprint-id=-1"}, {"--story-points", "1.5"}, {"--story-points", "true"},
	}
	for _, value := range []string{"", "null", `{}`, `[{"id":"1","value":0}]`, `[{"id":1.5,"value":0}]`, `[null]`, `[{"id":1}]`, `[{"id":0,"value":1}]`, `[{"id":1,"value":0,"subject":"injected"}]`, `[{"id":1,"value":0},{"id":1,"value":2}]`, `[] trailing`} {
		flags = append(flags, []string{"--custom-fields", value})
	}
	for _, command := range []string{"create", "update"} {
		for _, invalid := range flags {
			t.Run(command+strings.Join(invalid, " "), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Error("invalid flags caused HTTP request")
					w.WriteHeader(http.StatusInternalServerError)
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				args := []string{"issue", "update", "101"}
				if command == "create" {
					args = createIssueArgs()
				}
				stdout, stderr, code := captureRun(t, append(args, invalid...))
				if code != 2 || stdout != "" || stderr == "" {
					t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
				}
			})
		}
	}
}

func TestIssueSprintFilters(t *testing.T) {
	for _, command := range []string{"list", "search"} {
		t.Run(command, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				for key, expected := range map[string]string{"easy_sprint_id": "34", "set_filter": "1", "status_id": "*", "limit": "2", "offset": "3"} {
					if query.Get(key) != expected {
						t.Errorf("%s=%q, want %q", key, query.Get(key), expected)
					}
				}
				_, _ = io.WriteString(w, `{"issues":[],"offset":3,"limit":2,"total_count":0}`)
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			_, stderr, code := captureRun(t, []string{"issue", command, "--sprint-id", "34", "--all-statuses", "--limit", "2", "--offset", "3", "--quiet"})
			if code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
		})
	}
}

func TestInvalidSprintFiltersFailBeforeHTTP(t *testing.T) {
	for _, args := range [][]string{
		{"issue", "list", "--sprint-id", "0"},
		{"issue", "search", "--sprint-id=-1"},
		{"issue", "search", "--sprint-id", "34", "--all-statuses", "--status", "New"},
		{"issue", "search", "--sprint-id", "34", "--all-statuses", "--status-id", "1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid filters caused HTTP request")
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			stdout, stderr, code := captureRun(t, args)
			if code != 2 || stdout != "" || stderr == "" {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}

func TestIssueUpdateEmptyResponseReadback(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		for _, mode := range []string{"--quiet", "--json"} {
			t.Run(fmt.Sprintf("%d%s", status, mode), func(t *testing.T) {
				methods := []string{}
				body := `{"issue":{"id":101,"easy_sprint":null,"easy_story_points":0,"custom_fields":[],"unknown":{"n":9007199254740993}}}`
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					methods = append(methods, r.Method)
					if r.Method == http.MethodPut {
						w.WriteHeader(status)
						return
					}
					_, _ = io.WriteString(w, body)
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				stdout, stderr, code := captureRun(t, []string{"issue", "update", "101", "--clear-sprint", "--story-points", "0", mode})
				if code != 0 || strings.Join(methods, ",") != "PUT,GET" {
					t.Fatalf("code=%d methods=%v stderr=%s", code, methods, stderr)
				}
				if mode == "--json" {
					stdout = string(decodeEnvelope(t, stdout).Data)
				}
				assertSameJSON(t, body, stdout)
			})
		}
	}
}

func TestIssueResponseErrorsStayFailures(t *testing.T) {
	for _, command := range []string{"show", "list", "search", "create", "update", "readback"} {
		for _, body := range []string{"not json", `null`, `{}`, `{"issue":`, `{"errors":["invalid"]}`} {
			t.Run(command+body, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if command == "readback" && r.Method == http.MethodPut {
						w.WriteHeader(http.StatusNoContent)
						return
					}
					_, _ = io.WriteString(w, body)
				}))
				defer server.Close()
				setTestEnv(t, server.URL)
				args := []string{"issue", command, "101"}
				switch command {
				case "list", "search":
					args = []string{"issue", command, "--sprint-id", "34"}
				case "create":
					args = createIssueArgs()
				case "readback":
					args = []string{"issue", "update", "101"}
				}
				stdout, stderr, code := captureRun(t, append(args, "--quiet"))
				if code != 1 || stdout != "" || stderr == "" {
					t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
				}
				if command == "readback" && (!strings.Contains(stderr, "readback failed") || !strings.Contains(stderr, "may have succeeded")) {
					t.Fatalf("missing readback context: %s", stderr)
				}
			})
		}
	}
}

func TestIssueReadbackAPIErrorNeverPrintsSuccess(t *testing.T) {
	for _, mode := range []string{"", "--quiet", "--json"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, "read denied")
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			args := []string{"issue", "update", "101", "--story-points", "5"}
			if mode != "" {
				args = append(args, mode)
			}
			stdout, stderr, code := captureRun(t, args)
			if code != 1 || stdout != "" || !strings.Contains(stderr, "readback failed") || !strings.Contains(stderr, "403") || !strings.Contains(stderr, "read denied") {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}

func TestIssueEmptyReadAndCreateResponsesFail(t *testing.T) {
	for _, args := range [][]string{
		{"issue", "show", "101", "--quiet"},
		{"issue", "list", "--quiet"},
		append(createIssueArgs(), "--quiet"),
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			stdout, stderr, code := captureRun(t, args)
			if code != 1 || stdout != "" || stderr == "" {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}

func TestPlanningFlagsAreDiscoverable(t *testing.T) {
	stdout, stderr, code := captureRun(t, []string{"commands", "--quiet"})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	for _, flag := range []string{"--sprint-id", "--story-points", "--clear-sprint", "--clear-story-points", "--custom-fields", "--all-statuses"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("catalog missing %s", flag)
		}
	}
	setTestHome(t)
	for _, command := range []string{"create", "update"} {
		_, help, _ := captureRun(t, []string{"issue", command, "--help"})
		for _, flag := range []string{"-sprint-id", "-story-points", "-clear-sprint", "-clear-story-points", "-custom-fields"} {
			if !strings.Contains(help, flag) {
				t.Errorf("%s help missing %s", command, flag)
			}
		}
	}
}

func TestIssuePlanningHumanOutput(t *testing.T) {
	for _, command := range []string{"show", "list"} {
		t.Run(command, func(t *testing.T) {
			issue := `{"id":101,"subject":"Planning","easy_sprint":{"id":34,"name":"September"},"easy_story_points":0,"custom_fields":[{"id":7,"name":"Labels","value":["a","b"]},{"id":8,"name":"Empty","value":null}]}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if command == "show" {
					_, _ = io.WriteString(w, `{"issue":`+issue+`}`)
				} else {
					_, _ = io.WriteString(w, `{"issues":[`+issue+`]}`)
				}
			}))
			defer server.Close()
			setTestEnv(t, server.URL)
			args := []string{"issue", command}
			if command == "show" {
				args = append(args, "101")
			}
			stdout, stderr, code := captureRun(t, args)
			if code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
			for _, expected := range []string{"#34 September", "Story points", "Custom fields", "Labels (#7)", `["a","b"]`, "Empty (#8)", "null"} {
				if !strings.Contains(stdout, expected) {
					t.Errorf("missing %q in %s", expected, stdout)
				}
			}
		})
	}
}
