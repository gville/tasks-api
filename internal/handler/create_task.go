package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"tasks-api/internal/service"
)

func (h *Handler) createTask(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	var input CreateTaskRequest

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

	task, err := h.taskService.CreateTask(
		ctx,
		input.Title,
		input.Description,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTask) {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusBadRequest,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"message":"Invalid task"}`,
			}, nil
		}

		log.Printf("ERROR operation=CreateTask requestId=%s error=%v",
			request.RequestContext.RequestID,
			err,
		)

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"message":"Internal Server Error"}`,
		}, nil
	}

	body, err := json.Marshal(task)
	if err != nil {
		log.Printf(
			"ERROR operation=CreateTask requestId=%s error=failed_to_marshal_response details=%v",
			request.RequestContext.RequestID,
			err,
		)

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"message":"Internal Server Error"}`,
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Location":     "/tasks/" + task.ID,
		},
		Body: string(body),
	}, nil
}
