# AGENTS.md

## Project

tasks-api is a serverless REST API for managing tasks, built with Go and AWS. It is a learning project focused on practicing AWS and cloud-native backend development.

## Stack

- Go 1.27
- AWS Lambda (runtime `provided.al2023`, x86_64)
- Amazon API Gateway HTTP API (payload format v2)
- Amazon DynamoDB (single table, partition key `id`)
- AWS SAM (infrastructure defined in `template.yaml`)
- Amazon Cognito (JWT authorizer on all routes)
- Region: `us-east-2`

## Layout

```
cmd/tasks-api/main.go     entrypoint; wires repository -> service -> handler
internal/handler/         HTTP layer: routing by method, JSON, status codes
internal/service/         business logic: validation, ULID generation
internal/repository/      DynamoDB access
internal/model/           Task domain type
template.yaml             AWS SAM template
infra/ci/template.yaml    deployment identities: CFN service role + CI deploy policy
collection/               Bruno API collection
```

## Commands

```
make build    # sam build
make test     # go test ./...
make local    # sam local start-api against the deployed table
make deploy   # sam deploy --profile process
```

Repository integration tests require AWS credentials and a dedicated test table:

```
AWS_PROFILE=process AWS_DEFAULT_REGION=us-east-2 \
  TASKS_TABLE_NAME="tasks-api-integration-test" go test -tags=integration -count=1 ./internal/repository/...
```

The suite creates, updates and deletes tasks, so it MUST NOT run against the
application table. `tasks-api-integration-test` is a standalone table (partition key
`id` of type String, no secondary indexes, `PAY_PER_REQUEST`) managed separately
from the application stack; the test run neither creates nor deletes it.

Create it with:

```
aws dynamodb create-table --table-name tasks-api-integration-test \
  --attribute-definitions AttributeName=id,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST --region us-east-2 --profile process
```

The application table (the one deployed by the `tasks-api` stack) is not a valid
target for the integration suite.

## Deployment

Deployments use a least-privilege CloudFormation service role, configured through `role_arn` in `samconfig.toml` for local SAM deployments. The GitHub Actions role (`GitHubActionsTasksApiRole`) uses the scoped `TasksApiCiDeployPolicy` instead of `PowerUserAccess`.

The deployment identities are defined in `infra/ci/template.yaml`, a bootstrap stack kept separate from the application stack because the service role must exist before it can operate that stack.

The bootstrap stack is deployed separately by an authorized administrator:

```
aws cloudformation deploy --stack-name tasks-api-ci \
  --template-file infra/ci/template.yaml \
  --capabilities CAPABILITY_NAMED_IAM --region us-east-2 --profile aws-admin
```

Updating `tasks-api-ci` requires an identity authorized to modify the inline policy of the `TasksApiCloudFormationServiceRole` role, i.e. `iam:PutRolePolicy` on that role. Do not assume the `process` profile has those permissions: in this account `process` maps to `aws-developer`, which is denied that action, so the update fails with `AccessDenied` and rolls back. The `aws-admin` profile is the identity used for the bootstrap in this environment; that profile name is environment-specific, so verify the caller's permissions before updating the stack.

The template creates the CloudFormation service role and the managed CI deploy policy. The GitHub Actions role and its trust relationship must be configured separately. The GitHub OIDC provider and the SAM-managed artifact bucket (`aws-sam-cli-managed-default-*`) are also bootstrap dependencies and are not automatically recreated if deleted.

## Environment

- AWS profile: `process` (override with `AWS_PROFILE`).
- Node `24.22.0` is required for the OpenSpec CLI.
- OpenSpec CLI: `1.14.2`.

## Conventions

- Layering: `handler` handles HTTP/JSON only; `service` owns business rules and ID generation; `repository` owns DynamoDB. Depend inward through the interfaces `handler.TaskService` and `service.TaskRepository`.
- IDs are ULIDs generated in the service layer.
- Update semantics: `PUT /tasks/{id}` replaces the whole task and requires `title`, `description`, and `completed` (an explicit `completed: false` counts as present). `PATCH /tasks/{id}` modifies only the provided fields, requires at least one, and preserves omitted fields. Both reject unknown fields, `null` values, empty/whitespace bodies, non-object JSON, and trailing JSON values with 400.
- Error mapping: `repository.ErrTaskNotFound` -> 404, `service.ErrInvalidTask` -> 400, anything else -> 500 with a generic message. Only a DynamoDB `ConditionalCheckFailedException` maps to `ErrTaskNotFound`; every other DynamoDB error is surfaced as 500.
- Update responses return the task as persisted (DynamoDB `ReturnValues: ALL_NEW`), not reconstructed from the request body.
- Persistence uses DynamoDB conditional writes (`attribute_exists(id)`) to enforce existence for update and delete. Updates are a single `UpdateItem` with `ExpressionAttributeNames`/`ExpressionAttributeValues`; there is no read-modify-write.
- Tests: unit tests mock the layer below; repository integration tests sit behind the `integration` build tag and are not run by `make test`.

## Domain notes

- Tasks are intentionally global (no per-user ownership) for now; this is a learning project. Do not add ownership scoping unless asked.

## OpenSpec

This project uses OpenSpec for spec-driven development. Specs live in `openspec/specs/` and changes in `openspec/changes/`. Drive them with the `openspec` CLI. Tool integrations under `.opencode/` are local and regenerated with `openspec update`.

## Security

- Never commit AWS credentials, tokens, or secrets.
- AWS access uses the `process` profile via `AWS_PROFILE`, not files in the repo.
