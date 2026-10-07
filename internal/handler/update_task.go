package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"tasks-api/internal/model"
	"tasks-api/internal/repository"
	"tasks-api/internal/service"
)

func (h *Handler) updateTask(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

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

	err = h.taskService.UpdateTask(ctx, task)
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
}
