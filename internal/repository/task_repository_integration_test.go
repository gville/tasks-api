//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"tasks-api/internal/model"
)

func newIntegrationRepository(t *testing.T) *TaskRepository {
	t.Helper()

	tableName := os.Getenv("TASKS_TABLE_NAME")
	if tableName == "" {
		t.Fatal("TASKS_TABLE_NAME is required")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	return NewTaskRepository(client, tableName)
}

func TestIntegrationCreateAndGetTask(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := model.Task{
		ID:          "integration-test-001",
		Title:       "Integration test",
		Description: "Created in DynamoDB",
		Completed:   false,
	}

	t.Cleanup(func() {
		_ = repository.DeleteTask(context.Background(), task.ID)
	})

	err := repository.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	got, err := repository.GetTask(
		context.Background(),
		task.ID,
	)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != task {
		t.Fatalf("GetTask returned %+v, want %+v", got, task)
	}
}

func TestIntegrationGetTaskNotFound(t *testing.T) {
	repository := newIntegrationRepository(t)

	_, err := repository.GetTask(
		context.Background(),
		"integration-test-not-found",
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask returned error %v, want %v", err, ErrTaskNotFound)
	}
}

func TestIntegrationUpdateTask(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := model.Task{
		ID:          "integration-test-update",
		Title:       "Original title",
		Description: "Original description",
		Completed:   false,
	}

	t.Cleanup(func() {
		_ = repository.DeleteTask(context.Background(), task.ID)
	})

	err := repository.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	task.Title = "Updated title"
	task.Description = "Updated description"
	task.Completed = true

	err = repository.UpdateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("UpdateTask returned error: %v", err)
	}

	got, err := repository.GetTask(
		context.Background(),
		task.ID,
	)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != task {
		t.Fatalf("GetTask returned %+v, want %+v", got, task)
	}
}

func TestIntegrationUpdateTaskNotFound(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := model.Task{
		ID:          "integration-test-update-not-found",
		Title:       "Updated title",
		Description: "Updated description",
		Completed:   true,
	}

	err := repository.UpdateTask(context.Background(), task)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf(
			"UpdateTask returned error %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}
}

func TestIntegrationDeleteTask(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := model.Task{
		ID:          "integration-test-delete",
		Title:       "Task to delete",
		Description: "Will be deleted",
		Completed:   false,
	}

	err := repository.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	err = repository.DeleteTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}

	_, err = repository.GetTask(
		context.Background(),
		task.ID,
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf(
			"GetTask after DeleteTask returned error %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}
}

func TestIntegrationGetTasks(t *testing.T) {
	repository := newIntegrationRepository(t)

	tasks := []model.Task{
		{
			ID:          "integration-test-list-001",
			Title:       "First task",
			Description: "First task description",
			Completed:   false,
		},
		{
			ID:          "integration-test-list-002",
			Title:       "Second task",
			Description: "Second task description",
			Completed:   true,
		},
	}

	t.Cleanup(func() {
		for _, task := range tasks {
			_ = repository.DeleteTask(context.Background(), task.ID)
		}
	})

	for _, task := range tasks {
		err := repository.CreateTask(context.Background(), task)
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}
	}

	got, err := repository.GetTasks(context.Background())
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}

	found := make(map[string]model.Task)

	for _, task := range got {
		found[task.ID] = task
	}

	for _, want := range tasks {
		got, ok := found[want.ID]

		if !ok {
			t.Fatalf("GetTasks did not return task %q", want.ID)
		}

		if got != want {
			t.Fatalf("GetTasks returned %+v, want %+v", got, want)
		}
	}
}
