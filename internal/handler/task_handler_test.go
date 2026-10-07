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
	deletedID         string
	createTitle       string
	createDescription string
	getID             string

	createError   error
	getError      error
	getTasksError error
	updateError   error
	deleteError   error
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
) error {
	m.updatedTask = task
	return m.updateError
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
				Method: http.MethodPatch,
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
