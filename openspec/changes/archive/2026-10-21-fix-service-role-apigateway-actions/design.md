# Design: fix-service-role-apigateway-actions

## Context

See `proposal.md` for motivation. Current state verified against the live account
(region `us-east-2`, account `524547833731`) on 2026-10-21 with read-only commands:

- The `ApiGatewayV2` statement of `TasksApiCloudFormationServiceRole` (inline policy
  `TasksApiCloudFormationServiceRolePolicy`; no managed policy attached) lists
  `apigatewayv2:CreateApi`, `apigatewayv2:UpdateApi`, `apigatewayv2:GetApi`,
  `apigatewayv2:DeleteApi`, `apigatewayv2:CreateStage`, `apigatewayv2:UpdateStage`,
  `apigatewayv2:GetStage`, `apigatewayv2:DeleteStage`, `apigatewayv2:TagResource` and
  `apigatewayv2:UntagResource`, on `arn:aws:apigateway:us-east-2::/apis/3547pxd431`
  and `arn:aws:apigateway:us-east-2::/apis/3547pxd431/stages/*`.
- IAM has no `apigatewayv2` service prefix. API Gateway Management V2 is authorized
  under `apigateway` with HTTP-verb action names, and the SDK operation names are not
  actions either. The statement therefore matches no request and grants nothing.
  `iam simulate-principal-policy` confirms it: `apigateway:GET|POST|PUT|PATCH|DELETE`
  on the API ARN all evaluate to `implicitDeny` for the role, while the same action
  evaluates to `allowed` for the administrator identity that bootstrapped the stack.
- The failed update (`tasks-api` → `UPDATE_ROLLBACK_COMPLETE`, 2026-10-21) failed on
  `ServerlessHttpApi` with `apigateway:PUT` denied on
  `arn:aws:apigateway:us-east-2::/apis/3547pxd431`. `TasksFunction` in the same
  change set updated successfully, so the other statements of the role do work.
- The API is defined entirely by its OpenAPI `Body`: the paths and the Cognito JWT
  `securityScheme` live there. The stack has no `AWS::ApiGatewayV2::Route`,
  `Integration`, `Authorizer` or `Deployment` resources, and its only stage is
  `$default` with `AutoDeploy: true`.
- Because the definition lives in the `Body`, CloudFormation applies an API change
  with `ReimportApi`, which is authorized with `apigateway:DELETE` + `apigateway:POST`
  + `apigateway:PUT` (plus `iam:PassRole`) on the API ARN. That is the operation the
  denied `apigateway:PUT` came from.
- No validator available here catches this: `sam validate --lint` passes the broken
  template, `aws accessanalyzer validate-policy --policy-type IDENTITY_POLICY` returns
  zero findings for a probe policy containing `apigateway:UpdateApi` and
  `apigatewayv2:CreateApi`, and the IAM simulator reports API Gateway resources as the
  template `arn:aws:apigateway:${Region}::${ApiGatewayResourcePath}`, so it cannot
  confirm per-sub-resource scoping.

## Goals

- Make the service role's API Gateway permissions effective for the operations the
  application template performs, expressed with action names IAM recognizes.
- Keep the statement's resource scope exactly as it is today: no new resource, no new
  service, no wildcard.
- Make the repair verifiable: the policy is checked before the deploy is trusted, and
  the deploy's effect on status, permissions, and physical resource identity is
  recorded.

## Decisions

### D1: Replace the action names with the five verb-style `apigateway:` actions

Alternative A was to keep the SDK operation names and only correct the prefix
(`apigateway:UpdateApi`, `apigateway:CreateApi`, ...). Rejected: those are not IAM
actions, so the same trap would recur with a less obvious symptom.

Alternative B was `apigateway:*` on the API ARN. Rejected: it grants every API Gateway
action on that API, including the operations this design deliberately excludes, and it
hides which operations the role actually needs.

Chosen: `apigateway:GET`, `apigateway:POST`, `apigateway:PUT`, `apigateway:PATCH` and
`apigateway:DELETE`. These are exactly the actions the documented operations of this
stack need (`GetApi`, `ReimportApi`, `UpdateApi`, `TagResource`, `GetStage`,
`UpdateStage`, `CreateStage`, `DeleteStage`, `UntagResource`). API Gateway v2 offers no
finer granularity than the verb set; the resource ARN is what narrows the scope.

### D2: Keep the resource list unchanged

The statement already scopes to the API ARN and that API's stages, and those are the
only API Gateway resources the stack manages. The sub-resources are deliberately not
added: routes and integrations are created inside the `ReimportApi` call, which IAM
evaluates against the API ARN; the Cognito authorizer is a `securityScheme` inside the
OpenAPI body rather than a stack resource; and the stage has `AutoDeploy: true`, so no
`Deployment` resource exists. The `.../apis` collection ARN is left out on purpose:
`CreateApi` and `DeleteApi` are evaluated against the collection, so omitting it keeps
the role from creating or deleting APIs that are not part of this project.

### D3: `apigateway:DELETE` stays in the set

`ReimportApi` requires DELETE + POST + PUT on the API ARN, and the stage's delete path
(`DeleteStage`) needs it too. The denied `PUT` was reported first only because IAM
stopped at the first failing action of the operation, so the absence of a `DELETE`
denial is not evidence that it is unnecessary.

### D4: No `iam:PassRole` change

`ReimportApi` and `UpdateApi` list `iam:PassRole` for `apigateway.amazonaws.com`, but
this API has no `CredentialsArn` and its integrations are `aws_proxy` without
credentials, so no role is passed to API Gateway. The `ExecutionRole` statement already
allows `iam:PassRole` on the function role. A future change that gives the API its own
execution role must add that grant, scoped with `iam:PassedToService`.

### D5: The repair is delivered in two steps, and only the first one lives in the repository

Editing `infra/ci/template.yaml` changes nothing in AWS by itself: the policy is an
inline policy of the `AWS::IAM::Role` in the `tasks-api-ci` bootstrap stack, so it
takes effect only when that stack is updated by an administrator. The application deploy
is then re-run to apply the pending `PATCH /tasks/{id}` route. The steps are separated
because the second one is what proves the first one.

### D6: Verification is read-only and brackets the deploy

Before the deploy: `get-role-policy` shows the new actions, and
`simulate-principal-policy` on the role shows `allowed` for the actions the deploy
performs. After the deploy: `describe-stacks` and `describe-stack-events` show the
update completed without an access denial, `describe-stack-resources` shows the same
physical IDs, and the API answers the new route. `UPDATE_ROLLBACK_COMPLETE` is a
stable state that accepts a new update, so no `ContinueUpdateRollback` is needed.

## Alternatives considered

- `apigateway:*` on `arn:aws:apigateway:us-east-2::/apis/*`: simpler to maintain, but it
  lets the role operate APIs outside this project and hides the operation set; rejected
  as a least-privilege regression.
- Adding `routes/*`, `integrations/*`, `authorizers/*` and `deployments/*`: no evidence
  justifies them (see D2); rejected as unjustified widening.
- Replacing the hardcoded API ARN with a wildcard (`/apis/*`): it would survive an API
  replacement, but it grants access to every API in the account; rejected, consistent
  with the least-privilege trade-off already accepted by the deployment-security design.
- Correcting the policy inside the archived `2026-10-20-harden-ci-deploy` change, which
  documents the original (broken) policy: archived changes are immutable records;
  rejected.

## Risks

- [The `apigateway:DELETE` part of `ReimportApi` is inferred from the documentation, not
  observed] → The deploy cut off at the denied `PUT`, so `DELETE` was never evaluated.
  It is included because the same operation requires it; if the next deploy denies it,
  the message names the exact action and ARN to adjust.
- [The IAM simulator cannot confirm the sub-resource scoping] → API Gateway resource
  ARNs are normalized to `arn:aws:apigateway:${Region}::${ApiGatewayResourcePath}` in
  simulation results, and `simulate-custom-policy` fails in this account even for AWS
  managed policies, so the candidate could not be simulated without attaching it. The
  simulation is therefore used as a level-of-action check only; the deploy is the real
  test, and the resource scope is justified by the deployed template's resource
  inventory.
- [The hardcoded API ARN goes stale if the API is replaced] → Accepted trade-off of the
  least-privilege design: a replacement requires a coordinated update of `infra/ci/`
  first.
- [The bootstrap update is outside the repository's reach] → The policy change is inert
  until an administrator updates `tasks-api-ci`; the tasks record that dependency
  explicitly.
- [A deploy that changes the API may need `iam:PassRole` for API Gateway] → Not needed
  today; D4 records the trigger and the shape of the required grant.
- [No available tooling detects unknown action names] → `sam validate --lint` and
  Access Analyzer both pass the broken policy, so the same class of defect can recur
  silently. A repository check that fails when an action name is absent from the
  service's documented vocabulary would prevent that, but it is outside this change's
  scope.

## Validation

Executed locally on the corrected template:

- `sam validate --lint --region us-east-2 --template infra/ci/template.yaml` → valid
  SAM template.
- `grep -rn "apigatewayv2:"` over the repository → no matches left; the only API
  Gateway actions in the file are the five `apigateway:` verbs.
- `git diff --check` → clean.
- `openspec validate --strict` over the change and the specs → passing.

Executed in AWS and tracked as tasks: the `tasks-api-ci` bootstrap update (task 3.1),
the role inspection and simulation (tasks 3.2 and 3.3), the re-run of the deploy
(tasks 4.1 to 4.3), and the post-deploy verification of status and physical resource
identity (tasks 5.1 to 5.4).
