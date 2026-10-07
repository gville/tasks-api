# Tasks API

Serverless REST API built with Go and AWS.

This is a learning project focused on practicing AWS services and cloud-native backend development.

## Architecture

```text
Client
  │
  ▼
API Gateway HTTP API
  │
  ▼
Lambda (Go)
  │
  ▼
Service
  │
  ▼
Repository
  │
  ▼
DynamoDB
```

## Requirements

* Go
* AWS CLI
* AWS SAM CLI
* Docker
* AWS account

The project uses `us-east-2` by default.

## AWS profile

The default AWS profile is `process`.

It can be overridden when needed:

```bash
AWS_PROFILE=other-profile make deploy
```

AWS credentials are not stored in the repository.

## Commands

### Build

```bash
make build
```

Builds the application using AWS SAM.

### Test

```bash
make test
```

Runs the Go tests.

### Deploy

```bash
make deploy
```

Builds and deploys the application to AWS using the `tasks-api` CloudFormation stack.

### Local

```bash
make local
```

Starts the API locally using SAM Local.

The local environment uses the DynamoDB table created by the deployed CloudFormation stack.

## Infrastructure

AWS infrastructure is defined with AWS SAM in:

```text
template.yaml
```

CloudFormation manages the Lambda function, API Gateway, DynamoDB table and their permissions.

## Cleanup

To remove the deployed stack:

```bash
sam delete --stack-name tasks-api --region us-east-2
```
