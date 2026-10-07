package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"tasks-api/internal/repository"
)

func (h *Handler) deleteTask(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	id := request.PathParameters["id"]

	err := h.taskService.DeleteTask(ctx, id)
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
