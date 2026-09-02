# Code Conventions

## Go Style

- Idiomatic Go: explicit errors (`fmt.Errorf("...: %w", err)`), no `panic` outside `main`.
- Short package names, no unnecessary over-abstraction.
- Comments in English in code.

## Testing

- Table-driven tests for all classification logic (`pkg/analyzer`).
- Tests in `pkg/git` and `pkg/analyzer` must use **real temporary Git repos** (`t.TempDir()` + `exec.Command("git", "init", ...)`), not mocks — too risky to miss real cases.

## Workflow

- Work phase by phase (see `PLAN.md`). Do not mix phases in a single commit. One commit = one phase (or one clear sub-part of a phase).
- Never consider a task done until:
  - Code compiles with no lint warnings
  - Tests covering the new behavior exist and pass
  - No safety rule above is violated
  - README or CHANGELOG is updated if visible behavior changes
