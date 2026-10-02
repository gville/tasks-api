package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"tasks-api/internal/model"
	"tasks-api/internal/repository"
	"tasks-api/internal/service"
)

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func makeHandler(
	taskService *service.TaskService,
) func(
	context.Context,
	events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	return func(
		ctx context.Context,
		request events.APIGatewayV2HTTPRequest,
	) (events.APIGatewayV2HTTPResponse, error) {

		switch request.RequestContext.HTTP.Method {
		case http.MethodGet:
			id := request.PathParameters["id"]

			if id != "" {
				task, err := taskService.GetTask(
					ctx,
					id,
				)
				if err != nil {
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

			tasks, err := taskService.GetTasks(ctx)
			if err != nil {
				return events.APIGatewayV2HTTPResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       `{"message":"Internal Server Error"}`,
				}, nil
			}

			body, err := json.Marshal(tasks)
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

		case http.MethodPost:
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

			task, err := taskService.CreateTask(
				ctx,
				input.Title,
				input.Description,
			)
			if err != nil {
				return events.APIGatewayV2HTTPResponse{}, err
			}

			body, err := json.Marshal(task)
			if err != nil {
				return events.APIGatewayV2HTTPResponse{}, err
			}

			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: string(body),
			}, nil

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

			err = taskService.UpdateTask(
				ctx,
				task,
			)
			if err != nil {
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

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       `{"message":"Method not allowed"}`,
		}, nil
	}
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	client := dynamodb.NewFromConfig(cfg)

	tableName := os.Getenv("TASKS_TABLE_NAME")

	taskRepository := repository.NewTaskRepository(
		client,
		tableName,
	)
	taskService := service.NewTaskService(taskRepository)

	handler := makeHandler(taskService)

	lambda.Start(handler)
}
