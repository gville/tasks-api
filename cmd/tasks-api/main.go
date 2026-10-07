package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"tasks-api/internal/handler"
	"tasks-api/internal/repository"
	"tasks-api/internal/service"
)

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

	taskHandler := handler.NewHandler(taskService)

	lambda.Start(taskHandler)
}
