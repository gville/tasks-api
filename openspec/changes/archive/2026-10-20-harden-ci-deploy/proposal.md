# Proposal

## Why

The GitHub Actions deploy role `GitHubActionsTasksApiRole` currently carries the AWS managed policy `PowerUserAccess`, which violates least privilege by granting broad permissions across almost every service. It also hides a latent deploy failure: `PowerUserAccess` denies all `iam` actions, yet the `tasks-api` stack manages its own `AWS::IAM::Role`. CI can only update that role because it was originally created by a human with broader permissions; the moment a template change touches the role (for example enabling X-Ray tracing), `sam deploy` will fail with AccessDenied and roll the stack back. This must be resolved before any further infrastructure hardening.

## What Changes

- Introduce a **CloudFormation service role** that grants CloudFormation exactly the permissions required by the current `template.yaml`, defined as its own versioned stack under `infra/ci/` (separate from the application stack).
- Replace `PowerUserAccess` on `GitHubActionsTasksApiRole` with a scoped policy: CloudFormation stack operations, S3 artifact access, `iam:PassRole` limited to the service role, and `sts:GetCallerIdentity`. Remove the inline `TasksApiReadExecutionRole` policy (an `iam:GetRole` workaround now superseded by the service role).
- Configure `role_arn` in `samconfig.toml` so both local runs and CI pass the same service role to `sam deploy`.
- **BREAKING (deploy-time, not runtime)**: any future deploy must use the service role; the CI identity no longer holds direct resource-write permissions.
- Transition the **existing** `tasks-api` stack to use the service role without recreating it or interrupting the service, and verify the association.
- Keep the OIDC provider and the one-time role creation outside the application stack, documented as bootstrap.

## Capabilities

### New Capabilities

- `deployment-security`: defines how the deployment pipeline authenticates to AWS, which identities and permissions it uses, and how the application stack is operated through a least-privilege CloudFormation service role, including the required transition of the existing stack.

### Modified Capabilities

None. No existing specs; application runtime behavior is unchanged.

## Impact

- `.github/workflows/ci.yml`: `role-to-assume` is unchanged; the job relies on the new scoped permissions.
- `samconfig.toml`: adds `role_arn` for local and CI parity.
- New `infra/ci/`: CloudFormation template defining the CI role policy and the CloudFormation service role.
- AWS IAM: `GitHubActionsTasksApiRole` policy replaced; a new service role created; `TasksApiReadExecutionRole` removed.
- AWS CloudFormation: the existing `tasks-api` stack gains a `RoleARN` association.
- No changes to application resources in `template.yaml`, no API behavior, no DynamoDB/Cognito changes.
