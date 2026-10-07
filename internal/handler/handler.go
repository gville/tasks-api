package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"tasks-api/internal/model"
	"tasks-api/internal/repository"
	"tasks-api/internal/service"

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
		id := request.PathParameters["id"]

		var input UpdateTaskRequest

		err := json.Unmarshal([]byte(request.Body), &input)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusBadRequest,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"message":"Invalid JSON"}`,
			}, nil
		}

		task := model.Task{
			ID:          id,
			Title:       input.Title,
			Description: input.Description,
			Completed:   input.Completed,
		}

		err = h.taskService.UpdateTask(
			ctx,
			task,
		)
		if err != nil {
			if errors.Is(err, repository.ErrTaskNotFound) {
				return events.APIGatewayV2HTTPResponse{
					StatusCode: http.StatusNotFound,
					Body:       `{"message":"Task not found"}`,
				}, nil
			}

			if errors.Is(err, service.ErrInvalidTask) {
				return events.APIGatewayV2HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{"message":"Invalid task"}`,
				}, nil
			}

			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       `{"message":"Internal Server Error"}`,
			}, nil
		}

		body, err := json.Marshal(task)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       `{"message":"Internal Server Error"}`,
			}, nil
		}

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(body),
		}, nil
	case http.MethodDelete:
		id := request.PathParameters["id"]

		err := h.taskService.DeleteTask(
			ctx,
			id,
		)
		if err != nil {
			if errors.Is(err, repository.ErrTaskNotFound) {
				return events.APIGatewayV2HTTPResponse{
					StatusCode: http.StatusNotFound,
					Body:       `{"message":"Task not found"}`,
				}, nil
			}

			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       `{"message":"Internal Server Error"}`,
			}, nil
		}

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNoContent,
		}, nil
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
