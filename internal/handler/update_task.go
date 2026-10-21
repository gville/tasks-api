package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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

	input, err := parseUpdateBody(request.Body)
	if err != nil {
		return badRequest("Invalid JSON"), nil
	}

	if input.Title == nil || input.Description == nil || input.Completed == nil {
		return badRequest("Invalid task"), nil
	}

	task := model.Task{
		ID:          id,
		Title:       *input.Title,
		Description: *input.Description,
		Completed:   *input.Completed,
	}

	updated, err := h.taskService.UpdateTask(ctx, task)
	if err != nil {
		return h.updateErrorResponse(
			request,
			"UpdateTask",
			id,
			err,
		), nil
	}

	return h.taskResponse(request, "UpdateTask", id, updated), nil
}

func (h *Handler) updateErrorResponse(
	request events.APIGatewayV2HTTPRequest,
	operation string,
	id string,
	err error,
) events.APIGatewayV2HTTPResponse {

	if errors.Is(err, repository.ErrTaskNotFound) {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNotFound,
			Body:       `{"message":"Task not found"}`,
		}
	}

	if errors.Is(err, service.ErrInvalidTask) {
		return badRequest("Invalid task")
	}

	log.Printf(
		"ERROR operation=%s requestId=%s taskId=%s error=%v",
		operation,
		request.RequestContext.RequestID,
		id,
		err,
	)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"message":"Internal Server Error"}`,
	}
}

func (h *Handler) taskResponse(
	request events.APIGatewayV2HTTPRequest,
	operation string,
	id string,
	task model.Task,
) events.APIGatewayV2HTTPResponse {

	body, err := json.Marshal(task)
	if err != nil {
		log.Printf(
			"ERROR operation=%s requestId=%s taskId=%s error=failed_to_marshal_response details=%v",
			operation,
			request.RequestContext.RequestID,
			id,
			err,
		)

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"message":"Internal Server Error"}`,
		}
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}
}

func badRequest(message string) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusBadRequest,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"message":"` + message + `"}`,
	}
}
