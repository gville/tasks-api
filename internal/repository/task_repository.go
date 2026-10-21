package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"tasks-api/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

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

	if len(result.Item) == 0 {
		return model.Task{}, ErrTaskNotFound
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
) (model.Task, error) {
	result, err := r.client.UpdateItem(
		ctx,
		&dynamodb.UpdateItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: task.ID,
				},
			},
			ConditionExpression: aws.String(
				"attribute_exists(id)",
			),
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
			ReturnValues: types.ReturnValueAllNew,
		},
	)

	if err != nil {
		var conditionalCheckFailed *types.ConditionalCheckFailedException

		if errors.As(err, &conditionalCheckFailed) {
			return model.Task{}, ErrTaskNotFound
		}

		return model.Task{}, err
	}

	var updated model.Task

	err = attributevalue.UnmarshalMap(
		result.Attributes,
		&updated,
	)
	if err != nil {
		return model.Task{}, err
	}

	return updated, nil
}

func (r *TaskRepository) PatchTask(
	ctx context.Context,
	id string,
	update model.TaskUpdate,
) (model.Task, error) {
	expression := "SET "
	names := map[string]string{}
	values := map[string]types.AttributeValue{}

	assignments := []string{}

	if update.Title != nil {
		assignments = append(
			assignments,
			"#title = :title",
		)
		names["#title"] = "title"
		values[":title"] = &types.AttributeValueMemberS{
			Value: *update.Title,
		}
	}

	if update.Description != nil {
		assignments = append(
			assignments,
			"#description = :description",
		)
		names["#description"] = "description"
		values[":description"] = &types.AttributeValueMemberS{
			Value: *update.Description,
		}
	}

	if update.Completed != nil {
		assignments = append(
			assignments,
			"#completed = :completed",
		)
		names["#completed"] = "completed"
		values[":completed"] = &types.AttributeValueMemberBOOL{
			Value: *update.Completed,
		}
	}

	expression += strings.Join(assignments, ", ")

	result, err := r.client.UpdateItem(
		ctx,
		&dynamodb.UpdateItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: id,
				},
			},
			ConditionExpression: aws.String(
				"attribute_exists(id)",
			),
			UpdateExpression:          aws.String(expression),
			ExpressionAttributeNames:  names,
			ExpressionAttributeValues: values,
			ReturnValues:              types.ReturnValueAllNew,
		},
	)

	if err != nil {
		var conditionalCheckFailed *types.ConditionalCheckFailedException

		if errors.As(err, &conditionalCheckFailed) {
			return model.Task{}, ErrTaskNotFound
		}

		return model.Task{}, err
	}

	var updated model.Task

	err = attributevalue.UnmarshalMap(
		result.Attributes,
		&updated,
	)
	if err != nil {
		return model.Task{}, err
	}

	return updated, nil
}

func (r *TaskRepository) DeleteTask(
	ctx context.Context,
	id string,
) error {
	_, err := r.client.DeleteItem(
		ctx,
		&dynamodb.DeleteItemInput{
			TableName: &r.tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: id,
				},
			},
			ConditionExpression: aws.String(
				"attribute_exists(id)",
			),
		},
	)

	if err != nil {
		var conditionalCheckFailed *types.ConditionalCheckFailedException

		if errors.As(err, &conditionalCheckFailed) {
			return ErrTaskNotFound
		}

		return err
	}

	return nil
}
