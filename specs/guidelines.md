# Development Guidelines

> These guidelines are mandatory for all code in this project. No exceptions.

---

## Testing

- **Unit tests**: Every function/method must have unit tests covering happy path, edge cases, and error paths.
- **Integration tests**: Full flow tests for every feature (API endpoint → service → repository → database).
- **Database tests**: Use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) with Docker to spin up real PostgreSQL instances. No database mocks.
- **Coverage**: Must be **> 98%** at all times. This is non-negotiable.

## Code Quality

- **Clean code**: Professional quality. Meaningful names, small functions, single responsibility.
- **Comments**: In **English only**. Only add comments when the code is complex and warrants explanation of _what it does and why_. Do not comment obvious code.
- **Error handling**: Follow the layer-prefixed wrapping convention (see `plan.md`). Every error must be traceable through the chain.

## Workflow

Each task follows this progressive cycle:

```
Implement task → QA review → Fix issues → Approve PR → Merge → Next task
```

- No task is considered complete until tests pass and QA approves.
- PRs must include all tests for the implemented functionality.
- Never skip tests to move faster.
- Each PR should be focused on one task or a small group of related tasks.

## Go-Specific

- **Package Oriented Design (POD)**: Packages by domain, not by technical layer.
- **Error wrapping**: `fmt.Errorf("domain.layer: %w", err)` — chain reads `handler: service: repository: error`.
- **Linting**: `go vet`, `staticcheck`, `golangci-lint` must pass.
- **Formatting**: `gofmt` / `goimports` on all files.

## Frontend-Specific

- **TypeScript strict mode**: No `any` types.
- **Linting**: ESLint + Prettier must pass.
- **Testing**: Jest + React Native Testing Library for components.
