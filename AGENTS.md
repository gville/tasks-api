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

Repository integration tests require AWS credentials and the deployed table:

```
AWS_PROFILE=process TASKS_TABLE_NAME=<table-name> go test -tags=integration ./internal/repository/...
```

Resolve the table name from the stack output:

```
aws cloudformation describe-stacks --stack-name tasks-api \
  --query "Stacks[0].Outputs[?OutputKey=='TasksTableName'].OutputValue" --output text
```

## Deployment

Deployments use a least-privilege CloudFormation service role, configured through `role_arn` in `samconfig.toml` for local SAM deployments. The GitHub Actions role (`GitHubActionsTasksApiRole`) uses the scoped `TasksApiCiDeployPolicy` instead of `PowerUserAccess`.

The deployment identities are defined in `infra/ci/template.yaml`, a bootstrap stack kept separate from the application stack because the service role must exist before it can operate that stack.

The bootstrap stack is deployed separately by an authorized administrator:

```
aws cloudformation deploy --stack-name tasks-api-ci \
  --template-file infra/ci/template.yaml \
  --capabilities CAPABILITY_NAMED_IAM --region us-east-2
```

The template creates the CloudFormation service role and the managed CI deploy policy. The GitHub Actions role and its trust relationship must be configured separately. The GitHub OIDC provider and the SAM-managed artifact bucket (`aws-sam-cli-managed-default-*`) are also bootstrap dependencies and are not automatically recreated if deleted.

## Environment

- AWS profile: `process` (override with `AWS_PROFILE`).
- Node `24.22.0` is required for the OpenSpec CLI.
- OpenSpec CLI: `1.14.2`.

## Conventions

- Layering: `handler` handles HTTP/JSON only; `service` owns business rules and ID generation; `repository` owns DynamoDB. Depend inward through the interfaces `handler.TaskService` and `service.TaskRepository`.
- IDs are ULIDs generated in the service layer.
- Error mapping: `repository.ErrTaskNotFound` -> 404, `service.ErrInvalidTask` -> 400, anything else -> 500 with a generic message.
- Persistence uses DynamoDB conditional writes (`attribute_exists(id)`) to enforce existence for update and delete.
- Tests: unit tests mock the layer below; repository integration tests sit behind the `integration` build tag and are not run by `make test`.

## Domain notes

- Tasks are intentionally global (no per-user ownership) for now; this is a learning project. Do not add ownership scoping unless asked.

## OpenSpec

This project uses OpenSpec for spec-driven development. Specs live in `openspec/specs/` and changes in `openspec/changes/`. Drive them with the `openspec` CLI. Tool integrations under `.opencode/` are local and regenerated with `openspec update`.

## Security

- Never commit AWS credentials, tokens, or secrets.
- AWS access uses the `process` profile via `AWS_PROFILE`, not files in the repo.
