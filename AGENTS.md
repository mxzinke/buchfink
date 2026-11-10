# Buchfink

Buchfink is local bookkeeping software for small German companies (GmbH and UG). An AI agent keeps the books through the MCP server built into the app; the desktop app shows the state and takes the approvals only a human may give (committing periods, confirming submissions, adopting the annual accounts, company master data).

The software and its documentation are German. Identifiers are mostly English; code comments and docs are German.

## Layout

| Path | Content |
|---|---|
| `main.go` | Wails v3 desktop app |
| `internal/core` | host, tenant, read/write with the write lock, channels (`ui`, `system`, `agent:<slug>`), app-only operations |
| `internal/service`, `internal/domain`, `internal/accounting`, `internal/repository` | business logic, model, rules (SKR04, tax), SQLite per tenant |
| `internal/mcpserver` | MCP server with its own OAuth server (127.0.0.1:7465) |
| `internal/wailsbridge` | bindings for the frontend |
| `internal/declaration`, `internal/ebilanz`, `internal/einvoice`, `internal/invoice`, `internal/export`, `internal/procdoc` | tax returns, E-Bilanz, e-invoices, data export, procedure documentation |
| `cmd/buchfink-mcp`, `packaging/claude-desktop` | stdio bridge and `.mcpb` for Claude Desktop |
| `frontend/` | React/TypeScript UI |
| `docs/` | documentation, see `docs/README.md` |
| `website/` | project site (GitHub Pages) |
| `skills/`, `plugins/` | end-user skills and the Claude Code plugin |

`docs/entwicklung/codebase.md` maps topics to code.

## Build and test

```sh
npm --prefix frontend ci
task dev            # run the app
task check          # gofmt check, go vet, Go tests, check scripts
task test           # Go tests without the bridge
npm --prefix frontend test && npm --prefix frontend run build
```

On Linux, Wails builds against GTK 4 / WebKitGTK 6.0. With GTK 3 / WebKitGTK 4.1 add the `gtk3` tag to everything that compiles `internal/wailsbridge` or `main.go`: `GOFLAGS=-tags=gtk3 task test:bridge`, `task build EXTRA_TAGS=gtk3`, `EXTRA_TAGS=gtk3 task dev`. `go build ./...` needs `frontend/dist` (`npm --prefix frontend run build`). Details: `docs/entwicklung/README.md`.

Operations reserved for the human are refused for agents in the core and the services (`core.AppOnly`, `service.requireHuman`). Keep these checks there; an MCP tool must not work around them.

## Skills

The end-user skill lives under `skills/buchfink/`: `SKILL.md` holds the ground rules and a router to the references, `references/<area>/` the domain knowledge and working steps per area. Every statement of law cites its source (statute, BMF letter, form); `skills/QUELLEN.md` lists the sources. What Buchfink computes or enforces belongs in code, not in the skill.

## Documenting

We do use code comments and other markdown files to explain background information, decisions we took or legal references/requirements. Every comment can go stale or contradict the code. Keep minimal and document logics, requirements or background information; default to deleting a comment if it tells about things which have been done in the past. Only ever document the current state.

If a lot of documentation is needed, it's a sign for bad code, naming or wrong architecture; rewrite or restructure in clear modules and interfaces instead.

Writing style for German texts: `docs/gestaltung/schreibweise.md`.

## Commits

Use Conventional Commits for both commit messages and PR titles: `type(scope): summary` — e.g. `fix(webui): …`, `feat(runtime): …`, `refactor(controller): …`. The **commit body** is where the _why_ belongs — decisions, rationale and trade-offs go in commit body rather than in code comments.
