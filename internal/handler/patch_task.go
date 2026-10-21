package handler

import (
	"context"

	"github.com/aws/aws-lambda-go/events"

	"tasks-api/internal/model"
)

func (h *Handler) patchTask(
	ctx context.Context,
	request events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {

	id := request.PathParameters["id"]

	input, err := parseUpdateBody(request.Body)
	if err != nil {
		return badRequest("Invalid JSON"), nil
	}

	if input.Title == nil && input.Description == nil && input.Completed == nil {
		return badRequest("Invalid task"), nil
	}

	update := model.TaskUpdate{
		Title:       input.Title,
		Description: input.Description,
		Completed:   input.Completed,
	}

	updated, err := h.taskService.PatchTask(ctx, id, update)
	if err != nil {
		return h.updateErrorResponse(
			request,
			"PatchTask",
			id,
			err,
		), nil
	}

	return h.taskResponse(request, "PatchTask", id, updated), nil
}
