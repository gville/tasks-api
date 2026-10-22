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

## 3. Update the bootstrap stack (administrator)

- [x] 3.1 Update the `tasks-api-ci` stack with the corrected template, using an identity authorized to modify the service role's inline policy (`iam:PutRolePolicy` on `TasksApiCloudFormationServiceRole`; the `process` profile is denied that action in this environment): applied with `--profile aws-admin` as a pre-reviewed change set (`create-change-set` + `execute-change-set` over `infra/ci/template.yaml`, `CAPABILITY_NAMED_IAM`, region `us-east-2`, change set `fix-apigateway-actions`) instead of `cloudformation deploy`, because the reviewed change set was a single `Modify`; verified `tasks-api-ci` reached `UPDATE_COMPLETE` (LastUpdatedTime `2026-10-21T22:07:02Z`) with no rollback, and the service role was not replaced (`Replacement: False`, `RequiresRecreation: Never`; `RoleId AROAXUIL4X6B3UGUWIHAF` and `CreateDate 2026-10-19T20:43:00+00:00` preserved)
- [x] 3.2 Inspect the live role: `aws iam get-role-policy --role-name TasksApiCloudFormationServiceRole --policy-name TasksApiCloudFormationServiceRolePolicy --profile process --region us-east-2` shows the five `apigateway:` actions (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`) and the unchanged resource list (`arn:aws:apigateway:us-east-2::/apis/3547pxd431` and `.../apis/3547pxd431/stages/*`); `TasksApiCiDeployPolicy` was left untouched (`v3`, `AttachmentCount: 1`, still attached to `GitHubActionsTasksApiRole`)
- [x] 3.3 Simulate the deploy's operations for the role: `aws iam simulate-principal-policy --policy-source-arn arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole --action-names apigateway:GET apigateway:POST apigateway:PUT apigateway:PATCH apigateway:DELETE --resource-arns arn:aws:apigateway:us-east-2::/apis/3547pxd431 arn:aws:apigateway:us-east-2::/apis/3547pxd431/stages/\$default --profile process --region us-east-2`; every pair evaluates to `allowed`; recorded limitation: the simulator normalizes API Gateway resource ARNs to `arn:aws:apigateway:${Region}::${ApiGatewayResourcePath}`, so it validates the action level only, not the sub-resource

## 4. Re-run the deployment

- [x] 4.1 Record the pre-deploy state: `describe-stacks` showed `UPDATE_ROLLBACK_COMPLETE` with `RoleARN` = `arn:aws:iam::524547833731:role/TasksApiCloudFormationServiceRole`, and `describe-stack-resources` showed the physical IDs `3547pxd431` (API), `$default` (stage), `tasks-api-TasksFunction-woOv7SWZ5JEV` (function) and `tasks-api-TasksTable-15WCT1VKKIOF5` (table); the last event was the `apigateway:PUT` `AccessDeniedException` on `ServerlessHttpApi` (2026-10-21T20:15:49Z)
- [x] 4.2 Re-run the deploy workflow for `tasks-api` so the pending `PATCH /tasks/{id}` route is applied: the prepared change set `samcli-deploy1792621692` was executed (2026-10-21T23:16:19Z → 23:16:38Z); it carried `Modify ServerlessHttpApi`, `Modify TasksFunction` (code) and `Add TasksFunctionPatchTaskPermission`, with no replacement on any resource
- [x] 4.3 Confirm the update completed: `describe-stacks` reports `UPDATE_COMPLETE` (`LastUpdatedTime` 2026-10-21T23:16:38.728Z) and `describe-stack-events` shows no `AccessDeniedException` on an `apigateway:` action (the 20:15:49 denial is no longer the tail of the event history)

## 5. Post-deploy verification

- [x] 5.1 `describe-stack-resources` shows the same physical IDs as in task 4.1: the API `3547pxd431`, the stage `$default`, the function `tasks-api-TasksFunction-woOv7SWZ5JEV` and the table `tasks-api-TasksTable-15WCT1VKKIOF5` were updated in place, not replaced (`Action: None` for every resource)
- [x] 5.2 `get-template` shows the API body now contains the `patch` operation for `/tasks/{id}`: `/tasks` exposes `get`/`post` and `/tasks/{id}` exposes `delete`/`get`/`patch`/`put`
- [x] 5.3 The API serves the new route: a request to `PATCH /tasks/{id}` with an empty body returns `400` (not `404`), confirming the route is deployed (verified manually against the deployed endpoint with the Bruno collection, together with the PUT replacement and partial-update behaviour)
- [x] 5.4 The deployed service role policy is re-inspected with `get-role-policy` to confirm the applied policy matches the versioned definition in `infra/ci/template.yaml`: the live inline policy has the same 6 statements, with the `ApiGatewayV2` statement listing `apigateway:GET`, `apigateway:POST`, `apigateway:PUT`, `apigateway:PATCH` and `apigateway:DELETE` on the unchanged `arn:aws:apigateway:us-east-2::/apis/3547pxd431` and `.../stages/*`; the operations the deploy performs simulate to `allowed`, while out-of-scope ones (`apigateway:POST`/`DELETE` on the `.../apis` collection, `dynamodb:UpdateContinuousBackups`, `dynamodb:UpdateTimeToLive`) still simulate to `implicitDeny`
