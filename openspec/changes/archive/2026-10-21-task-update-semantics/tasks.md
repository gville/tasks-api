# Tasks: task-update-semantics

## 1. Specification

- [x] 1.1 Author the `task-management` delta spec covering PUT and PATCH requirements and scenarios; verify `openspec validate task-update-semantics --strict` passes

## 2. HTTP API route

- [x] 2.1 Add a `PATCH /tasks/{id}` `HttpApi` event to `TasksFunction` in `template.yaml`; verify `sam build` and `sam validate` succeed and the route is registered

## 3. Handler

- [x] 3.1 Route `http.MethodPatch` to a new `patchTask` handler in `handler.go`
- [x] 3.2 Parse PUT and PATCH as a single JSON object: reject empty/whitespace-only bodies, non-object JSON, and trailing values after the object (`400`); reject unknown fields via `DisallowUnknownFields()` decoding into the typed struct (not a map-only decode); reject `null` values via the raw `map[string]json.RawMessage` pre-pass. Verify each case returns `400` for both PUT and PATCH
- [x] 3.3 Enforce PUT requires `title`, `description`, and `completed`; verify each missing field returns `400` and `completed: false` succeeds
- [x] 3.4 Enforce PATCH requires at least one field; verify `{}` returns `400`
- [x] 3.5 Ensure both handlers serialize the task returned by the service (the persisted state from `ReturnValues: ALL_NEW`); verify the body equals that returned task, and that a follow-up read matches only in tests with no concurrent writes

## 4. Service

- [x] 4.1 Change `TaskService.UpdateTask` and add `TaskService.PatchTask` to return `(model.Task, error)`; update the `TaskService` interface and all mocks so the returned task is the persisted one from `UpdateItem`
- [x] 4.2 Apply the existing `title` non-empty validation to provided fields only; verify an omitted `title` is not validated or modified and no extra read is performed

## 5. Repository

- [x] 5.1 Implement PATCH as a single `UpdateItem` with a dynamic `UpdateExpression` over provided attributes, aliasing every attribute name through `ExpressionAttributeNames` and every value through `ExpressionAttributeValues`, plus `ConditionExpression: attribute_exists(id)` and `ReturnValues: ALL_NEW`
- [x] 5.2 Ensure PUT keeps a full replacement (all three attributes) with `ReturnValues: ALL_NEW` and the `attribute_exists(id)` condition, using `ExpressionAttributeNames`/`ExpressionAttributeValues`; verify both operations map only a `types.ConditionalCheckFailedException` (via `errors.As`) to `ErrTaskNotFound`, while every other DynamoDB error is passed through unchanged and never reported as `404`
- [x] 5.3 Verify that only the fixed attribute names (`title`, `description`, `completed`) are used in the `UpdateExpression` and that no user-supplied fragment is interpolated

## 6. Tests

- [x] 6.1 Handler: PUT full success; PUT missing title/description/completed -> 400; PUT `completed: false` success; PATCH only title/description/completed true/completed false; PATCH multiple; PATCH `{}` -> 400; empty body -> 400; non-object JSON -> 400; trailing JSON values -> 400; unknown field -> 400 (PUT **and** PATCH); `null` field -> 400 (PUT **and** PATCH); invalid JSON -> 400; `title` empty -> 400; `title` whitespace-only -> 400; nonexistent -> 404; response equals persisted state; every invalid request performs no write
- [x] 6.2 Service: `UpdateTask`/`PatchTask` return the persisted task; provided-field `title` validation (empty and whitespace-only); omitted `title` untouched; invalid request performs no write; nonexistent propagates `ErrTaskNotFound`
- [x] 6.3 Repository: conditional failure maps to `ErrTaskNotFound` while an unrelated DynamoDB error is returned unchanged (not `404`)
- [x] 6.4 Integration (`integration` tag): PATCH only `completed: false` flips a `true` task and preserves title/description; PATCH only `title` preserves description/completed; PUT full replacement verified by a follow-up read (no concurrent writes); PUT/PATCH on a missing id return `ErrTaskNotFound`. Verified against a dedicated DynamoDB table (environment and command in `design.md`, "Validation"): the 13 integration tests and the repository unit tests included in the same package run passed

## 7. Client and docs

- [x] 7.1 Add a Bruno `Patch.yml` request (partial) and update `Update.yml` to include `completed`
- [x] 7.2 Update `AGENTS.md` conventions for the PUT/PATCH contract and error mapping
