# task-management Specification

## Purpose

Defines the observable contract for updating an existing task via `PUT` and
`PATCH`, so clients get predictable replacement and partial-update semantics and
responses always reflect the persisted state.

## Requirements

### Requirement: PUT replaces the entire task

PUT on `/tasks/{id}` SHALL replace `title`, `description`, and `completed` for the
identified task and SHALL respond `200` with the task in its persisted state.

#### Scenario: Successful full replacement

- **WHEN** a PUT includes `title`, `description`, and `completed` for an existing task
- **THEN** the response is `200` and the body is the task as persisted after the update

#### Scenario: Explicit false is accepted

- **WHEN** a PUT sets `completed` to `false`
- **THEN** the update succeeds and the persisted task has `completed` false

### Requirement: PUT requires all three fields

PUT MUST respond `400` when any of `title`, `description`, or `completed` is absent
from the request. An explicit `completed: false` MUST count as present.

#### Scenario: Missing title

- **WHEN** a PUT omits `title`
- **THEN** the response is `400` and the task is not modified

#### Scenario: Missing description

- **WHEN** a PUT omits `description`
- **THEN** the response is `400` and the task is not modified

#### Scenario: Missing completed

- **WHEN** a PUT omits `completed`
- **THEN** the response is `400` and the task is not modified

### Requirement: PATCH modifies only the provided fields

PATCH on `/tasks/{id}` SHALL modify only the fields present in the request and
SHALL leave every absent field unchanged, responding `200` with the task in its
persisted state.

#### Scenario: Only title

- **WHEN** a PATCH contains only `title` for an existing task
- **THEN** only `title` changes and `description` and `completed` keep their previous values

#### Scenario: Only description

- **WHEN** a PATCH contains only `description` for an existing task
- **THEN** only `description` changes and `title` and `completed` keep their previous values

#### Scenario: Only completed true

- **WHEN** a PATCH contains only `completed: true`
- **THEN** only `completed` changes and `title` and `description` keep their previous values

#### Scenario: Only completed false

- **WHEN** a PATCH contains only `completed: false`
- **THEN** only `completed` changes and `title` and `description` keep their previous values

#### Scenario: Multiple fields

- **WHEN** a PATCH contains two or more of the fields
- **THEN** exactly those fields change and any absent field keeps its previous value

#### Scenario: Omitted fields are preserved

- **WHEN** a PATCH omits a field
- **THEN** that field retains the value it had before the update

### Requirement: PATCH requires at least one field

PATCH MUST respond `400` when the request contains none of `title`, `description`,
or `completed`.

#### Scenario: Empty object

- **WHEN** a PATCH body is `{}`
- **THEN** the response is `400` and the task is not modified

### Requirement: Explicit false differs from an absent field

For PATCH, `completed: false` MUST set the value to false, while an absent
`completed` MUST leave the existing value unchanged.

#### Scenario: Explicit false changes the value

- **WHEN** an existing task has `completed: true` and a PATCH sets `completed: false`
- **THEN** the persisted task has `completed` false

#### Scenario: Absent completed preserves the value

- **WHEN** an existing task has `completed: true` and a PATCH omits `completed`
- **THEN** the persisted task still has `completed` true

### Requirement: Unknown fields are rejected

PUT and PATCH MUST respond `400` when the request body contains a field other than
`title`, `description`, or `completed`.

#### Scenario: Unknown field in PUT

- **WHEN** a PUT body includes an unrecognized field
- **THEN** the response is `400` and the task is not modified

#### Scenario: Unknown field in PATCH

- **WHEN** a PATCH body includes an unrecognized field
- **THEN** the response is `400` and the task is not modified

### Requirement: Null values are rejected

PUT and PATCH MUST respond `400` when a recognized field is present with a `null`
value, so that `null` is never treated as an explicit value or as an omitted field.

#### Scenario: Null field in PUT

- **WHEN** a PUT body sets one of `title`, `description`, or `completed` to `null`
- **THEN** the response is `400` and the task is not modified

#### Scenario: Null field in PATCH

- **WHEN** a PATCH body sets one of `title`, `description`, or `completed` to `null`
- **THEN** the response is `400` and the task is not modified

### Requirement: Existing field validations apply

PUT and PATCH SHALL apply the existing task validations to the fields provided. In
particular, a provided `title` MUST NOT be empty after the current validation, and
no new validation rules are introduced. A request that fails validation MUST NOT
modify the task.

#### Scenario: Invalid title in PUT

- **WHEN** a PUT provides a `title` that fails the existing validation
- **THEN** the response is `400` and the task is not modified

#### Scenario: Invalid title in PATCH

- **WHEN** a PATCH provides a `title` that fails the existing validation
- **THEN** the response is `400` and the task is not modified

#### Scenario: Omitted title is untouched

- **WHEN** a PATCH omits `title`
- **THEN** no validation is applied to `title` and its value is unchanged

### Requirement: Missing task returns not found

PUT and PATCH MUST respond `404` when the identified task does not exist, and MUST
NOT modify any data in that case.

#### Scenario: PUT on a nonexistent task

- **WHEN** a PUT targets an id with no task
- **THEN** the response is `404` and no task is created or modified

#### Scenario: PATCH on a nonexistent task

- **WHEN** a PATCH targets an id with no task
- **THEN** the response is `404` and no task is created or modified

### Requirement: Response reflects the persisted state

A successful PUT or PATCH MUST respond `200` with the task as returned by the
persistence operation itself, not reconstructed from the request body. Comparing
that response against a subsequent read is meaningful only when no concurrent
write intervenes; the contract does not guarantee equality across separate
operations under concurrency.

#### Scenario: Body matches persisted state

- **WHEN** a PUT or PATCH succeeds and no concurrent write occurs
- **THEN** the response body equals the state returned by the persistence operation and matches a subsequent read of that id

### Requirement: Request body must be a single JSON object

PUT and PATCH MUST respond `400` when the body is empty, when it is not a JSON
object, or when it contains JSON values after the object.

#### Scenario: Empty body

- **WHEN** the request body is empty or contains only whitespace
- **THEN** the response is `400` and the task is not modified

#### Scenario: Non-object JSON body

- **WHEN** the body is a JSON value that is not an object, such as an array, string, number, boolean, or `null`
- **THEN** the response is `400` and the task is not modified

#### Scenario: Trailing JSON values

- **WHEN** the body contains a valid object followed by additional JSON values
- **THEN** the response is `400` and the task is not modified

### Requirement: Malformed requests are rejected

PUT and PATCH MUST respond `400` when the request body is not valid JSON, and `500`
with a generic message on an internal failure.

#### Scenario: Invalid JSON

- **WHEN** the body is not valid JSON
- **THEN** the response is `400` and the task is not modified

#### Scenario: Internal error

- **WHEN** an unexpected internal error occurs
- **THEN** the response is `500` with a generic message
