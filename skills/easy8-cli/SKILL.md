---
name: easy8-cli
description: "Use easy8 CLI to fetch and update Easy8 issues and PBIs. Verify easy8 is installed and provide install/setup steps when missing."
---

# Easy8 CLI

Use this skill when user asks to read or update Easy8 tasks, issues, tickets, or PBIs.

## Agent Invariants (MUST)

1. Run Runtime Preflight before any easy8 command.
2. If easy8 is missing, stop and return installation steps.
3. Prefer machine output with `--quiet`; use `--json` when summary/breadcrumbs help.
4. Never update Issue/PBI unless user explicitly asks for a field change.
5. For missing ID or ambiguous search results, show top matches and ask one focused disambiguation question.

## Runtime Preflight (MUST)

Run:

```bash
easy8 version
```

If preflight fails because `easy8` is unavailable, stop and return this guidance:

macOS/Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.sh | bash
easy8 version
easy8 setup
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/Easy8Com/easy8-cli/main/scripts/install.ps1 | iex
easy8 version
easy8 setup
```

If `easy8` is still not found after install, ask user to restart terminal/session so PATH reloads.

Do not use source-build fallbacks in normal user flows.

To update an existing installation from GitHub Releases:

```bash
easy8 update
```

To enable silent daily self-update checks for normal commands:

```bash
easy8 setup --autoupdate
```

Interactive `easy8 setup` asks about autoupdate and defaults to yes. Use `easy8 setup --autoupdate=false` when automatic updates must be disabled, for example in Docker images.

Autoupdate is skipped for `easy8 version`, `easy8 help`, `easy8 commands`, `easy8 update`, and `easy8 setup`.

## Authentication / Config Prerequisites

Required config or env:

- Run `easy8 setup` (recommended), or
- Set `EASY8_BASE_URL` and `EASY8_API_KEY`

Quick check:

```bash
easy8 auth status --quiet
```

If auth/config is missing, stop and ask for one of:

- `easy8 setup`
- set `EASY8_BASE_URL` and `EASY8_API_KEY`.

## Intent Routing

- `task`, `issue`, `ticket` -> issue commands
- `pbi`, `product backlog item`, `easy product backlog item` -> pbi commands
- verbs `fetch`, `show`, `list`, `search`, `find`, `fix` -> read flow
- verbs `update`, `change`, `set`, `rename`, `mark` -> update flow

ID parsing:

- Accept `#124`, `124`, `issue-124`, `pbi-42`
- Extract numeric ID

## Command Mapping

### Issue detail

```bash
easy8 issue show 124 --quiet
```

### Issue search when ID missing

```bash
easy8 issue search --q "<user text>" --quiet
```

### Issue create (only explicit user request)

```bash
easy8 issue create --subject "New title" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --quiet
easy8 issue create --subject "New title" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --parent-id 100 --quiet
easy8 issue create --subject "New title" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --attachment "./spec.pdf" --attachment-description "Specification" --quiet
```

### Issue update (only explicit user request)

```bash
easy8 issue update 124 --status-id 5 --quiet
easy8 issue update 124 --done-ratio 80 --notes "progress update" --quiet
easy8 issue update 124 --parent-id 100 --quiet
easy8 issue update 124 --sprint-id 34 --story-points 5 --quiet
easy8 issue update 124 --clear-sprint --clear-story-points --quiet
easy8 issue update 124 --custom-fields '[{"id":7,"value":["a","b"]}]' --quiet
easy8 issue update 124 --subject "New title" --description "Updated text" --quiet
easy8 issue update 124 --attachment "./build.log" --quiet
easy8 issue update 124 --attachment "./screenshot.png" --attachment-description "Failure screenshot" --quiet
```

Attachment rules:

- `--notes` is optional, including attachment-only updates.
- `--attachment` can be repeated.
- `--attachment-description` is optional and applies to the immediately preceding `--attachment`.

Issue update expects IDs for lookup fields:

- `--status-id`
- `--priority-id`
- `--assigned-to-id`
- `--parent-id`

Native agile fields are `easy_sprint_id` / `easy_story_points`, separate from
custom fields. Set flags take integers; zero story points remains zero. Clear
flags send explicit null and conflict with the corresponding set flags. Omitted
fields stay unchanged. Custom-field JSON is an array of `{id, value}` objects.

Sprint filtering:

```bash
easy8 issue list --sprint-id 34 --all-statuses --quiet
easy8 issue search --sprint-id 34 --project-id 12 --all-statuses --quiet
```

`--all-statuses` includes closed tasks and conflicts with search `--status` /
`--status-id`. Sprint IDs can be discovered in the Easy8 UI or via the agile MCP
sprint list/get tools. Server permissions and model rules apply to every write.
Sprint filters default to open tasks unless a status/all-statuses is explicit.
Every returned task must have a matching sprint object ID; unavailable metadata
or mismatched results cause an error rather than a broadened result list.

### PBI detail

```bash
easy8 pbi show 42 --quiet
```

### PBI search/list when ID missing

```bash
easy8 pbi list --q "<user text>" --quiet
```

### PBI update (only when explicitly requested)

```bash
easy8 pbi update 42 --status done --quiet
easy8 pbi update 42 --name "New name" --estimate 5 --description "Details" --quiet
```

### CLI self-update

```bash
easy8 update --quiet
easy8 setup --autoupdate
easy8 setup --autoupdate=false
```

Use `easy8 update --quiet` only when the user asks to update the easy8 CLI immediately. Use `easy8 setup --autoupdate` only when the user asks to enable automatic daily CLI updates. Use `easy8 setup --autoupdate=false` when the user asks to disable automatic CLI updates.

## Output Format

- `--quiet`: raw API-shaped JSON (preferred for parsing).
- `--json`: envelope with `ok`, `data`, `summary`, optional `breadcrumbs/context`.

Issue machine output preserves the full API document, including unknown nested
plugin/custom fields and original JSON types. After an empty successful update,
the CLI returns the readback document. Requested agile values are checked in
create/update results, including zero, null and number/string integer points.
Missing/mismatched values or failed readback exit nonzero with no success output;
the HTTP write may have succeeded partially and no rollback was performed.
Do not blindly repeat that write. Wrong issue IDs and malformed issue lists
(null entries or nonpositive IDs) also fail; an empty array is valid.

The current Easy8 REST endpoint omits `easy_sprint` after clearing, just as it
does for inaccessible sprint fields. Therefore `--clear-sprint` may perform the
write but exit nonzero because the result cannot be verified. Check the task;
do not retry automatically or interpret omission as confirmed clearing.
Readable cleared story points are returned as explicit null by that endpoint.

After fetching entity, return a short brief:

- Entity (Issue or PBI)
- ID
- Title (`subject` for issue, `name` for pbi)
- Status
- Assignee/Owner
- Description (shortened)
- Related (linked issues for PBI when available)
- Next Action

## Safety Rules

- Always prefer machine output (`--quiet` or `--json`) for agent parsing.
- Do not update Issue/PBI unless user explicitly asks to change fields.
- For ambiguous text without ID and with multiple search hits, show top hits and ask one focused disambiguation question.
- If entity is not found, report that directly and show the exact command used.
- If update intent has no concrete target field/value, ask one focused clarifying question.

## Examples

- `fix issue #124` -> `easy8 issue show 124 --quiet`
- `find pbi onboarding` -> `easy8 pbi list --q "onboarding" --quiet`
- `set pbi #42 to done` -> `easy8 pbi update 42 --status done --quiet`
- `change issue #123 done ratio to 80` -> `easy8 issue update 123 --done-ratio 80 --quiet`
- `attach ./build.log to issue #123` -> `easy8 issue update 123 --attachment "./build.log" --quiet`
- `attach ./shot.png to issue #123 with description "Failure screenshot"` -> `easy8 issue update 123 --attachment "./shot.png" --attachment-description "Failure screenshot" --quiet`
- `create issue "Fix login" with attachment ./spec.pdf` -> `easy8 issue create --subject "Fix login" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --attachment "./spec.pdf" --quiet`
- `create issue "Fix login" under #100` -> `easy8 issue create --subject "Fix login" --project-id 1 --tracker-id 1 --status-id 1 --priority-id 1 --author-id 1 --assigned-to-id 2 --parent-id 100 --quiet`
