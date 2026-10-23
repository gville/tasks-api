#!/bin/bash

set -e

trap 'rm -f env.json' EXIT

TABLE_NAME=$(aws cloudformation describe-stacks \
    --stack-name tasks-api \
    --query "Stacks[0].Outputs[?OutputKey=='TasksTableName'].OutputValue" \
    --output text)

if [ -z "$TABLE_NAME" ] || [ "$TABLE_NAME" = "None" ]; then
    echo "Could not read TasksTableName from the tasks-api stack." >&2
    echo "Deploy the stack first and verify AWS_PROFILE and AWS_REGION." >&2
    exit 1
fi

cat > env.json <<EOF
{
  "TasksFunction": {
    "TASKS_TABLE_NAME": "$TABLE_NAME"
  }
}
EOF

sam local start-api \
    --skip-pull-image \
    --env-vars env.json
