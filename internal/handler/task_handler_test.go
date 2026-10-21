package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"tasks-api/internal/model"
	"tasks-api/internal/repository"
	"tasks-api/internal/service"

	"github.com/aws/aws-lambda-go/events"
)

type mockTaskService struct {
	createdTask model.Task

	task              model.Task
	tasks             []model.Task
	updatedTask       model.Task
	updateInput       model.Task
	patchedUpdate     model.TaskUpdate
	patchedID         string
	deletedID         string
	createTitle       string
	createDescription string
	getID             string

	createError   error
	getError      error
	getTasksError error
	updateError   error
	patchError    error
	deleteError   error

	updateCalls int
	patchCalls  int
}

func (m *mockTaskService) updateCallCount() int {
	return m.updateCalls
}

func (m *mockTaskService) patchCallCount() int {
	return m.patchCalls
}

func (m *mockTaskService) CreateTask(
	ctx context.Context,
	title string,
	description string,
) (model.Task, error) {
	m.createTitle = title
	m.createDescription = description
	return m.createdTask, m.createError
}

func (m *mockTaskService) GetTask(
	ctx context.Context,
	id string,
) (model.Task, error) {
	m.getID = id

	return m.task, m.getError
}

func (m *mockTaskService) GetTasks(
	ctx context.Context,
) ([]model.Task, error) {
	return m.tasks, m.getTasksError
}

func (m *mockTaskService) UpdateTask(
	ctx context.Context,
	task model.Task,
) (model.Task, error) {
	m.updateCalls++
	m.updateInput = task
	m.updatedTask = task
	return m.updatedTask, m.updateError
}

func (m *mockTaskService) PatchTask(
	ctx context.Context,
	id string,
	update model.TaskUpdate,
) (model.Task, error) {
	m.patchCalls++
	m.patchedID = id
	m.patchedUpdate = update
	m.updatedTask = m.task
	return m.updatedTask, m.patchError
}

func (m *mockTaskService) DeleteTask(
	ctx context.Context,
	id string,
) error {
	m.deletedID = id
	return m.deleteError
}

func TestCreateTaskInvalidJSON(t *testing.T) {
	taskService := &mockTaskService{}
	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPost,
			},
		},
		Body: `{"title":`,
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}

	if response.Body != `{"message":"Invalid JSON"}` {
		t.Fatalf(
			"expected body %s, got %s",
			`{"message":"Invalid JSON"}`,
			response.Body,
		)
	}
}

func TestCreateTask(t *testing.T) {
	task := model.Task{
		ID:          "01TEST",
		Title:       "Comprar leche",
		Description: "Comprar leche entera",
		Completed:   false,
	}

	taskService := &mockTaskService{
		createdTask: task,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPost,
			},
		},
		Body: `{
			"title": "Comprar leche",
			"description": "Comprar leche entera"
		}`,
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			response.StatusCode,
		)
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %s",
			response.Headers["Content-Type"],
		)
	}

	if response.Headers["Location"] != "/tasks/"+task.ID {
		t.Fatalf(
			"expected Location /tasks/%s, got %s",
			task.ID,
			response.Headers["Location"],
		)
	}

	if taskService.createTitle != "Comprar leche" {
		t.Fatalf(
			"expected title %q, got %q",
			"Comprar leche",
			taskService.createTitle,
		)
	}

	if taskService.createDescription != "Comprar leche entera" {
		t.Fatalf(
			"expected description %q, got %q",
			"Comprar leche entera",
			taskService.createDescription,
		)
	}

	var responseTask model.Task

	err = json.Unmarshal(
		[]byte(response.Body),
		&responseTask,
	)
	if err != nil {
		t.Fatalf("expected valid JSON response, got %v", err)
	}

	if responseTask != task {
		t.Fatalf(
			"expected task %+v, got %+v",
			task,
			responseTask,
		)
	}
}

func TestCreateTaskInvalidTask(t *testing.T) {
	taskService := &mockTaskService{
		createError: service.ErrInvalidTask,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPost,
			},
		},
		Body: `{
			"title": "",
			"description": "Sin título"
		}`,
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}

	if response.Body != `{"message":"Invalid task"}` {
		t.Fatalf(
			"expected body %s, got %s",
			`{"message":"Invalid task"}`,
			response.Body,
		)
	}
}

func TestGetTask(t *testing.T) {
	task := model.Task{
		ID:          "01TEST",
		Title:       "Comprar leche",
		Description: "Comprar leche entera",
		Completed:   false,
	}

	taskService := &mockTaskService{
		task: task,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		PathParameters: map[string]string{
			"id": task.ID,
		},
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if taskService.getID != task.ID {
		t.Fatalf(
			"expected id %q, got %q",
			task.ID,
			taskService.getID,
		)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %s",
			response.Headers["Content-Type"],
		)
	}

	var responseTask model.Task

	err = json.Unmarshal(
		[]byte(response.Body),
		&responseTask,
	)
	if err != nil {
		t.Fatalf("expected valid JSON response, got %v", err)
	}

	if responseTask != task {
		t.Fatalf(
			"expected task %+v, got %+v",
			task,
			responseTask,
		)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	taskService := &mockTaskService{
		getError: repository.ErrTaskNotFound,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.StatusCode,
		)
	}

	if response.Body != `{"message":"Task not found"}` {
		t.Fatalf(
			"expected body %s, got %s",
			`{"message":"Task not found"}`,
			response.Body,
		)
	}
}

func TestGetTaskInternalServerError(t *testing.T) {
	taskService := &mockTaskService{
		getError: errors.New("database error"),
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.StatusCode,
		)
	}

	if response.Body != `{"message":"Internal Server Error"}` {
		t.Fatalf(
			"expected body %s, got %s",
			`{"message":"Internal Server Error"}`,
			response.Body,
		)
	}
}

func TestGetTasks(t *testing.T) {
	tasks := []model.Task{
		{
			ID:          "01TEST1",
			Title:       "Comprar leche",
			Description: "Comprar leche entera",
			Completed:   false,
		},
		{
			ID:          "01TEST2",
			Title:       "Pagar factura",
			Description: "Pagar factura de luz",
			Completed:   true,
		},
	}

	taskService := &mockTaskService{
		tasks: tasks,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %s",
			response.Headers["Content-Type"],
		)
	}

	var responseTasks []model.Task

	err = json.Unmarshal(
		[]byte(response.Body),
		&responseTasks,
	)
	if err != nil {
		t.Fatalf("expected valid JSON response, got %v", err)
	}

	if len(responseTasks) != len(tasks) {
		t.Fatalf(
			"expected %d tasks, got %d",
			len(tasks),
			len(responseTasks),
		)
	}

	for i := range tasks {
		if responseTasks[i] != tasks[i] {
			t.Fatalf(
				"expected task %+v, got %+v",
				tasks[i],
				responseTasks[i],
			)
		}
	}
}

func TestGetTasksInternalServerError(t *testing.T) {
	taskService := &mockTaskService{
		getTasksError: errors.New("database error"),
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
	}

	response, err := handler.Handle(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.StatusCode,
		)
	}

	if response.Body != `{"message":"Internal Server Error"}` {
		t.Fatalf(
			"expected body %s, got %s",
			`{"message":"Internal Server Error"}`,
			response.Body,
		)
	}
}

func TestUpdateTask(t *testing.T) {
	taskService := &mockTaskService{}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPut,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: `{
			"title": "Comprar leche",
			"description": "Comprar leche entera",
			"completed": true
		}`,
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if response.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q",
			response.Headers["Content-Type"])
	}

	var task model.Task

	err = json.Unmarshal([]byte(response.Body), &task)
	if err != nil {
		t.Fatalf("unexpected invalid JSON response: %v", err)
	}

	if task.ID != "01TEST" {
		t.Fatalf("expected id %q, got %q", "01TEST", task.ID)
	}

	if task.Title != "Comprar leche" {
		t.Fatalf("expected title %q, got %q", "Comprar leche", task.Title)
	}

	if task.Description != "Comprar leche entera" {
		t.Fatalf(
			"expected description %q, got %q",
			"Comprar leche entera",
			task.Description,
		)
	}

	if !task.Completed {
		t.Fatal("expected completed to be true")
	}

	if taskService.updatedTask != task {
		t.Fatalf("expected updated task %+v, got %+v",
			task,
			taskService.updatedTask)
	}
}

func TestUpdateTaskInvalidJSON(t *testing.T) {
	taskService := &mockTaskService{}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPut,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: `{"title":`,
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Invalid JSON"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestUpdateTaskInvalidTask(t *testing.T) {
	taskService := &mockTaskService{
		updateError: service.ErrInvalidTask,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPut,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: `{
			"title": "",
			"description": "Descripción",
			"completed": false
		}`,
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Invalid task"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestUpdateTaskNotFound(t *testing.T) {
	taskService := &mockTaskService{
		updateError: repository.ErrTaskNotFound,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPut,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: `{
			"title": "Comprar leche",
			"description": "Comprar leche entera",
			"completed": true
		}`,
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Task not found"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestUpdateTaskInternalServerError(t *testing.T) {
	taskService := &mockTaskService{
		updateError: errors.New("database error"),
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPut,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: `{
			"title": "Comprar leche",
			"description": "Comprar leche entera",
			"completed": true
		}`,
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Internal Server Error"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestDeleteTask(t *testing.T) {
	taskService := &mockTaskService{}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodDelete,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.StatusCode)
	}

	if response.Body != "" {
		t.Fatalf("expected empty body, got %q", response.Body)
	}

	if taskService.deletedID != "01TEST" {
		t.Fatalf(
			"expected deleted id %q, got %q",
			"01TEST",
			taskService.deletedID,
		)
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	taskService := &mockTaskService{
		deleteError: repository.ErrTaskNotFound,
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodDelete,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Task not found"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestDeleteTaskInternalServerError(t *testing.T) {
	taskService := &mockTaskService{
		deleteError: errors.New("database error"),
	}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodDelete,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Internal Server Error"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	taskService := &mockTaskService{}

	handler := NewHandler(taskService)

	request := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodOptions,
			},
		},
	}

	response, err := handler.Handle(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Method not allowed"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func updateRequest(method string, body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		PathParameters: map[string]string{
			"id": "01TEST",
		},
		Body: body,
	}
}

func TestPutMissingFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing title", `{"description":"d","completed":true}`},
		{"missing description", `{"title":"t","completed":true}`},
		{"missing completed", `{"title":"t","description":"d"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskService := &mockTaskService{}
			handler := NewHandler(taskService)

			response, err := handler.Handle(
				context.Background(),
				updateRequest(http.MethodPut, tc.body),
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400, got %d",
					response.StatusCode,
				)
			}

			if taskService.updateCallCount() != 0 {
				t.Fatalf(
					"expected UpdateTask not to be called, got %d calls",
					taskService.updateCallCount(),
				)
			}
		})
	}
}

func TestPutExplicitFalse(t *testing.T) {
	taskService := &mockTaskService{}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(
			http.MethodPut,
			`{"title":"t","description":"d","completed":false}`,
		),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.updateInput.Completed {
		t.Fatal("expected completed false to be applied")
	}
}

func TestUpdateInvalidBodies(t *testing.T) {
	cases := []struct {
		name   string
		method string
		body   string
	}{
		{"put empty body", http.MethodPut, ``},
		{"put whitespace body", http.MethodPut, `   `},
		{"patch empty body", http.MethodPatch, ``},
		{"patch whitespace body", http.MethodPatch, "\n\t "},
		{"put root null", http.MethodPut, `null`},
		{"patch root null", http.MethodPatch, `null`},
		{"put root null with spaces", http.MethodPut, `  null  `},
		{"put non-object array", http.MethodPut, `[]`},
		{"patch non-object array", http.MethodPatch, ` [1,2,3] `},
		{"patch non-object string", http.MethodPatch, `"title"`},
		{"put non-object number", http.MethodPut, `42`},
		{"patch non-object boolean", http.MethodPatch, `false`},
		{"put trailing values", http.MethodPut, `{"title":"t","description":"d","completed":true}{}`},
		{"patch trailing values", http.MethodPatch, `{"title":"t"} extra`},
		{"put trailing null", http.MethodPut, `{"title":"t","description":"d","completed":true} null`},
		{"put unknown field", http.MethodPut, `{"title":"t","description":"d","completed":true,"foo":1}`},
		{"patch unknown field", http.MethodPatch, `{"foo":1}`},
		{"put capitalized title", http.MethodPut, `{"Title":"t","description":"d","completed":true}`},
		{"patch capitalized description", http.MethodPatch, `{"Description":"d"}`},
		{"patch capitalized completed", http.MethodPatch, `{"Completed":true}`},
		{"put uppercase title", http.MethodPut, `{"TITLE":"t","description":"d","completed":true}`},
		{"put capitalized field with exact others", http.MethodPut, `{"title":"t","description":"d","Completed":true}`},
		{"patch mixed casing", http.MethodPatch, `{"title":"t","Description":"d","COMPLETED":true}`},
		{"put null title", http.MethodPut, `{"title":null,"description":"d","completed":true}`},
		{"patch null completed", http.MethodPatch, `{"completed":null}`},
		{"put invalid json", http.MethodPut, `{"title":`},
		{"patch invalid json", http.MethodPatch, `{`},
		{"patch empty object", http.MethodPatch, `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskService := &mockTaskService{}
			handler := NewHandler(taskService)

			response, err := handler.Handle(
				context.Background(),
				updateRequest(tc.method, tc.body),
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400, got %d (body %s)",
					response.StatusCode,
					response.Body,
				)
			}

			if taskService.updateCallCount() != 0 {
				t.Fatalf(
					"expected UpdateTask not to be called, got %d calls",
					taskService.updateCallCount(),
				)
			}

			if taskService.patchCallCount() != 0 {
				t.Fatalf(
					"expected PatchTask not to be called, got %d calls",
					taskService.patchCallCount(),
				)
			}
		})
	}
}

func TestPatchOnlyTitle(t *testing.T) {
	taskService := &mockTaskService{
		task: model.Task{
			ID:          "01TEST",
			Title:       "nuevo",
			Description: "vieja",
			Completed:   true,
		},
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"title":"nuevo"}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.patchedID != "01TEST" {
		t.Fatalf("expected id 01TEST, got %s", taskService.patchedID)
	}

	if taskService.patchedUpdate.Title == nil ||
		*taskService.patchedUpdate.Title != "nuevo" {
		t.Fatal("expected title to be forwarded")
	}

	if taskService.patchedUpdate.Description != nil ||
		taskService.patchedUpdate.Completed != nil {
		t.Fatal("expected omitted fields to remain nil")
	}

	var responseTask model.Task

	if err := json.Unmarshal([]byte(response.Body), &responseTask); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if responseTask != taskService.task {
		t.Fatalf(
			"expected persisted task %+v, got %+v",
			taskService.task,
			responseTask,
		)
	}
}

func TestPatchOnlyDescription(t *testing.T) {
	taskService := &mockTaskService{
		task: model.Task{
			ID:          "01TEST",
			Title:       "vieja",
			Description: "nueva",
			Completed:   true,
		},
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"description":"nueva"}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.patchedID != "01TEST" {
		t.Fatalf("expected id 01TEST, got %s", taskService.patchedID)
	}

	if taskService.patchedUpdate.Description == nil ||
		*taskService.patchedUpdate.Description != "nueva" {
		t.Fatal("expected description to be forwarded")
	}

	if taskService.patchedUpdate.Title != nil ||
		taskService.patchedUpdate.Completed != nil {
		t.Fatal("expected omitted fields to remain nil")
	}

	var responseTask model.Task

	if err := json.Unmarshal([]byte(response.Body), &responseTask); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if responseTask != taskService.task {
		t.Fatalf(
			"expected persisted task %+v, got %+v",
			taskService.task,
			responseTask,
		)
	}
}

func TestPatchOnlyCompletedTrue(t *testing.T) {
	taskService := &mockTaskService{
		task: model.Task{
			ID:          "01TEST",
			Title:       "t",
			Description: "d",
			Completed:   true,
		},
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"completed":true}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.patchedUpdate.Completed == nil ||
		*taskService.patchedUpdate.Completed != true {
		t.Fatal("expected completed true to be forwarded as present")
	}

	if taskService.patchedUpdate.Title != nil ||
		taskService.patchedUpdate.Description != nil {
		t.Fatal("expected omitted fields to remain nil")
	}

	var responseTask model.Task

	if err := json.Unmarshal([]byte(response.Body), &responseTask); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if responseTask != taskService.task {
		t.Fatalf(
			"expected persisted task %+v, got %+v",
			taskService.task,
			responseTask,
		)
	}
}

func TestPatchOnlyCompletedFalse(t *testing.T) {
	taskService := &mockTaskService{
		task: model.Task{
			ID:          "01TEST",
			Title:       "t",
			Description: "d",
			Completed:   false,
		},
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"completed":false}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.patchedUpdate.Completed == nil ||
		*taskService.patchedUpdate.Completed != false {
		t.Fatal("expected completed false to be forwarded as present")
	}

	if taskService.patchedUpdate.Title != nil ||
		taskService.patchedUpdate.Description != nil {
		t.Fatal("expected omitted fields to remain nil")
	}
}

func TestPatchMultipleFields(t *testing.T) {
	taskService := &mockTaskService{
		task: model.Task{
			ID:          "01TEST",
			Title:       "t",
			Description: "nueva",
			Completed:   true,
		},
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(
			http.MethodPatch,
			`{"description":"nueva","completed":true}`,
		),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if taskService.patchedUpdate.Description == nil ||
		*taskService.patchedUpdate.Description != "nueva" {
		t.Fatal("expected description to be forwarded")
	}

	if taskService.patchedUpdate.Completed == nil ||
		*taskService.patchedUpdate.Completed != true {
		t.Fatal("expected completed true to be forwarded")
	}

	if taskService.patchedUpdate.Title != nil {
		t.Fatal("expected omitted title to remain nil")
	}
}

func TestPatchInvalidTitle(t *testing.T) {
	taskService := &mockTaskService{
		patchError: service.ErrInvalidTask,
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"title":"   "}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Invalid task"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestPatchNotFound(t *testing.T) {
	taskService := &mockTaskService{
		patchError: repository.ErrTaskNotFound,
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"completed":true}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Task not found"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}

func TestPatchInternalServerError(t *testing.T) {
	taskService := &mockTaskService{
		patchError: errors.New("database error"),
	}
	handler := NewHandler(taskService)

	response, err := handler.Handle(
		context.Background(),
		updateRequest(http.MethodPatch, `{"completed":true}`),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}

	if response.Body != `{"message":"Internal Server Error"}` {
		t.Fatalf("unexpected body: %s", response.Body)
	}
}
