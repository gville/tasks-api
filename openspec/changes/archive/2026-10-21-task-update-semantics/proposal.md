# Proposal: task-update-semantics

## Why

`PUT /tasks/{id}` today accepts partial bodies and fills the missing fields with
zero values (`internal/handler/update_task.go`, `internal/repository/task_repository.go:130`),
silently erasing data. It also responds with a task rebuilt from the request
rather than the persisted state (`internal/handler/update_task.go:96`), and ignores
unknown fields. There is no `PATCH` route at all. The update contract is implicit
and ambiguous, so clients cannot rely on either replacement or partial semantics.

## What changes

- Add a `PATCH /tasks/{id}` route.
- `PUT /tasks/{id}` replaces the whole task and requires `title`, `description`,
  and `completed`; any missing field returns `400`. An explicit `completed: false`
  is valid.
- `PATCH /tasks/{id}` modifies only the fields present, requires at least one
  field (`{}` returns `400`), leaves absent fields unchanged, and distinguishes an
  explicit `completed: false` from an absent `completed`.
- Both operations reject unknown fields and `null` values with `400`.
- Both apply the existing validations to the provided fields; an invalid request
  does not modify the task.
- Both return `200` with the task as actually persisted.

## Out of scope

- The rest of the CRUD (create, list, get, delete) is not specified here.
- No new validation rules beyond the existing `title` non-empty check.
- No changes to authentication, throttling, or the table definition.

## Capability

`task-management` (new).

## Impact

- `template.yaml`: add the `PATCH /tasks/{id}` HTTP API event.
- `internal/handler/handler.go`: route PATCH; request types with presence support.
- `internal/handler/update_task.go`: enforce the full-replacement contract.
- `internal/handler/patch_task.go` (new): partial update.
- `internal/service/task_service.go`: update operations return the persisted task.
- `internal/repository/task_repository.go`: `UpdateItem` with `ReturnValues: ALL_NEW`
  and a dynamic update expression for PATCH.
- Tests (handler, service, integration), Bruno collection, `AGENTS.md`.
- No runtime IAM change: `DynamoDBCrudPolicy` already grants `UpdateItem`.
