# easy8-cli

```text
                                   +---------+
███████╗ █████╗ ███████╗██╗   ██╗  | ███████ |
██╔════╝██╔══██╗██╔════╝╚██╗ ██╔╝  |██     ██|
█████╗  ███████║███████╗ ╚████╔╝   | ███████ |
██╔══╝  ██╔══██║╚════██║  ╚██╔╝    |██     ██|
███████╗██║  ██║███████║   ██║     | ███████ |
╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝     +---------+
```

Small Go CLI for [Easy8](https://easy8.com). Current scope: Issues and Product Backlog Items (PBIs).

## Requirements

- Easy8 API key
- Go 1.22+ only when building from source

## Install

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.ps1 | iex
```

The installers detect OS/architecture, download the matching binary from [GitHub Releases](https://github.com/Easy8Com/easy8-cli/releases), and verify SHA-256 checksums.

If `easy8` is not found after installation, restart the terminal or make sure the install directory is on your `PATH`.

## Configure

Run the setup wizard once:

```bash
easy8 setup
```

The wizard saves the Easy8 base URL and API key to a YAML config file. It also asks whether to enable automatic daily updates and defaults to yes.

For scripts and CI, use non-interactive setup:

```bash
easy8 setup --non-interactive --global --base-url "https://demo.easy8.com" --api-key "<your-key>"
easy8 setup --non-interactive --local --base-url "https://demo.easy8.com" --api-key "<your-key>"
```

Or configure with environment variables:

```bash
export EASY8_BASE_URL="https://demo.easy8.com"
export EASY8_API_KEY="<your-key>"
```

## First Command

```bash
easy8 issue list --limit 10
```

## Command Overview

| Command | Purpose |
| --- | --- |
| `easy8 issue create` | Create an issue |
| `easy8 issue show` | Show issue detail |
| `easy8 issue list` | List issues |
| `easy8 issue search` | Search issues |
| `easy8 issue update` | Update an issue |
| `easy8 pbi list` | List product backlog items |
| `easy8 pbi show` | Show PBI detail |
| `easy8 pbi update` | Update a PBI |
| `easy8 auth status` | Show authentication status |
| `easy8 auth login` | Save API key |
| `easy8 auth logout` | Remove API key |
| `easy8 setup` | Configure base URL, API key, defaults, and autoupdate |
| `easy8 skill` | Print/install bundled agent skills |
| `easy8 commands` | Print command catalog for agents |
| `easy8 update` | Update easy8 from GitHub Releases |
| `easy8 version` | Print version |

## Configuration

Config files:

- Global: `~/.config/easy8/config.yaml`
- Local: `.easy8.yaml` in the current directory or a parent directory

Load priority, highest first:

1. Environment variables
2. Local config `.easy8.yaml`
3. Global config `~/.config/easy8/config.yaml`
4. Built-in defaults

Environment variables:

```bash
export EASY8_BASE_URL="https://demo.easy8.com"
export EASY8_API_KEY="<your-key>"
export EASY8_AUTOUPDATE=true
```

Optional default IDs for `issue create`:

```bash
export EASY8_DEFAULT_PROJECT_ID=1
export EASY8_DEFAULT_TRACKER_ID=1
export EASY8_DEFAULT_STATUS_ID=1
export EASY8_DEFAULT_PRIORITY_ID=1
export EASY8_DEFAULT_AUTHOR_ID=1
export EASY8_DEFAULT_ASSIGNED_TO_ID=1
```

The same defaults can be saved through setup flags:

```bash
easy8 setup --non-interactive --global \
  --base-url "https://demo.easy8.com" \
  --api-key "<your-key>" \
  --project-id 1 \
  --tracker-id 1 \
  --status-id 1 \
  --priority-id 1 \
  --author-id 1 \
  --assigned-to-id 1
```

Invalid integer and boolean environment values produce a warning on stderr.

## Issues

### List Issues

```bash
easy8 issue list --limit 10
easy8 issue list --limit 10 --sort "priority:desc,due_date"
easy8 issue list --q "onboarding"
easy8 issue list --sprint-id 34 --all-statuses
```

### Show Issue Detail

```bash
easy8 issue show 123
easy8 issue show 123 --include journals,attachments
easy8 issue show 123 --json
easy8 issue show 123 --quiet
```

`easy8 issue show --id 123` is also supported for compatibility.

### Search Issues

Fulltext search:

```bash
easy8 issue search --q "onboarding"
```

Search with ID filters:

```bash
easy8 issue search --q "petr" --assignee-id 51 --status-id 2 --priority-id 3 --due-date 2024-01-10 --subject "Login" --task-type-id 1
```

Search with name lookups:

```bash
easy8 issue search --q "petr" --assignee "Alice Doe" --status "New" --priority "High" --task-type "Task" --project "Project A"
```

For assignee, status, priority, task type, and project you can use either name or ID.

### Create Issue

The Easy8 API requires these fields when creating an issue: `subject`, `project_id`, `tracker_id`, `status_id`, `priority_id`, `author_id`, and `assigned_to_id`.

```bash
easy8 issue create \
  --subject "Fix onboarding" \
  --project-id 1 \
  --tracker-id 1 \
  --status-id 1 \
  --priority-id 1 \
  --author-id 1 \
  --assigned-to-id 2 \
  --parent-id 100 \
  --description "Short summary" \
  --done-ratio 0
```

Use `--parent-id <issue-id>` to create the issue as a subtask.

With attachments:

```bash
easy8 issue create \
  --subject "Fix onboarding" \
  --project-id 1 \
  --tracker-id 1 \
  --status-id 1 \
  --priority-id 1 \
  --author-id 1 \
  --assigned-to-id 2 \
  --attachment ./spec.pdf \
  --attachment-description "Specification" \
  --attachment ./build.log
```

### Update Issue

```bash
easy8 issue update 123 --status-id 5 --done-ratio 80
easy8 issue update 123 --subject "New subject"
easy8 issue update 123 --parent-id 100
easy8 issue update 123 --notes "Progress update"
easy8 issue update 123 --attachment ./error.log
easy8 issue update 123 --attachment ./screenshot.png --attachment-description "Failure screenshot"
```

`easy8 issue update --id 123 --status-id 5` is also supported for compatibility.

Issue update notes:

- `--done-ratio` must be between 0 and 100.
- `--parent-id` sets the parent issue and must be greater than 0.
- `--attachment` can be repeated.
- `--attachment-description` is optional and applies to the immediately preceding `--attachment`.
- `--notes` is optional, including attachment-only updates.

### Sprint Planning and Custom Fields

Issue create/update support native agile fields and custom fields:

```bash
easy8 issue update 123 --sprint-id 34 --story-points 5 --quiet
easy8 issue update 123 --story-points 0 --quiet
easy8 issue update 123 --clear-sprint --clear-story-points --quiet
easy8 issue update 123 --custom-fields '[{"id":7,"value":"Planning"},{"id":8,"value":["a","b"]}]' --quiet
```

- `--sprint-id` sends `easy_sprint_id` (a positive integer).
- `--story-points` sends integer `easy_story_points`; zero is a value, not a clear.
- `--clear-sprint` and `--clear-story-points` send explicit JSON `null`. Each
  conflicts with its corresponding set flag. Omitted fields stay unchanged.
- `--custom-fields` accepts one JSON array of objects containing exactly `id`
  (positive integer) and `value`. Values preserve JSON types, including arrays,
  numbers and null. Duplicate IDs are rejected. Only supplied fields are sent;
  the server validates field formats, visibility and permissions.
- Native sprint/story-point fields are distinct from `custom_fields`; they are
  not custom-field IDs. The Easy8 server's agile plugin and model rules determine
  whether fields can be read or changed.

Find task records using a known sprint ID:

```bash
easy8 issue list --sprint-id 34 --all-statuses --quiet
easy8 issue search --sprint-id 34 --project-id 12 --all-statuses --quiet
```

Sprint filtering explicitly defaults to open tasks (`status_id=o`) unless a
status or `--all-statuses` is supplied. The CLI checks every returned task's
`easy_sprint.id` against the requested sprint; missing/inaccessible metadata or
an unrelated task fails the command instead of printing a broadened list.

`--all-statuses` sends `status_id=*` to include closed tasks. On search it cannot
be combined with `--status` or `--status-id`. Sprint discovery itself is available
through the Easy8 UI or the plugin's MCP `easy8_agile_sprints_list/get` tools.

Human detail/table output includes available sprint, story-point and custom-field
values. After an empty successful update response, the CLI reads the issue back.
Explicitly supplied agile fields are verified against create/update responses
or the update readback: sprint object ID, integer points (JSON number or integer
string), and explicit nulls for clears. Missing fields do not prove success.
String story points are supported because the Easy8 agile Swagger publishes
them as strings while the model uses an integer; other known REST field types
remain strict.

If readback fails or requested values cannot be verified, the CLI exits nonzero
without successful output. The error states that the HTTP write may have
succeeded partially and that no rollback was performed. Inspect the issue before
retrying it. A response with the wrong issue ID also fails.

**Current REST limitation:** Easy8's issue REST endpoint omits `easy_sprint`
after a successful clear, but also omits it when the field is inaccessible.
Consequently, `--clear-sprint` can remove the sprint and still exit nonzero
because the result cannot be verified. Check the task before retrying. The CLI
does not treat omission as proof of clearing. This change does not modify the
server's REST contract. The current endpoint returns explicit null for readable
cleared story points, so `--clear-story-points` can be verified.

New human-rendered sprint/custom-field labels and values escape control characters
such as newline, tab and ESC. Story points display zero as `0`, null as `null`,
and an absent field as an empty table cell (omitted detail line).

## Product Backlog Items

### List PBIs

```bash
easy8 pbi list --limit 10
easy8 pbi list --status to_do --board-id 17
easy8 pbi list --q "design" --author-id 51
```

Filters: `--status` (`to_do`, `realization`, `done`, `deleted`), `--author-id`, `--board-id`, and `--q`.

### Show PBI Detail

```bash
easy8 pbi show 42
easy8 pbi show 42 --json
easy8 pbi show 42 --quiet
```

`easy8 pbi show --id 42` is also supported for compatibility.

### Update PBI

```bash
easy8 pbi update 42 --status done
easy8 pbi update 42 --status done --json
easy8 pbi update 42 --name "New name" --estimate 5 --description "Details"
```

`easy8 pbi update --id 42 --status done` is also supported for compatibility.

Updatable fields: `--name`, `--description`, `--status`, and `--estimate`.

## Machine Output

Entity and helper commands support two machine-readable modes:

- `--json`: envelope format with `ok`, `data`, `summary`, and optional `breadcrumbs` / `context`
- `--quiet`: raw API-shaped JSON data

For issue list/search/show/create/update, quiet output and envelope `data`
preserve the **complete issue API response**, including unknown plugin fields,
nested metadata, custom fields, nulls, zeroes, empty arrays and numeric precision.
No missing properties are synthesized and no server values are converted between
strings and numbers. JSON formatting may change. Invalid JSON/API responses
remain errors. For an empty update response, the preserved document is the
subsequent issue readback.

Issue lists require an array of tasks with positive IDs. `issues: null`, null
entries, empty objects and nonpositive IDs are invalid; `issues: []` is valid.

Examples:

```bash
easy8 issue list --json
easy8 issue list --quiet
easy8 issue show 123 --json
easy8 issue show 123 --quiet
easy8 pbi list --json
easy8 pbi list --quiet
easy8 pbi show 42 --json
easy8 pbi show 42 --quiet
easy8 update --json
easy8 update --quiet
```

For a full command catalog:

```bash
easy8 commands --json
easy8 commands --quiet
```

## Agent Skills

This repository contains bundled skill files for agent-driven Easy8 workflows:

```text
skills/easy8-cli/SKILL.md
```

Print the embedded primary skill:

```bash
easy8 skill
```

List bundled skills:

```bash
easy8 skill list
```

Install only the primary `easy8-cli` skill:

```bash
easy8 skill install --target opencode
easy8 skill install --target claude
easy8 skill install --target codex --local
```

Sync all bundled skills:

```bash
easy8 skill sync --target opencode
easy8 skill sync --target opencode --local
easy8 skill sync --target opencode --dry-run
```

Example prompts after installing the skill:

```text
fix issue #1234
fix pbi #42
find pbi onboarding
```

## Updates

Update the current executable from GitHub Releases:

```bash
easy8 update
easy8 update --json
easy8 update --quiet
```

Enable or disable silent daily update checks:

```bash
easy8 setup --autoupdate
easy8 setup --autoupdate=false
```

When enabled, easy8 checks GitHub Releases at most once every 24 hours on normal command startup, verifies `checksums.txt`, and updates the current executable when a newer release exists.

Autoupdate state is stored in `~/.config/easy8/update-state.yaml`.

Autoupdate is skipped for `easy8 version`, `easy8 help`, `easy8 commands`, `easy8 update`, and `easy8 setup`.

## Auth Helpers

```bash
easy8 auth status
easy8 auth login --api-key "<your-key>"
easy8 auth logout
```

For local auth in the current repository:

```bash
easy8 auth login --api-key "<your-key>" --local
easy8 auth logout --local
```

## Build From Source

```bash
go build -o easy8 ./cmd/easy8
```

Build with a version stamp:

```bash
go build -ldflags "-X easy8-cli/internal/cli.Version=0.1.8" -o easy8 ./cmd/easy8
```

Run without installing:

```bash
go run ./cmd/easy8 issue list --limit 10
```

## Testing

Unit tests:

```bash
go test ./...
```

Integration tests require a running Easy8 server:

```bash
EASY8_BASE_URL="https://demo.easy8.com" EASY8_API_KEY="<your-key>" go test -tags integration -v -timeout 600s ./internal/api/
```

Integration tests use the `//go:build integration` build tag and skip automatically when `EASY8_BASE_URL` / `EASY8_API_KEY` are not set.

## Roadmap

- Additional entities: projects, users, time entries, and more
- Config profiles
- Convenience commands: quick create, templates

## License

MIT
