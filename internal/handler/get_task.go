package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"tasks-api/internal/repository"
)

func (h *Handler) getTask(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	id := request.PathParameters["id"]

	task, err := h.taskService.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusNotFound,
				Body:       `{"message":"Task not found"}`,
			}, nil
		}

		log.Printf(
			"ERROR operation=GetTask requestId=%s error=%v",
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
