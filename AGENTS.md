# archaea

Go library of shared base abstractions for event-driven microservices: repository CRUD, ConnectRPC handlers and Kafka messaging.

The common agent policy (branches and worktrees, PR-only changes, real checks, Rabbit ticket lifecycle) is installed at user level and applies here; this file only adds facts specific to this repository.

## Commands

- `make vet`
- `make lint`
- `make test`

## Checks before opening a PR

- `make vet`
- `make lint`
- `make test`

Hooks: run `make hooks` once per clone. The hook path is shared, so new worktrees pick the hooks up automatically.

## Docs

- [`docs/context.md`](docs/context.md)
