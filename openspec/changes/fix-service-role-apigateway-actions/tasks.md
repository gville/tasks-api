# Tasks: fix-service-role-apigateway-actions

## 1. Correct the service role policy

- [x] 1.1 Replace the invalid action names in the `ApiGatewayV2` statement of `infra/ci/template.yaml` with `apigateway:GET`, `apigateway:POST`, `apigateway:PUT`, `apigateway:PATCH` and `apigateway:DELETE`; verified the statement's resource list (`.../apis/3547pxd431` and `.../apis/3547pxd431/stages/*`) is unchanged and no other statement was modified
- [x] 1.2 Verify no reference to `apigatewayv2:` remains in the template: `grep -rn "apigatewayv2:" infra/ template.yaml` returns no matches, and `grep -n "apigateway:" infra/ci/template.yaml` lists only the five verbs
- [x] 1.3 Confirm the deployed role is affected only through the bootstrap stack: `aws iam get-role` shows the trust policy limited to `cloudformation.amazonaws.com`, `list-role-policies` shows the single inline policy `TasksApiCloudFormationServiceRolePolicy`, and `list-attached-role-policies` shows no managed policy

## 2. Local validation

- [x] 2.1 `sam validate --lint --region us-east-2 --template infra/ci/template.yaml` → "is a valid SAM Template"
- [x] 2.2 `git diff --cached --check` → clean (checks the staged changes; the working tree carries no unstaged modifications)
- [x] 2.3 `openspec validate --strict` for this change and for the specifications → passing
- [x] 2.4 `go test ./...` and `gofmt`/`go vet` → unaffected by this change, no application code touched

## 3. Update the bootstrap stack (administrator, pending)

- [ ] 3.1 Update the `tasks-api-ci` stack with the corrected template: `aws cloudformation deploy --stack-name tasks-api-ci --template-file infra/ci/template.yaml --capabilities CAPABILITY_NAMED_IAM --region us-east-2 --profile process`; verify it reaches `UPDATE_COMPLETE` and that the service role is not replaced
- [ ] 3.2 Inspect the live role: `aws iam get-role-policy --role-name TasksApiCloudFormationServiceRole --policy-name TasksApiCloudFormationServiceRolePolicy --profile process --region us-east-2` shows the five `apigateway:` actions and the unchanged resource list
- [ ] 3.3 Simulate the deploy's operations for the role: `aws iam simulate-principal-policy --policy-source-arn arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole --action-names apigateway:GET apigateway:POST apigateway:PUT apigateway:PATCH apigateway:DELETE --resource-arns arn:aws:apigateway:us-east-2::/apis/3547pxd431 arn:aws:apigateway:us-east-2::/apis/3547pxd431/stages/\$default --profile process --region us-east-2`; verify every pair evaluates to `allowed`, and record the limitation that the simulator normalizes API Gateway resource ARNs, so it validates the action level only

## 4. Re-run the deployment (pending)

- [ ] 4.1 Record the pre-deploy state: `describe-stacks` shows `UPDATE_ROLLBACK_COMPLETE` with `RoleARN` set, and `describe-stack-resources` shows the physical IDs `3547pxd431` (API), `$default` (stage), `tasks-api-TasksFunction-woOv7SWZ5JEV` (function) and `tasks-api-TasksTable-15WCT1VKKIOF5` (table)
- [ ] 4.2 Re-run the deploy workflow for `tasks-api` so the pending `PATCH /tasks/{id}` route is applied
- [ ] 4.3 Confirm the update completed: `describe-stacks` reports `UPDATE_COMPLETE` and `describe-stack-events` shows no `AccessDeniedException` on an `apigateway:` action

## 5. Post-deploy verification (pending)

- [ ] 5.1 `describe-stack-resources` shows the same physical IDs as in task 4.1: the API, the stage, the function and the table were updated in place, not replaced
- [ ] 5.2 `get-template` shows the API body now contains the `patch` operation for `/tasks/{id}`
- [ ] 5.3 The API serves the new route: a request to `PATCH /tasks/{id}` with an empty body returns `400` (not `404`), confirming the route is deployed
- [ ] 5.4 The deployed service role policy is re-inspected with `get-role-policy` to confirm the applied policy matches the versioned definition in `infra/ci/template.yaml`
