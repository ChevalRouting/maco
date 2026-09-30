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

Run `task lint` before pushing: `golangci-lint` (config in `.golangci.yml`).
