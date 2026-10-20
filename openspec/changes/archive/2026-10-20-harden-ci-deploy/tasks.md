# Tasks

## 1. Author the versioned deployment configuration

- [x] 1.1 Create the `infra/ci/` CloudFormation template defining `TasksApiCloudFormationServiceRole` with the `cloudformation.amazonaws.com` trust policy and the resource-scoped permission policy from design D3; verify it passes `aws cloudformation validate-template` (and `sam validate --template infra/ci/template.yaml` if SAM renders it)
- [x] 1.2 Define the `GitHubActionsTasksApiRole` scoped permission policy from design D4 in the same stack (as a managed policy to attach); verify the rendered JSON has no `PowerUserAccess` reference and every statement targets the `tasks-api*` stack, the artifact bucket, or the service role
- [x] 1.3 Add `role_arn = "arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole"` under `[default.deploy.parameters]` in `samconfig.toml`; verify it is present and `sam deploy --help` confirms the flag is honored from config
- [x] 1.4 Document the bootstrap boundary (OIDC provider and `infra/ci/` are deployed once by a human; the SAM-managed artifact bucket is not self-healing) in `AGENTS.md`; verify the described command matches the one in task 2.1

## 2. Bootstrap the roles

- [x] 2.1 Deploy the `infra/ci/` stack once with the admin profile (`aws cloudformation deploy --stack-name tasks-api-ci --capabilities CAPABILITY_NAMED_IAM`); verify the stack reaches `CREATE_COMPLETE`
- [x] 2.2 Inspect the service role and confirm it matches design D3, with no DynamoDB-hardening, log-retention, or tracing permissions (`aws iam get-role-policy`/`list-attached-role-policies`); verify the trust policy lists only `cloudformation.amazonaws.com`. The deployed policy has six statements (`ApiGatewayV2`, `Lambda`, `DynamoDB`, `ExecutionRole`, `InvokeServerlessTransform`, `ReadArtifacts`); D3 was updated to document the two added during the first real deploy and the rollback recovery. Trust lists only the CloudFormation service principal

## 3. Swap the CI deploy identity policy

- [x] 3.1 Attach the scoped policy from `infra/ci/` to `GitHubActionsTasksApiRole` using the admin profile; verify it appears in `list-attached-role-policies`
- [x] 3.2 Detach `PowerUserAccess` from `GitHubActionsTasksApiRole`; verify it no longer appears in `list-attached-role-policies`
- [x] 3.3 Delete the inline `TasksApiReadExecutionRole` policy; verify `list-role-policies` returns an empty list
- [x] 3.4 Negative check: using credentials assumed as `GitHubActionsTasksApiRole`, attempt an out-of-scope call such as `aws dynamodb scan` against the table; verify AWS denies it. Evidence is a `simulate-principal-policy` run returning `implicitDeny` for `dynamodb:Scan`/`PutItem`, not a real denied API call

## 4. Transition the existing `tasks-api` stack

- [x] 4.1 Run a deploy with a real template or code change (the change that combines with this one) so an actual change set is produced; verify `sam deploy` succeeds and `aws cloudformation describe-stacks --stack-name tasks-api --query 'Stacks[0].RoleARN'` returns the service role ARN instead of empty
- [x] 4.2 Confirm the stack was updated in place: verify `StackStatus` is `UPDATE_COMPLETE` and the physical IDs of `TasksFunction`, `TasksTable`, and `ServerlessHttpApi` are unchanged from before the update
- [x] 4.3 Confirm no service interruption: verify a request against the existing API endpoint (or the repository integration tests) succeeds immediately after the update. The endpoint (`GET /tasks`) was verified after the deploy completed, not during the stack update

## 5. Verify the end state

- [x] 5.1 Confirm the deploy's resource operations ran under the service role rather than the CI identity: verify CloudTrail `lookup-events` for the deploy window shows the resource calls made via `TasksApiCloudFormationServiceRole`
- [x] 5.2 Confirm local/CI parity: verify a local `sam deploy` using `samconfig.toml` passes the same `role_arn` and the stack `RoleARN` is unchanged. Verified the local configuration passes `role_arn` and a local `sam deploy` used it; a real CI deploy was not executed, so full runtime parity remains unproven
- [x] 5.3 Confirm no broad policy remains anywhere: verify a repository-wide search finds no `PowerUserAccess` or `TasksApiReadExecutionRole` reference and that `ci.yml` still only sets `role-to-assume`

## Workflow follow-up

- Archive the change after the association is verified and the review requirements are satisfied.
- Verify the archived result.
