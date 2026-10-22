# Proposal: fix-service-role-apigateway-actions

## Why

The `ApiGatewayV2` statement of the CloudFormation service role grants IAM actions that
do not exist: `apigatewayv2:CreateApi`, `apigatewayv2:UpdateApi`, and the rest of the
SDK operation names. API Gateway Management V2 is authorized under the `apigateway`
service prefix with HTTP-verb action names (`apigateway:GET|POST|PUT|PATCH|DELETE`);
there is no `apigatewayv2` service prefix, and `apigateway:UpdateApi` is not an action
either. IAM accepts such a policy without any error but never matches a request, so the
statement grants nothing and the service role is denied every API Gateway operation.
The pending deploy that adds the `PATCH /tasks/{id}` route failed on `apigateway:PUT`
against `arn:aws:apigateway:us-east-2::/apis/3547pxd431` and left the `tasks-api` stack
in `UPDATE_ROLLBACK_COMPLETE`.

## What changes

- Replace the invalid action names in the `ApiGatewayV2` statement of
  `infra/ci/template.yaml` with the actions IAM recognizes for API Gateway Management
  V2: `apigateway:GET`, `apigateway:POST`, `apigateway:PUT`, `apigateway:PATCH`,
  `apigateway:DELETE`.
- Keep the statement's resource scope exactly as it is today: the application API ARN
  and that API's stages. No resource, service, or wildcard is added.
- Update the `tasks-api-ci` bootstrap stack so the service role's inline policy
  actually contains those actions (administrator action, executed on 2026-10-21).
- Re-run the `tasks-api` deployment so the pending API change is applied, and verify
  stack status, granted permissions, and physical resource identity before and after.

## Out of scope

- The application stack template `template.yaml`, the application code, the API
  contract, and the `PATCH` route itself: the route is already defined by the pending
  change and only needs the service role to be allowed to apply it.
- The rest of the service role policy (`Lambda`, `DynamoDB`, `ExecutionRole`,
  `InvokeServerlessTransform`, `ReadArtifacts`) and the CI deploy policy
  `TasksApiCiDeployPolicy`: they were exercised successfully in the same failed
  update and are not at fault.
- Widening the service role to other services, other APIs, or wildcards.
- Editing the archived `2026-10-20-harden-ci-deploy` change, which records the
  original design.

## Capability

`deployment-security` (modified).

## Impact

- `infra/ci/template.yaml`: one statement's action list, ten entries replaced by five.
  The resource list and every other statement are untouched.
- AWS IAM: the inline policy of `TasksApiCloudFormationServiceRole` changes only when
  the `tasks-api-ci` bootstrap stack is updated; the role itself is not replaced.
- AWS CloudFormation: `tasks-api` stays in `UPDATE_ROLLBACK_COMPLETE` until the
  bootstrap update lands and the deploy is re-run.
- No application code, runtime behavior, or data changes.
