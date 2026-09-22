# easy8-cli AGENTS

This document defines how agents (human or automated) should extend the
easy8-cli project. Keep it short, pragmatic, and consistent with the API.

## Project intent
- Provide a small, fast Go CLI for Easy8.
- Current scope: Issues (tasks) and Product Backlog Items (PBIs).
- Design for future entities without breaking CLI UX.
- Product website: https://easy8.com

## API basics
- Base URL: configurable (default for demo use: https://demo.easy8.com).
- Authentication:
  - Preferred: header api key `X-Redmine-API-Key`.
  - Optional fallback: query parameter `key`.
- Endpoints that use easy_query (issues, users, projects) require `set_filter=1`.
- Enumeration endpoints (trackers, statuses, priorities) return full lists without pagination.

## Endpoints in scope

### Issues
- `GET /issues.json` list issues
- `GET /issues/{id}.json` show issue detail
- `POST /issues.json` create issue
- `PUT /issues/{id}.json` update issue

### Product Backlog Items (PBIs)
- `GET /easy_product_backlog_items.json` list PBIs (requires `set_filter=1`)
- `GET /easy_product_backlog_items/{id}.json` show PBI detail
- `PUT /easy_product_backlog_items/{id}.json` update PBI (returns 204 No Content)

### Lookup endpoints (used for name-to-ID resolution)
- `GET /trackers.json` -- no pagination
- `GET /issue_statuses.json` -- no pagination
- `GET /enumerations/issue_priorities.json` -- no pagination
- `GET /users.json` -- paginated, requires `set_filter=1`
- `GET /projects.json` -- paginated, requires `set_filter=1`

## Create issue fields
- CLI-required fields (IDs can come from configured defaults):
- `subject`
- `project_id`
- `tracker_id`
- `assigned_to_id`
- Optional fields: `status_id`, `priority_id`, `author_id`.
- Omit optional fields when their effective value is zero so the server can apply defaults and workflow rules.
- Explicit flags override config/env defaults; explicit zero omits the field.

## CLI command map
- `easy8 issue create`
- `easy8 issue show`
- `easy8 issue list`
- `easy8 issue search`
- `easy8 issue update`
- `easy8 pbi list`
- `easy8 pbi show`
- `easy8 pbi update`
- `easy8 auth status`
- `easy8 auth login`
- `easy8 auth logout`
- `easy8 setup`
- `easy8 skill`
- `easy8 skill install`
- `easy8 commands`
- `easy8 update`
- `easy8 version`

## Configuration
- Environment variables (recommended):
  - `EASY8_BASE_URL`
  - `EASY8_API_KEY`
  - Optional autoupdate: `EASY8_AUTOUPDATE`
  - Optional defaults: `EASY8_DEFAULT_PROJECT_ID`, `EASY8_DEFAULT_TRACKER_ID`,
    `EASY8_DEFAULT_STATUS_ID`, `EASY8_DEFAULT_PRIORITY_ID`,
    `EASY8_DEFAULT_AUTHOR_ID`, `EASY8_DEFAULT_ASSIGNED_TO_ID`
- Invalid integer/boolean env vars produce a warning on stderr (not silently ignored).
- Interactive `easy8 setup` asks whether to enable autoupdate and defaults to yes.
- `easy8 setup --autoupdate` enables a silent GitHub Releases self-update check at most once every 24 hours.
- `easy8 setup --autoupdate=false` disables it explicitly for non-interactive environments.
- Autoupdate state is stored in `~/.config/easy8/update-state.yaml`.
- Autoupdate is skipped for `version`, `help`, `commands`, `update`, and `setup`.
- Optional config files:
  - Global: `~/.config/easy8/config.yaml`
  - Local: `.easy8.yaml` (current directory or parent)
  - Env vars override both config files.

## Output format
- Human-readable table by default (list, search, create, update).
- Key-value detail format for `issue show` and `pbi show`.
- `--json` flag for envelope machine output (`ok`, `data`, `summary`, optional `breadcrumbs`).
- `--quiet` flag for raw machine-readable output (API-shaped JSON).

## Validation
- `--done-ratio` must be between 0 and 100.

## Error handling
- Non-2xx responses must include status code and error body in stderr.
- Exit non-zero on API errors.
- Invalid config env vars produce a warning on stderr.

## Testing
- Every code change must be covered by tests; aim for line-level coverage.
- Unit tests: `go test ./...`
- Integration tests (require running Easy8 server):
  ```
  EASY8_BASE_URL="https://demo.easy8.com" EASY8_API_KEY="<your-key>" go test -tags integration -v -timeout 600s ./internal/api/
  ```
- Integration tests use build tag `//go:build integration` and skip automatically
  when `EASY8_BASE_URL` / `EASY8_API_KEY` are not set.

## Version
- Set at build time: `go build -ldflags "-X easy8-cli/internal/cli.Version=0.1.9" -o easy8 ./cmd/easy8`
- Defaults to `0.1.9` when not set.

## Extension guidance
- Keep API client in a small internal package (e.g., `internal/api`).
- Add new entity commands under a dedicated package to avoid monolith.
- Do not change existing CLI flags unless there is a strong compatibility reason.
