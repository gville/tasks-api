package handler

import (
	"context"
	"net/http"
	"tasks-api/internal/model"

	"github.com/aws/aws-lambda-go/events"
)

type Handler struct {
	taskService TaskService
}

func NewHandler(taskService TaskService) *Handler {
	return &Handler{
		taskService: taskService,
	}
}

func (h *Handler) Handle(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	switch request.RequestContext.HTTP.Method {
	case http.MethodGet:
		if request.PathParameters["id"] != "" {
			return h.getTask(ctx, request)
		}

		return h.getTasks(ctx, request)

	case http.MethodPost:
		return h.createTask(ctx, request)

	case http.MethodPut:
		return h.updateTask(ctx, request)

	case http.MethodDelete:
		return h.deleteTask(ctx, request)
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Body:       `{"message":"Method not allowed"}`,
	}, nil
}

type TaskService interface {
	CreateTask(
		ctx context.Context,
		title string,
		description string,
	) (model.Task, error)

	GetTask(
		ctx context.Context,
		id string,
	) (model.Task, error)

	GetTasks(
		ctx context.Context,
	) ([]model.Task, error)

	UpdateTask(
		ctx context.Context,
		task model.Task,
	) error

	DeleteTask(
		ctx context.Context,
		id string,
	) error
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}
