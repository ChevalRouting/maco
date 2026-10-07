# Code style

Rules that gofmt and eslint do not enforce. Applied across the whole tree; new
code must follow them.

## Go

### Logical blocks

A logical block is a short run of statements that belong together, the canonical
case being a call and its error check:

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, err
}

var m types.VMManifest
if err := yaml.Unmarshal(data, &m); err != nil {
    return nil, err
}

return &m, nil
```

* The assignment that produces `err` and the `if err != nil` check are never
  separated by a blank line: they are one block.
* Every block that ends with a closing brace (`if`, `for`, `switch`, `select`,
  `range`) is followed by a blank line when more code follows in the same scope.
* Plain statements are grouped by intent; a blank line separates groups, not
  every line.

### Comments

Code is the documentation, and it carries no comments. The only `//` lines that
belong in the tree are machine directives (`//go:build`, `//go:embed`,
`//go:generate`, `//nolint`) and the swag annotations (`// @...`) that generate
the OpenAPI spec. Naming and structure must make the intent clear on their own;
if a fragment cannot be understood without prose, restructure it until it can.

The same rule holds for the TypeScript frontend: no comments, the only allowed
`//` and `/* */` lines being required tooling directives (`/// <reference ... />`,
`@ts-expect-error`, `eslint-disable-*`). Generated code is exempt because it is
never hand-edited: `web/src/api/generated/` and the sqlc output in
`pkg/db/generated/` keep their generator headers as-is.

### No inline functions and structs

Anonymous functions and anonymous struct types in the middle of logic hurt
readability. Prefer:

* a named top-level function over a closure passed inline, unless the closure
  captures locals and is only a few lines (cobra `RunE` handlers are the
  accepted exception, since they capture command flags);
* a named type over `struct{ ... }` literals declared at the use site, including
  for JSON request/response shapes;
* named types for map/slice element structs.

Small `func()` literals for `sort.Slice`, `defer`, or goroutine bodies of one or
two lines are fine.

### Ignored errors

Intentionally ignored errors are written as explicit `_ =` assignments, never
left bare.

## Frontend

Pages stay thin: layout, fetching, and wiring. Sizeable feature UI belongs in
`web/src/components/`. Use named request/response types and named handlers for
multi-step actions. Short handlers that capture component state are acceptable.

Configuration forms use cheval-ui `PreferencesGroup`, `EntryRow`, `ComboRow`,
and `SwitchRow`. Selections use the shared `Select` primitives. Creation flows
use the shared `Dialog`; destructive actions use `AlertDialog`. Do not recreate
native selects or checkbox controls when the shared system provides them.

Numeric values use `IntegerEntryRow`, a standard cheval-ui text input with a
numeric keyboard hint. Preserve raw input while editing, validate whole digits
and field-specific limits, show inline errors, and block invalid submission.
Convert valid text to numbers only at the input boundary.

Follow the [GNOME HIG button guidance](https://developer.gnome.org/hig/patterns/controls/buttons.html):
each view has at most one suggested or destructive action. Routine and table
actions use neutral buttons; destructive confirmation uses the destructive
variant. Icon actions need an accessible label and tooltip.

Use cheval-ui components and semantic palette tokens instead of custom colors.
Status pills use the shared `StatusBadge` components: pending is warning,
running jobs are info, successful jobs and running VMs are success, failed jobs
are destructive, and stopped VMs are neutral. Always include a text label;
color must not be the only indication of state. Preserve the shared theme's
light and dark palette and visible keyboard focus.

## Text

No em dashes anywhere (code, UI strings, docs); use a comma, colon, or
parentheses.

## Commits

Lowercase imperative subject, optionally prefixed by the touched area (`vm: ...`,
`manifest: ...`). No conventional-commit prefixes, no trailers.

## Linting

Agents must run `task check-style` before building and before committing. It
checks both Go modules, the UI, and repository-specific style rules. `task lint`
is an alias for the same check.

### Automated validation

The shared Go configuration is `.golangci.yml` (golangci-lint v2). The UI uses
`web/eslint.config.mjs`, `web/lint/rules.mjs`, and an independent comment check.
Repository-specific Go syntax checks, text checks and lint orchestration live
in `cmd/stylecheck/`. `task check-style` builds this Go command as
`build/stylecheck` for the execution host, then runs it.

Install a linter built with the same or a newer Go toolchain than this project:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
npm --prefix web ci
```

The pinned UI development dependencies use ESLint 10, its TypeScript parser,
and the maintained stylistic plugin. Use a supported Node version for ESLint
10: Node 20.19+, 22.13+, or 24+.

Before building, run the dedicated task and proceed only when it passes:

```sh
task check-style
task build
```

Use the same sequence for `task build-ui`, `task build-mcp`, and `task web-build`.
Run `task test-lint` after changing lint configuration or custom rules.

`task check-style` uses the Go binary to run all groups and returns a failure
if any group fails. It checks both the root Go module and the independent
`mcp/` module, Go syntax and text, and handwritten TypeScript/TSX. It reports existing violations too: there is no
baseline suppression or changed-lines-only mode. Do not mistake a valid linter
configuration for a clean codebase. Resolve violations before building or
claiming validation passed.

Build and run the checker directly when useful:

```sh
task build-style-checker
./build/stylecheck
./build/stylecheck --only style
./build/stylecheck --only text
./build/stylecheck --only style cmd/stylecheck/main.go
```

The command accepts `--root` with a path inside the repository. Without file
arguments, it checks tracked and unignored untracked files. Optional file
arguments restrict syntax/text checks to those paths. `--only go` checks both
Go modules and `--only web` runs the UI checks. Checks report failures with a
nonzero exit status and do not modify source files.

For focused feedback:

```sh
task lint-go
task lint-mcp
task lint-style
task lint-web
npm --prefix web run lint:fix
```

The UI fix command applies safe ESLint formatting fixes. It does not rewrite
components or delete prose comments. Use `gofmt -w` on edited Go files.

| Rule | Enforcement |
| --- | --- |
| Go formatting | Go syntax checker; golangci-lint's gofmt formatter is configured too |
| Blank line after a Go control-flow block | `wsl_v5` plus Go syntax checker, including switch/select cases |
| Error assignment immediately followed by its check | `wsl_v5` plus Go syntax checker for `err` and names ending in `Err` |
| Prose comments | Go syntax checker and UI comment checker; compiler directives, swag annotations, and cgo preambles are preserved |
| Named Go model types | Go syntax checker rejects anonymous nonempty structs, including test cases |
| Short Go closures | Go syntax checker caps anonymous bodies at three direct statements and five nonblank source lines; Cobra `RunE` handlers are exempt |
| Explicitly ignored Go errors | `errcheck`, including tests, without default cleanup/printing exclusions; `_ =` and `_, _ =` remain allowed |
| Named UI object types | `stylecheck/named-types` rejects inline object shapes unless they directly declare a named type alias |
| Named multi-step UI event handlers | `stylecheck/named-handlers` flags inline JSX handlers with more than two direct statements |
| Shared select, checkbox and numeric controls | `stylecheck/shared-controls` |
| Semantic palette | `stylecheck/semantic-palette` checks literal utility colors, hex colors and RGB/HSL strings; SVG artwork fill/stroke is excluded |
| Neutral table actions | `stylecheck/neutral-table-actions` |
| Icon action labels and tooltips | `stylecheck/accessible-icon-actions` checks statically identifiable Lucide icon-only buttons |
| UI formatting | Single quotes, no semicolons, no trailing spaces, final newline |
| Em dashes | Repository text checker, including documentation |
| Commit prefixes, lowercase imperative subjects and trailers | Agent review against the Commits section |

Comment-naming checks ST1020, ST1021 and ST1022 are disabled because the policy
requires no prose comments and permits swag annotations on exported symbols.
Unused ESLint disable directives fail validation, and golangci-lint suppressions
must name a specific linter.

Generated Go directories and `web/src/api/generated/` are excluded. Required
TypeScript directives remain allowed. A blanket `eslint-disable` comment is
not allowed, and cannot disable the independent comment check.

Some rules still require review: grouping plain statements by intent, whether a
small Go closure captures locals, whether a page contains too much feature UI,
input-state preservation and field limits, one emphasized action per rendered
view, destructive actions using `AlertDialog`, status pill semantics, dynamic
palette expressions, keyboard focus, and imperative commit grammar. Automated
size limits are a conservative interpretation of "short"; a linter pass does
not replace those checks.

### Agent workflow

Read this document before editing. Run `task check-style` before each build and
before committing, and review the rules listed above that need human judgment.
Keep commit subjects lowercase and imperative, optionally prefixed by the
actual touched area, with no conventional prefixes or trailers.

The check runs on the working tree and does not stage files, change Git
configuration, or install commit hooks. Build tasks stay independent; agents
must run the dedicated check first. Use `task check-style` and `task test-lint`
in CI if hosted checks are added later.

Report actual failures. Do not weaken checks or add broad exclusions to conceal
an existing violation. If a tool fails to load packages or cannot read the Go
export-data format, fix the toolchain installation first; that failure is not
a source-code diagnostic.
