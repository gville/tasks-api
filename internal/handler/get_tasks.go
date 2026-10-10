package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func (h *Handler) getTasks(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	tasks, err := h.taskService.GetTasks(ctx)
	if err != nil {
		log.Printf(
			"ERROR operation=GetTasks requestId=%s error=%v",
			request.RequestContext.RequestID,
			err,
		)

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"message":"Internal Server Error"}`,
		}, nil
	}

	body, err := json.Marshal(tasks)
	if err != nil {
		log.Printf(
			"ERROR operation=GetTasks requestId=%s error=failed_to_marshal_response details=%v",
			request.RequestContext.RequestID,
			err,
		)

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
