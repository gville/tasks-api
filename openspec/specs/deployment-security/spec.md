# deployment-security Specification

## Purpose

Defines how the deployment pipeline authenticates to AWS and operates the application stack using least-privilege identities, so that deploys no longer depend on broad managed policies or manually pre-created resources.

## Requirements

### Requirement: Least-privilege CI deploy identity

The CI deploy identity `GitHubActionsTasksApiRole` SHALL hold only the permissions required to drive a deployment: CloudFormation stack operations on the designated stacks, read/write access to the SAM artifact bucket, `iam:PassRole` limited to the CloudFormation service role, and `sts:GetCallerIdentity`. It MUST NOT hold `PowerUserAccess` or any similarly broad managed policy.

#### Scenario: Deploy succeeds within the scoped identity

- **WHEN** CI runs `sam deploy` on a valid template change
- **THEN** the deployment completes using only the scoped permissions, without requiring `PowerUserAccess`

#### Scenario: Direct resource writes are denied

- **WHEN** the CI identity attempts a resource operation outside its scope, such as writing directly to DynamoDB
- **THEN** AWS denies the request

### Requirement: CloudFormation service role scoped to the application template

A CloudFormation service role SHALL exist whose permissions cover exactly the resource operations required by the current `template.yaml` (API Gateway HTTP API, Lambda function and permissions, DynamoDB table, and the function's IAM execution role), including the actions needed to create, update, and delete those resources, expressed with action names IAM recognizes for each service. It MUST NOT include permissions for resources not present in the current template.

#### Scenario: Stack resources are provisioned through the service role

- **WHEN** CloudFormation applies a stack update
- **THEN** it provisions and updates the application resources using the service role's credentials

#### Scenario: Out-of-scope service operations are denied

- **WHEN** the service role is asked to perform an operation for a resource type not defined in the current template, such as enabling DynamoDB point-in-time recovery
- **THEN** AWS denies the request

#### Scenario: API definition changes are applied under the service role

- **WHEN** a stack update changes the HTTP API definition, such as adding a route with its integration
- **THEN** CloudFormation applies the change with the service role's API Gateway permissions, without an access denial

### Requirement: Service role is assumable only by CloudFormation

The service role's trust policy SHALL allow assumption only by the `cloudformation.amazonaws.com` service principal.

#### Scenario: CloudFormation assumes the role

- **WHEN** CloudFormation executes a stack operation that passes the service role
- **THEN** it assumes the role successfully

#### Scenario: Non-CloudFormation principal is rejected

- **WHEN** any principal other than CloudFormation attempts to assume the service role
- **THEN** the assumption is denied

### Requirement: Existing stack is transitioned without recreation

The existing `tasks-api` stack SHALL be associated with the service role through a stack update that does not recreate or replace the stack, and does not interrupt the deployed API. The application's resources MUST retain their identity and the service MUST remain available during and after the transition.

#### Scenario: Stack gains the service role

- **WHEN** a deploy runs with the service role configured and a real template or code change
- **THEN** the stack associates the service role with the existing `tasks-api` stack without deleting or replacing it

#### Scenario: Service stays available

- **WHEN** the association update completes
- **THEN** the existing API endpoint and DynamoDB table continue to serve traffic with no recreation

### Requirement: Service role association is verified

After the transition, the service role association SHALL be verifiable by inspecting the stack, so that the assumption of least-privilege deploys can be confirmed rather than inferred.

#### Scenario: Confirm the association

- **WHEN** the stack is described after a deploy with the service role configured
- **THEN** the reported stack role is the CloudFormation service role, not empty

### Requirement: Deploy identities are defined as versioned infrastructure

The CI deploy identity's permissions and the CloudFormation service role SHALL be defined in versioned infrastructure as code, in a stack separate from the application stack, so the deployment configuration is reproducible and reviewed alongside the code. The GitHub OIDC provider and the initial bootstrap remain outside the application stack.

#### Scenario: Roles are reproducible from the repository

- **WHEN** the deployment configuration is applied to an account
- **THEN** the CI policy and the service role are created from the versioned definition, not configured by hand

### Requirement: Local and CI deploys use the same service role

The deploy configuration SHALL pass the same CloudFormation service role for both local runs and CI, so a deploy behaves identically regardless of where it is triggered.

#### Scenario: Local deploy uses the service role

- **WHEN** a developer runs `sam deploy` locally with the project configuration
- **THEN** the service role is passed and used for the stack operation

#### Scenario: CI deploy uses the service role

- **WHEN** CI runs `sam deploy` with the project configuration
- **THEN** the same service role is passed and used for the stack operation

### Requirement: Superseded broad and workaround policies are removed

The broad managed policy and the inline `iam:GetRole` workaround policy SHALL be removed from the CI deploy identity once the service role provides the equivalent capability.

#### Scenario: Prior policies are gone

- **WHEN** the CI deploy identity's policies are inspected after the transition
- **THEN** neither `PowerUserAccess` nor `TasksApiReadExecutionRole` remains attached

### Requirement: Deployment identity permissions are verified effective before they are relied upon

The permissions of the deployment identities SHALL be checked against the documented IAM action vocabulary of every service they name, and inspected on the live role, before a deployment is trusted to succeed, because a policy that names actions IAM does not recognize grants nothing while still passing syntax validation and linters.

#### Scenario: Unrecognized action names are detected before deployment

- **WHEN** the service role's policy is reviewed against the service authorization reference of each service it names
- **THEN** action names that IAM does not recognize for that service are found and corrected before the policy is applied

#### Scenario: The live role's permissions are confirmed

- **WHEN** the applied policy is inspected and the operations a deploy performs are simulated for the role
- **THEN** those operations are allowed for the role and operations outside the deploy scope remain denied
