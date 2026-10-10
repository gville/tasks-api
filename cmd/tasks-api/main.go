package main

import (
	"context"
	"log"
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
		log.Fatalf("ERROR operation=LoadAWSConfig error=%v", err)
	}

	tableName := os.Getenv("TASKS_TABLE_NAME")
	if tableName == "" {
		log.Fatal("ERROR operation=Initialize error=TASKS_TABLE_NAME_not_configured")
	}

	client := dynamodb.NewFromConfig(cfg)

	taskRepository := repository.NewTaskRepository(
		client,
		tableName,
	)
	taskService := service.NewTaskService(taskRepository)

	taskHandler := handler.NewHandler(taskService)

	lambda.Start(taskHandler.Handle)
}
