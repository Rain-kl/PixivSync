# Wavelet Agent Index

This file is the project-level guide for agents working in Wavelet. More
specialized workflows still live in `.agent/skills/`.

## Always Read The Matching Skill

- `new-async-task`: use when adding or changing Asynq tasks, scheduled jobs,
  task metadata, task payload validation, task logs, task retry behavior, or
  Admin task APIs.
- `new-setting`: use when adding or changing startup config, database-backed
  system/business/public settings, `/admin/system` parameters, or
  `/admin/settings` graphical settings.
- `database-migration`: use when adding or changing database schema, indexes,
  seed data, system config defaults, template defaults, default admin data,
  goose SQL migrations, or the database upgrade flow.
- Go skills: use the focused `go-*` skills for Go implementation details such
  as testing, error handling, packages, context, concurrency, logging,
  documentation, and review.
- `shadcn`: use when adding, changing, or composing shadcn/ui components.

## Non-Negotiable Project Guardrails

- Do not delete `frontend/node_modules`; reinstall with `pnpm install` if
  dependencies need refreshing.
- Keep `internal/util/` framework-free. Do not import Gin, GORM, sessions, or
  other HTTP/framework packages from `internal/util/` or its subpackages.
- Register all HTTP routes only in `internal/router/router.go`.
- Update Swagger (`make swagger`) when API handlers change.
- Run `make code-check` before submitting changes.

## Quick Commands

| Command | When |
| --- | --- |
| `make code-check` | Required before submit |
| `make build-test` | Functional build verification |
| `make swagger` | After adding/changing APIs |
| `make build-embedded` | Release binary with embedded frontend |
| `make license` | After adding Go files |
| `make license-check` | CI/license validation |

# Wavelet Project Development Guide

Use this guide for ordinary Wavelet development. If the task is specifically
about Asynq/background/scheduled tasks, use `new-async-task` as the detailed
workflow.

## Tech Stack

- Backend: Go 1.25+, Gin, GORM, PostgreSQL, optional ClickHouse, Redis, Asynq,
  Cobra, Viper, Swaggo, OpenTelemetry, Zap, AWS SDK v2, Snowflake IDs.
- Frontend: Next.js App Router, TypeScript, Tailwind CSS, pnpm, shadcn/ui.

## Directory Map

Top level:

- `main.go`: program entry, delegates to `internal/cmd`.
- `config.example.yaml`: committed config template. Keep it updated when adding
  config fields.
- `config.yaml`: local runtime config. Do not treat it as committed source.
- `docker/`: integrated, frontend-only, and backend-only Dockerfiles.
- `docs/`: generated Swagger docs. Do not hand edit generated files.
- `frontend/`: Next.js app.
- `internal/`: private Go backend code.
- `scripts/`: local and CI helper scripts.
- `support-files/`: auxiliary deployment files.

Backend:

- `internal/cmd/`: Cobra commands for API, worker, scheduler, root init.
- `internal/config/`: Viper loading and config structs. Runtime code should use
  `config.Config.<Section>.<Field>`.
- `internal/router/`: the only HTTP route registration point.
- `internal/apps/`: feature modules and HTTP handlers.
- `internal/model/`: GORM entities and model-level business methods.
- `internal/db/`: PostgreSQL, Redis, ClickHouse, GORM logging, ID generation,
  and goose SQL migration wiring.
- `internal/storage/`: S3-compatible storage and cache abstraction.
- `internal/task/`: Asynq task framework; see `new-async-task` for changes.
- `internal/service/`: complex business services when handlers/models are too
  narrow a home.
- `internal/common/`: shared response, bind, constants, and common errors.
- `internal/util/`: pure utilities with no framework imports.
- `internal/logger/`: Zap and OTel logging helpers.
- `internal/listener/`: event listeners and message/webhook consumers.
- `internal/otel_trace/`: tracing helpers.

Frontend:

- `frontend/app/`: App Router pages, route groups, root layout, globals.
- `frontend/components/ui/`: shadcn/ui base components.
- `frontend/components/common/`: cross-page business components.
- `frontend/components/layout/`: Header, Sidebar, Footer, app layout pieces.
- `frontend/components/auth/`, `home/`, `animate-ui/`, `providers/`: scoped UI.
- `frontend/contexts/`, `hooks/`, `lib/`, `types/`, `public/`: shared state,
  hooks, clients/utilities, TypeScript types, static assets.

Important common components:

- `components/common/admin/tasks.tsx`: task dispatch UI.
- `components/common/admin/task-executions.tsx`: task execution log/retry UI.
- `components/common/admin/system.tsx`: system config management.
- `components/common/admin/users.tsx`: user management.
- `components/common/general/manage-pannel.tsx`: generic list/detail manager.
- `components/common/general/password-dialog.tsx`: sensitive-action password
  confirmation dialog.
- `components/common/settings/system-settings.tsx`: admin system settings.

## Backend Rules

Naming:

- Go packages and files use lowercase snake words: `auth_source`,
  `postgres_logger.go`.
- Exported Go identifiers use PascalCase; unexported identifiers use camelCase.
- Request/response structs use camelCase with suffixes like
  `listUsersRequest` and `listUsersResponse`.
- Error message constants are camelCase string `const` values, not package-level
  `error` values.
- YAML config keys use lowercase snake case.

Handlers:

- Handler names are verb + noun, for example `ListUsers`.
- Bind with `ShouldBindQuery` or `ShouldBindJSON`.
- Return success through `util.OK(data)`, `util.OKNil()`, or
  `response.RespondSuccess`.
- Return failures with `util.Err(msg)` or `response.RespondFailure`.
- API responses must have the outer shape `{ "error_msg": "", "data": ... }`.
- Pagination responses use `{ "total": 0, "results": [] }` under `data`.
- Every HTTP API needs complete Swagger comments; run `make swagger` after API
  changes.

错误处理与日志:

- 任何关键错误在被吞掉、转换为通用响应，或由后台 worker 忽略之前，
  都必须通过 `internal/logger` 打印日志。
- 禁止用 `_ = ...` 静默丢弃重要错误。如果某个错误因为 best-effort
  操作或确认无害而需要忽略，必须添加简短注释说明原因。
- Handler 可以返回对用户安全的错误信息，但如果底层运行错误对生产问题
  排查有价值，仍然必须记录日志。
- 避免重复刷日志：在真正处理或抑制错误的边界记录一次，然后返回或响应。

Routes and modules:

- Register routes only in `internal/router/router.go`.
- In `internal/apps/<module>/`, use:
  - `routers.go` or `controllers.go` for HTTP handlers.
  - `middlewares.go` for module-specific middleware.
  - `errs.go` for string error constants only.
  - `constants.go` for non-error business constants.
- For Admin modules, prefer `internal/apps/admin/<module>/`.
- If a handler file exceeds 600 lines, contains complex multi-step logic, or
  mixes independent domains, split business logic into `logic.go` or
  `logics.go`. Keep `routers.go` to binding, calling logic, and responding.

Middleware:

- Global middleware belongs in router setup: `gin.Recovery()`,
  `otelgin.Middleware()`, logger middleware, and session middleware.
- Use `oauth.LoginRequired()` for logged-in route groups.
- Use `admin.LoginAdminRequired()` for Admin route groups.

Config:

- Runtime code reads config from `config.Config`, never directly from
  `os.Getenv()`.
- When adding config, update both `config.example.yaml` and
  `internal/config/model.go`.

Database:

- Simple queries may use GORM directly from the model layer.
- Admin code should prefer `db.DB(ctx)` to get tracing-aware DB access.
- Do not put complex SQL in handlers; move it to `internal/model/` or
  `internal/service/`.
- Use goose SQL migrations under `internal/db/migrator/goose/`; do not add
  GORM AutoMigrate-based schema upgrades.
- Do not create physical database foreign keys. Add explicit indexes for
  relation fields instead.
- Database defaults must match Go model zero values (`nil`, `0`, `false`, `""`)
  to avoid surprising inserts.

Strict dependency guard:

- `internal/util/` and its subpackages must stay framework-free.
- Do not import `github.com/gin-gonic/gin`, `gorm.io/gorm`,
  `github.com/gin-contrib/sessions`, or HTTP middleware/framework packages from
  `internal/util/`.
- If utility logic needs web glue, keep pure validation/calculation in
  `internal/util/` and put Gin middleware/response handling in `internal/apps/`.

Admin module workflow:

1. Define or extend models in `internal/model/`.
2. Add goose SQL migrations under `internal/db/migrator/goose/`.
3. Create `internal/apps/admin/<module>/routers.go` and optional `errs.go`.
4. Register routes in `internal/router/router.go`.
5. Run `make swagger`.

## Frontend Rules

Styling:

- shadcn/ui base components should use their `variant` system and global CSS
  variables. Do not hardcode colors, backgrounds, or shadows in business
  `className` when a component variant should own the look.
- If an existing variant is insufficient, extend the shadcn/ui component
  variant instead of hardcoding one-off colors.
- Use Lucide icons for common icon needs. Put custom icons in
  `frontend/components/icons/` as named exports.

Page width:

- Page root containers must support full width. Use `w-full`.
- Do not hardcode page-level max widths like `max-w-6xl` or `max-w-4xl`; the
  main layout owns the normal/full-width constraint.

Component placement:

- Cross-page business components belong in `frontend/components/common/`.
- shadcn/ui primitives belong in `frontend/components/ui/`.
- Route/page-specific components belong in the closest feature directory.

Type safety:

- Do not use `any`.
- Use `unknown` only with explicit narrowing or type assertions before use.
- Use `never` sparingly and document why when it is non-obvious.
- Frontend changes must pass TypeScript and ESLint checks.

Services:

- Frontend API access goes through service classes and the exported `services`
  object.
- Create new services as:

```text
frontend/lib/services/<service-name>/
  types.ts
  <service-name>.service.ts
  index.ts
```

- Service classes extend `BaseService`, define `basePath`, and expose typed
  static methods.
- Register the new service in `frontend/lib/services/index.ts`.

## Quality Gates

- `make code-check`: required before submit; frontend typecheck + ESLint and
  backend golangci-lint.
- `make build-test`: build verification for frontend and Go backend.
- `make swagger`: regenerate Swagger after API changes.
- `make build-embedded`: release binary with frontend static export embedded.
- `make license`: run after adding Go files.
- `make license-check`: validate Go license headers.

Never delete `frontend/node_modules`; refresh dependencies with `pnpm install`.
