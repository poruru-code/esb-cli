# Repository Guidelines

## Top-Level Rule
- When writing complex features or significant refactors, use an ExecPlan (as described in `.agent/PLANS.md`) from design to implementation.

## Project Structure & Module Organization
- `cmd/esb/`: CLI entrypoint and process bootstrap.
- `internal/app/`: dependency injection and wiring.
- `internal/command/`: flag parsing, input resolution, and command orchestration.
- `internal/usecase/`: deploy workflow sequencing and business flow.
- `internal/domain/`: pure domain logic and core types.
- `internal/infra/`: Docker/Compose, SAM parsing, FS, UI, and other I/O adapters.
- `internal/architecture/`: dependency/layer guard tests.
- `assets/runtime-templates/`: embedded runtime template assets.
- `docs/`: architecture and build design notes.

Keep tests close to implementation (`*_test.go`). Golden files live under package `testdata/` directories.

## Build, Test, and Development Commands
- `mise trust && mise install && mise run setup`: install toolchain and git hooks.
- `mise run lint`: run `golangci-lint` with repo config.
- `mise run test`: run all Go tests (`go test ./...`).
- `mise run build`: build the CLI binary to `./esb`.
- `mise run dev`: watch files and rebuild/reinstall via `air`.
- `go build -o esb ./cmd/esb`: direct build path when `mise` tasks are not needed.

## Coding Style & Naming Conventions
- Language/toolchain: Go `1.25.1` (managed by `mise`).
- Formatting is enforced by hooks: `goimports` then `gofumpt` on staged Go files.
- Use lowercase package names and descriptive file names by feature (example: `deploy_inputs_resolve.go`).
- Follow existing layer boundaries (`command -> usecase -> domain/infra`); architecture tests enforce contracts.

## Testing Guidelines
- Prefer targeted runs during development, then run full suite before push.
- Example targeted run: `go test ./internal/command ./internal/usecase/deploy -count=1`.
- Update golden snapshots/testdata when intentional output changes occur.
- Add tests with every behavioral change, especially around deploy input resolution and workflow phases.

## Commit & Pull Request Guidelines
- Use Conventional Commit style seen in history: `feat: ...`, `fix(cli): ...`, `refactor: ...`, `test: ...`, `docs: ...`.
- Keep subject lines imperative and scoped when useful (`fix(artifact): normalize apply inputs`).
- PRs should include: purpose, key design/behavior changes, and exact verification commands run.
- Ensure CI passes (`lint`, `test`, `build`) before requesting review.

## Security & Configuration Tips
- Private module access may require `GOPRIVATE=github.com/poruru-code/esb/*`.
- Never commit secrets; CI auth for private modules is handled via GitHub App secrets.
