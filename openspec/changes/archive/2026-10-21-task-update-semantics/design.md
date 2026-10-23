# Design: task-update-semantics

> Note: account, API, role, table, bucket, and profile identifiers were replaced
> with placeholders when preparing this repository for a portfolio. The technical
> content and historical decisions are unchanged.

## Context

The update path is `handler.updateTask` -> `TaskService.UpdateTask` ->
`TaskRepository.UpdateTask`. Today:

- `UpdateTaskRequest` uses `string`/`bool` without presence markers, so an absent
  field is indistinguishable from a zero value.
- The repository performs an unconditional `SET` of all three fields and does not
  request return values.
- No `PATCH` route exists in `template.yaml`.

## Goals

- Make PUT a strict full replacement and PATCH a true partial update.
- Reject unknown fields and `null` values with `400`.
- Return the persisted state on success.
- Never modify the task when the request is invalid.

## Decisions

### D1: Presence detection, null rejection, and single-object body (Option A)

PATCH parses the body into a struct with pointer fields (`*string`, `*bool`),
which distinguishes an absent field (`nil`) from an explicit value. PUT uses the
same body discipline against its three required fields. To also reject `null`, the
handler inspects the raw body as a `map[string]json.RawMessage`; any value whose
trimmed bytes equal `null` is a `400`.

Verification note: the `null` and unknown-field checks must ultimately run through
`json.Decoder.DisallowUnknownFields()` **decoding into a typed struct**, because
decoding into a plain map does not reject unknown fields. The raw map read is a
pre-pass used only to detect `null`; the authoritative decode is the typed one
with `DisallowUnknownFields()`.

The body MUST be exactly one JSON object and nothing else. The handler enforces:

- **Empty body** (zero bytes, or whitespace only) -> `400`.
- **Non-object JSON** (array, string, number, boolean, `null`) -> `400`.
- **Trailing JSON values** after the object (e.g. `{...}{...}` or `{...} extra`) ->
  `400`; after decoding the object, a second decode MUST yield `io.EOF`.

Verified behavior:

- `{"completed":false}` -> present with value `false`.
- `{"completed":true}` -> present with value `true`.
- `{}` -> absent (nil).
- `{"completed":null}` -> pointer decoding yields nil (indistinguishable from
  absent), which is why the explicit `null` guard is required.
- `{"foo":1}` -> `json: unknown field "foo"` with `DisallowUnknownFields` (typed
  decode); a map-only decode would silently accept it.

Option A is chosen over a custom `nullable[T]` type because it keeps a single
typed parse path for presence and unknown fields, with a small `null` pre-pass.

### D2: PUT requires all three fields

PUT requires `title`, `description`, and `completed`. Absence of any of them is a
`400`. An explicit `completed: false` counts as present. This is the behavior that
fixes the current silent data loss.

### D3: PATCH requires at least one field

PATCH accepts any non-empty subset of the three fields. A body with none of them
(`{}`) is a `400`. Only the provided fields are written; absent fields keep their
current values. An explicit `completed: false` is applied.

### D4: Validation of provided fields

The existing rule applies: `title` must not be empty after `strings.TrimSpace`
(`internal/service/task_service.go`). It is applied only when the field is
provided. Omitting `title` neither validates nor changes it. A request that fails
validation makes no write.

### D5: Persisted state via UpdateItem for both PUT and PATCH

Both PUT and PATCH are served by a single `UpdateItem` call that returns the
persisted state, so the handler never performs a second read:

- `ReturnValues: ALL_NEW` is requested on the `UpdateItem` for **both** PUT and
  PATCH; the returned item is the task the handler serializes as the `200` body.
- `ConditionExpression: attribute_exists(id)` detects a missing task and maps to
  `ErrTaskNotFound` (404).
- PUT writes all three attributes (full replacement); PATCH writes only the
  provided attributes.

For PATCH, the `UpdateExpression` is dynamic over the provided attributes; for PUT
it sets all three. Both use `ExpressionAttributeNames` aliases for every attribute
name referenced in the `UpdateExpression` and `ExpressionAttributeValues` for every
value. The set of attribute names is fixed (`title`, `description`, `completed`);
no user-supplied fragment is ever interpolated into an expression string.

This avoids a `GetTask -> merge -> UpdateTask` read-modify-write and its race
window. Validation of a provided `title` happens in the service/handler on the
submitted value only; no extra pre-read is introduced for that purpose.

### D6: Update operations return the persisted task

`TaskService.UpdateTask` and `TaskService.PatchTask` return `(model.Task, error)`
(today `UpdateTask` returns only `error`). The task they return is the one
`UpdateItem` reports via `ReturnValues: ALL_NEW`, and the handler serializes it as
the `200` body. The interface signature and the `ReturnValues: ALL_NEW` choice are
consistent: the persisted item flows from the write straight to the response.

### D7: Error mapping

- Missing field in PUT, empty PATCH, unknown field, `null` value, invalid `title`
  -> `400`.
- Missing task -> `404`.
- Invalid JSON -> `400`.
- Unexpected error -> `500` with a generic message.

### D8: Conditional-write error mapping preserved

The existing PUT already uses a conditional write
(`internal/repository/task_repository.go:114`): `UpdateItem` with
`ConditionExpression: attribute_exists(id)`, mapping a failure via
`errors.As(err, &types.ConditionalCheckFailedException)` to `ErrTaskNotFound`
(`:152-160`). PATCH reuses exactly this pattern.

The mapping MUST stay precise: only a DynamoDB `ConditionalCheckFailedException`
becomes `ErrTaskNotFound` (404). Every other DynamoDB error (throughput, validation,
serialization, network) MUST be returned unchanged and surface as a generic `500`;
it must never be reported as `404`. `errors.As` on the SDK's typed error is the
mechanism, not string matching. A conditional failure and an unrelated DynamoDB
error are therefore never conflated, for either PUT or PATCH.

Consequence for tests: the repository must be exercised so a conditional failure
maps to `ErrTaskNotFound` while an unrelated error is passed through; only the
former yields `404`.

## Alternatives considered

- Custom `nullable[T]` with `UnmarshalJSON` for `null` (rejected: more code for the
  same observable behavior).
- PATCH as `GetTask` + merge + `UpdateTask` (rejected: adds a race window and an
  extra read).
- Returning the request-derived task (rejected: does not reflect persisted state).

## Risks

- Changing `TaskService.UpdateTask` signature touches the interface and all mocks.
- The dynamic `UpdateExpression` must be built safely: attribute names go through
  `ExpressionAttributeNames` and values through `ExpressionAttributeValues`, and the
  allowed attribute names are fixed (`title`, `description`, `completed`). No
  user-controlled text enters the expression, so the risk is limited to
  assembling the fixed alias/value maps correctly.

## Validation

The repository integration tests were executed against a dedicated DynamoDB table,
so the suite never reads or writes application data:

- Table: `YOUR_TASKS_TABLE_NAME`, independent from the application table.
- Region: `us-east-2`.
- AWS profile: `YOUR_AWS_PROFILE`.
- Schema: partition key `id` (String), no secondary indexes, `PAY_PER_REQUEST`
  billing.

Command:

```
AWS_PROFILE=YOUR_AWS_PROFILE AWS_DEFAULT_REGION=YOUR_AWS_REGION TASKS_TABLE_NAME="YOUR_TASKS_TABLE_NAME" go test -tags=integration -count=1 -v ./internal/repository/...
```

Result: `PASS`, `ok tasks-api/internal/repository 28.727s`. The 13 integration
tests and the repository unit tests included in the same package run passed.

An earlier run executed the PUT/PATCH tests against the application table with 9
tests passing; the complete validation above was performed against the
independent table. The integration test table is left in place; its lifecycle is
decided separately.
