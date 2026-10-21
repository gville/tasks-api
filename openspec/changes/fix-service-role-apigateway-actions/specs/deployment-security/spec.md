# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Deployment identity permissions are verified effective before they are relied upon

The permissions of the deployment identities SHALL be checked against the documented IAM action vocabulary of every service they name, and inspected on the live role, before a deployment is trusted to succeed, because a policy that names actions IAM does not recognize grants nothing while still passing syntax validation and linters.

#### Scenario: Unrecognized action names are detected before deployment

- **WHEN** the service role's policy is reviewed against the service authorization reference of each service it names
- **THEN** action names that IAM does not recognize for that service are found and corrected before the policy is applied

#### Scenario: The live role's permissions are confirmed

- **WHEN** the applied policy is inspected and the operations a deploy performs are simulated for the role
- **THEN** those operations are allowed for the role and operations outside the deploy scope remain denied
