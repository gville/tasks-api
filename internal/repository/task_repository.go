package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"tasks-api/internal/model"
)

type TaskRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewTaskRepository(
	client *dynamodb.Client,
	tableName string,
) *TaskRepository {
	return &TaskRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *TaskRepository) CreateTask(
	ctx context.Context,
	task model.Task,
) error {
	item, err := attributevalue.MarshalMap(task)
	if err != nil {
		return err
	}

	_, err = r.client.PutItem(
		ctx,
		&dynamodb.PutItemInput{
			TableName: &r.tableName,
			Item:      item,
		},
	)

	return err
}
