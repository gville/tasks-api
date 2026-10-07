#!/bin/bash

set -e

trap 'rm -f env.json' EXIT

TABLE_NAME=$(aws cloudformation describe-stacks \
    --stack-name tasks-api \
    --query "Stacks[0].Outputs[?OutputKey=='TasksTableName'].OutputValue" \
    --output text)

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
