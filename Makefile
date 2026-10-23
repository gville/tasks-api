AWS_PROFILE ?= default

.PHONY: build test deploy local

build:
	sam build

test:
	go test ./...

deploy: build
	sam deploy --profile $(AWS_PROFILE)

local:
	AWS_PROFILE=$(AWS_PROFILE) ./scripts/local.sh

build-TasksFunction:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/tasks-api
