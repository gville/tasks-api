package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

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

func (r *TaskRepository) GetTask(
	ctx context.Context,
	id string,
) (model.Task, error) {
	result, err := r.client.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: id,
				},
			},
		},
	)
	if err != nil {
		return model.Task{}, err
	}

	var task model.Task

	err = attributevalue.UnmarshalMap(
		result.Item,
		&task,
	)
	if err != nil {
		return model.Task{}, err
	}

	return task, nil
}

func (r *TaskRepository) GetTasks(
	ctx context.Context,
) ([]model.Task, error) {
	result, err := r.client.Scan(
		ctx,
		&dynamodb.ScanInput{
			TableName: &r.tableName,
		},
	)
	if err != nil {
		return nil, err
	}

	var tasks []model.Task

	err = attributevalue.UnmarshalListOfMaps(
		result.Items,
		&tasks,
	)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) UpdateTask(
	ctx context.Context,
	task model.Task,
) error {
	_, err := r.client.UpdateItem(
		ctx,
		&dynamodb.UpdateItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: task.ID,
				},
			},
			UpdateExpression: aws.String(
				"SET #title = :title, #description = :description, #completed = :completed",
			),
			ExpressionAttributeNames: map[string]string{
				"#title":       "title",
				"#description": "description",
				"#completed":   "completed",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":title": &types.AttributeValueMemberS{
					Value: task.Title,
				},
				":description": &types.AttributeValueMemberS{
					Value: task.Description,
				},
				":completed": &types.AttributeValueMemberBOOL{
					Value: task.Completed,
				},
			},
		},
	)

	return err
}
