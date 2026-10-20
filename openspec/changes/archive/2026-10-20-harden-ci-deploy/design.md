# Design

## Context

See `proposal.md` for motivation. Current state verified against the live account:

- `tasks-api` stack (region `us-east-2`, account `524547833731`) has 10 resources, all CloudFormation-managed: `ServerlessHttpApi` (`3547pxd431`) + stage, `TasksFunction` (`tasks-api-TasksFunction-woOv7SWZ5JEV`) + 5 `AWS::Lambda::Permission`, `TasksFunctionRole` (`tasks-api-TasksFunctionRole-Up1QN910ZA0J`), and `TasksTable` (`tasks-api-TasksTable-15WCT1VKKIOF5`). The 5 HTTP routes live inside the API's OpenAPI `Body`, so there are no standalone route resources.
- The stack's `RoleARN` is currently empty; it has never used a service role.
- `GitHubActionsTasksApiRole` (OIDC-federated, trust scoped to `repo:gville@8316152/tasks-api@1414858068:ref:refs/heads/main`) carries `PowerUserAccess` plus an inline `TasksApiReadExecutionRole` granting `iam:GetRole` on the function role.
- SAM artifacts live in the bootstrap bucket `aws-sam-cli-managed-default-samclisourcebucket-0rgjdmzxq67e` (its stack predates this change and is SAM bootstrap, not application infrastructure).
- `sam deploy` supports `--role-arn`, and CloudFormation `UpdateStack.RoleARN` persists: "CloudFormation always uses this role for all future operations on the stack." The association is one-way, it cannot be removed after set.

## Goals / Non-Goals

**Goals:**

- Deploy via a least-privilege CloudFormation service role; remove `PowerUserAccess` from CI.
- Keep deployment configuration versioned and separate from the application stack.
- Transition the existing stack in place: no recreation, no service interruption, verifiable association.

**Non-Goals:**

- DynamoDB/Cognito hardening (PITR, deletion protection, log retention, MFA, tracing). The service role deliberately excludes those permissions; they belong to a later change.
- Application/API behavior, `template.yaml` resources, and DynamoDB data model.
- Replacing the SAM CLI deploy mechanism or introducing a multi-stage pipeline.

## Decisions

### D1: CloudFormation service role, not a scoped identity policy or `sam pipeline bootstrap`

Alternative A was an inline least-privilege policy on the CI role covering every resource operation. It is simpler (one role) but brittle: the CI identity accumulates resource permissions and grows with each new resource type. Alternative C (`sam pipeline bootstrap`) generates an OIDC provider, pipeline role, CFN role, and bucket, but imposes a full pipeline structure heavier than a single-stage learning project needs.

Chosen: a service role. CloudFormation assumes it for resource operations; the CI identity shrinks to "operate the stack + pass one role". This is the AWS-recommended pattern, and because `template.yaml` already manages an `AWS::IAM::Role`, it also removes the latent AccessDenied failure described in the proposal.

### D2: Roles defined in a separate versioned stack under `infra/ci/`

The service role cannot be created by the stack it deploys (chicken-and-egg), so both roles live in their own stack, `infra/ci/`, deployed once by a human administrator. The GitHub OIDC provider already exists and is left out of version control to avoid any risk of recreation; it is documented as bootstrap. This keeps `template.yaml` (the application) untouched, matching the app-vs-deployment separation.

### D3: Service role scoped to the current `template.yaml` only

Permissions are derived resource-by-resource from the current template, including delete actions so a future `delete-stack` still works through the role. No future hardening permissions are included.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ApiGatewayV2",
      "Effect": "Allow",
      "Action": [
        "apigatewayv2:CreateApi", "apigatewayv2:UpdateApi", "apigatewayv2:GetApi", "apigatewayv2:DeleteApi",
        "apigatewayv2:CreateStage", "apigatewayv2:UpdateStage", "apigatewayv2:GetStage", "apigatewayv2:DeleteStage",
        "apigatewayv2:TagResource", "apigatewayv2:UntagResource"
      ],
      "Resource": [
        "arn:aws:apigateway:us-east-2::/apis/3547pxd431",
        "arn:aws:apigateway:us-east-2::/apis/3547pxd431/stages/*"
      ]
    },
    {
      "Sid": "Lambda",
      "Effect": "Allow",
      "Action": [
        "lambda:CreateFunction", "lambda:UpdateFunctionCode", "lambda:UpdateFunctionConfiguration",
        "lambda:GetFunction", "lambda:GetFunctionConfiguration", "lambda:DeleteFunction",
        "lambda:ListVersionsByFunction", "lambda:AddPermission", "lambda:RemovePermission",
        "lambda:TagResource", "lambda:UntagResource"
      ],
      "Resource": "arn:aws:lambda:us-east-2:524547833731:function:tasks-api-TasksFunction-woOv7SWZ5JEV"
    },
    {
      "Sid": "DynamoDB",
      "Effect": "Allow",
      "Action": [
        "dynamodb:CreateTable", "dynamodb:UpdateTable", "dynamodb:DeleteTable", "dynamodb:DescribeTable",
        "dynamodb:TagResource", "dynamodb:UntagResource"
      ],
      "Resource": "arn:aws:dynamodb:us-east-2:524547833731:table/tasks-api-TasksTable-15WCT1VKKIOF5*"
    },
    {
      "Sid": "ExecutionRole",
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole", "iam:DeleteRole", "iam:GetRole", "iam:UpdateAssumeRolePolicy",
        "iam:PutRolePolicy", "iam:DeleteRolePolicy", "iam:AttachRolePolicy", "iam:DetachRolePolicy",
        "iam:TagRole", "iam:UntagRole", "iam:PassRole"
      ],
      "Resource": "arn:aws:iam::524547833731:role/tasks-api-TasksFunctionRole-Up1QN910ZA0J"
    },
    {
      "Sid": "InvokeServerlessTransform",
      "Effect": "Allow",
      "Action": "cloudformation:CreateChangeSet",
      "Resource": "arn:aws:cloudformation:us-east-2:aws:transform/Serverless-2016-10-31"
    },
    {
      "Sid": "ReadArtifacts",
      "Effect": "Allow",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::aws-sam-cli-managed-default-samclisourcebucket-0rgjdmzxq67e/*"
    }
  ]
}
```

`iam:PassRole` here lets CloudFormation hand `TasksFunctionRole` to Lambda when creating the function. `iam:GetRole` and the role-management actions replace the CI inline workaround.

The policy has **six** statements in its deployed form. The first four (`ApiGatewayV2`, `Lambda`, `DynamoDB`, `ExecutionRole`) were derived from the template from the start. The final two were added during the first real deploy that associated the role with the existing stack, because the operations they authorize are performed by the service role only once CloudFormation operates the stack under it (previously the caller's broad identity covered them):

- `InvokeServerlessTransform` (`cloudformation:CreateChangeSet` on `arn:aws:cloudformation:us-east-2:aws:transform/Serverless-2016-10-31`): `template.yaml` declares `Transform: AWS::Serverless-2016-10-31`, so CloudFormation must invoke the SAM macro while creating a change set. With a service role attached, that invocation is authorized against the service role, and without this statement the change set failed with `AccessDenied` on the transform resource.
- `ReadArtifacts` (`s3:GetObject` on the SAM artifact bucket prefix): `AWS::Lambda::Function` stores its code as an S3 object, so whoever updates the resource must read the artifact. Under the service role this read is performed by the role; without it, `UpdateFunctionCode` failed with `s3:GetObject` AccessDenied, which also prevented the rollback from completing and left the stack in `UPDATE_ROLLBACK_FAILED` until the statement was added and the rollback resumed.

Both are the minimum needed for the deploy path and are scoped to a single transform ARN and a single bucket prefix respectively; they are not broadening beyond the current template.

Trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    { "Effect": "Allow", "Principal": { "Service": "cloudformation.amazonaws.com" }, "Action": "sts:AssumeRole" }
  ]
}
```

### D4: CI identity scoped policy

Replaces `PowerUserAccess`. The CloudFormation actions cover the application stack and the SAM-managed bootstrap stack (needed by `resolve_s3 = true`); the S3 statements cover only the artifact bucket; `iam:PassRole` is limited to the service role.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "OperateStacks",
      "Effect": "Allow",
      "Action": [
        "cloudformation:CreateChangeSet", "cloudformation:DeleteChangeSet", "cloudformation:DescribeChangeSet",
        "cloudformation:DescribeStacks", "cloudformation:DescribeStackEvents", "cloudformation:DescribeStackResources",
        "cloudformation:ExecuteChangeSet", "cloudformation:GetTemplate", "cloudformation:ListChangeSets",
        "cloudformation:ListStackResources"
      ],
      "Resource": [
        "arn:aws:cloudformation:us-east-2:524547833731:stack/tasks-api/*",
        "arn:aws:cloudformation:us-east-2:524547833731:stack/aws-sam-cli-managed-default/*"
      ]
    },
    {
      "Sid": "TemplateValidation",
      "Effect": "Allow",
      "Action": ["cloudformation:GetTemplateSummary", "cloudformation:ValidateTemplate"],
      "Resource": "*"
    },
    {
      "Sid": "Artifacts",
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::aws-sam-cli-managed-default-samclisourcebucket-0rgjdmzxq67e/*"
    },
    {
      "Sid": "ArtifactBucket",
      "Effect": "Allow",
      "Action": ["s3:ListBucket", "s3:GetBucketLocation"],
      "Resource": "arn:aws:s3:::aws-sam-cli-managed-default-samclisourcebucket-0rgjdmzxq67e"
    },
    {
      "Sid": "PassServiceRole",
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole",
      "Condition": { "StringEquals": { "cloudformation:RoleARN": "arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole" } }
    },
    {
      "Sid": "CallerIdentity",
      "Effect": "Allow",
      "Action": "sts:GetCallerIdentity",
      "Resource": "*"
    }
  ]
}
```

`GetTemplateSummary`/`ValidateTemplate` need `Resource: "*"` because they can be invoked against a template body with no stack. If the SAM-managed bucket or stack is ever lost, it is re-bootstrapped by hand; the CI role intentionally cannot recreate it.

### D5: `role_arn` in `samconfig.toml`, not `ci.yml`

The service role is added under `[default.deploy.parameters]` as `role_arn`, so local and CI deploys pass it identically without duplicating flags in the workflow. Alternative was an explicit `--role-arn` in `ci.yml`, which would leave local runs inconsistent.

### D6: Transition of the existing stack

Association is a stack attribute, not a resource, so it never recreates resources. Two constraints drive the order:

- `PowerUserAccess` denies `iam:PassRole`, so CI cannot associate the role until the CI policy is fixed.
- A no-op update produces an empty change set; `sam deploy` does not associate the role in that case. Association therefore happens on the first deploy with a real template or code change.

Ordering: create `infra/ci/` → bootstrap-deploy it with the admin profile → set `role_arn` in `samconfig.toml` → swap the CI policy (attach the scoped policy first, then detach `PowerUserAccess` and delete the inline policy) → run a real deploy → verify. This transition is the reason the change is a prerequisite for later infrastructure hardening: the hardening deploy is the natural first real change that performs the association.

## Risks / Trade-offs

- **One-way association** → Once set, the stack's service role cannot be removed; every future stack operator inherits it. Mitigation: keep the role minimal and confined to `tasks-api-*` resources (D3), and treat `infra/ci/` as sensitive code under review.
- **Scoped CI policy may be incomplete** → Missing an action breaks the deploy. Mitigation: rollback is re-attaching `PowerUserAccess` while iterating the policy; the association itself is harmless. Verify with a real deploy before relying on it.
- **Empty change set does not associate** → A no-op push leaves `RoleARN` unset. Mitigation: perform the association as part of a real change and verify with `describe-stacks`; document that association is not a standalone `update-stack`.
- **Delete/rollback use the role** → A future `delete-stack` runs as the service role. Mitigation: D3 includes delete actions for every resource type.
- **Service role policy is bound to current physical ARNs** → D3 scopes permissions to the existing physical identifiers (`.../TasksFunction-woOv7SWZ5JEV`, `.../TasksTable-15WCT1VKKIOF5`, `.../apis/3547pxd431`, `.../TasksFunctionRole-Up1QN910ZA0J`). If a future CloudFormation change causes a resource replacement, CloudFormation creates a new physical ARN that the policy does not cover, and the deployment can fail until `infra/ci/` is updated to include the new identifier. This is a deliberate consequence of least privilege, not a defect: the trade is a narrower, explicit policy in exchange for a pre-deployment policy update whenever a resource is replaced. Mitigation: treat resource replacements as a coordinated change that updates `infra/ci/` first; broaden the ARN patterns (for example to `tasks-api-TasksTable-*`) only if replacements become routine.
- **Bootstrap stack loss** → Losing the SAM-managed bucket/stack is not self-healing. Mitigation: accepted; re-bootstrap manually (documented), not an application concern.

## Migration Plan

1. Add `infra/ci/` CloudFormation template defining the service role (D3) and the CI identity policy (D4).
2. Bootstrap-deploy `infra/ci/` once with an admin profile.
3. Add `role_arn = "arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole"` to `[default.deploy.parameters]` in `samconfig.toml`; commit.
4. Update `GitHubActionsTasksApiRole`: attach the scoped policy, then detach `PowerUserAccess` and delete `TasksApiReadExecutionRole`.
5. Run a deploy with a real template or code change; this associates the service role.
6. Verify: `describe-stacks` reports the service role as `RoleARN`; CloudTrail shows resource operations performed under the service role; an out-of-scope call as CI (e.g. direct DynamoDB write) is denied.

Rollback: if the scoped CI policy proves insufficient, re-attach `PowerUserAccess` and iterate the policy. The service role association stays but is inert while unused. Full rollback to the prior state is not possible for the association, which is acceptable because it does not affect runtime.
