//go:build integration

package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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

func uniqueID(t *testing.T, prefix string) string {
	t.Helper()

	random := make([]byte, 8)

	if _, err := rand.Read(random); err != nil {
		t.Fatalf("failed to generate a unique id: %v", err)
	}

	return fmt.Sprintf(
		"integration-test-%s-%s",
		prefix,
		hex.EncodeToString(random),
	)
}

func createIntegrationTask(
	t *testing.T,
	repository *TaskRepository,
	task model.Task,
) model.Task {
	t.Helper()

	task.ID = uniqueID(t, task.ID)

	if err := repository.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	// Register cleanup only after the record was created, so a failed create
	// never deletes a record this test did not create.
	t.Cleanup(func() {
		_ = repository.DeleteTask(context.Background(), task.ID)
	})

	return task
}

func TestIntegrationCreateAndGetTask(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Integration test",
		Description: "Created in DynamoDB",
		Completed:   false,
	})

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
		uniqueID(t, "get-not-found"),
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask returned error %v, want %v", err, ErrTaskNotFound)
	}
}

func TestIntegrationUpdateTask(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   false,
	})

	task.Title = "Updated title"
	task.Description = "Updated description"
	task.Completed = true

	updated, err := repository.UpdateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("UpdateTask returned error: %v", err)
	}

	if updated != task {
		t.Fatalf("UpdateTask returned %+v, want %+v", updated, task)
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

func TestIntegrationUpdateTaskExplicitFalse(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   true,
	})

	task.Title = "Updated title"
	task.Description = "Updated description"
	task.Completed = false

	updated, err := repository.UpdateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("UpdateTask returned error: %v", err)
	}

	if updated != task {
		t.Fatalf("UpdateTask returned %+v, want %+v", updated, task)
	}

	if updated.Completed {
		t.Fatal("expected persisted task to have completed false")
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != task {
		t.Fatalf("GetTask returned %+v, want %+v", got, task)
	}
}

func TestIntegrationPatchTaskOnlyTitle(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   true,
	})

	title := "Patched title"

	updated, err := repository.PatchTask(
		context.Background(),
		task.ID,
		model.TaskUpdate{Title: &title},
	)
	if err != nil {
		t.Fatalf("PatchTask returned error: %v", err)
	}

	want := model.Task{
		ID:          task.ID,
		Title:       "Patched title",
		Description: "Original description",
		Completed:   true,
	}

	if updated != want {
		t.Fatalf("PatchTask returned %+v, want %+v", updated, want)
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != want {
		t.Fatalf("GetTask returned %+v, want %+v", got, want)
	}
}

func TestIntegrationPatchTaskOnlyDescription(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   true,
	})

	description := "Patched description"

	updated, err := repository.PatchTask(
		context.Background(),
		task.ID,
		model.TaskUpdate{Description: &description},
	)
	if err != nil {
		t.Fatalf("PatchTask returned error: %v", err)
	}

	want := model.Task{
		ID:          task.ID,
		Title:       "Original title",
		Description: "Patched description",
		Completed:   true,
	}

	if updated != want {
		t.Fatalf("PatchTask returned %+v, want %+v", updated, want)
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != want {
		t.Fatalf("GetTask returned %+v, want %+v", got, want)
	}
}

func TestIntegrationPatchTaskOnlyCompletedTrue(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   false,
	})

	completed := true

	updated, err := repository.PatchTask(
		context.Background(),
		task.ID,
		model.TaskUpdate{Completed: &completed},
	)
	if err != nil {
		t.Fatalf("PatchTask returned error: %v", err)
	}

	want := model.Task{
		ID:          task.ID,
		Title:       "Original title",
		Description: "Original description",
		Completed:   true,
	}

	if updated != want {
		t.Fatalf("PatchTask returned %+v, want %+v", updated, want)
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != want {
		t.Fatalf("GetTask returned %+v, want %+v", got, want)
	}
}

func TestIntegrationPatchTaskOnlyCompletedFalse(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   true,
	})

	completed := false

	updated, err := repository.PatchTask(
		context.Background(),
		task.ID,
		model.TaskUpdate{Completed: &completed},
	)
	if err != nil {
		t.Fatalf("PatchTask returned error: %v", err)
	}

	want := model.Task{
		ID:          task.ID,
		Title:       "Original title",
		Description: "Original description",
		Completed:   false,
	}

	if updated != want {
		t.Fatalf("PatchTask returned %+v, want %+v", updated, want)
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != want {
		t.Fatalf("GetTask returned %+v, want %+v", got, want)
	}
}

func TestIntegrationPatchTaskMultipleFields(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Original title",
		Description: "Original description",
		Completed:   false,
	})

	title := "Patched title"
	completed := true

	updated, err := repository.PatchTask(
		context.Background(),
		task.ID,
		model.TaskUpdate{
			Title:     &title,
			Completed: &completed,
		},
	)
	if err != nil {
		t.Fatalf("PatchTask returned error: %v", err)
	}

	want := model.Task{
		ID:          task.ID,
		Title:       "Patched title",
		Description: "Original description",
		Completed:   true,
	}

	if updated != want {
		t.Fatalf("PatchTask returned %+v, want %+v", updated, want)
	}

	got, err := repository.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}

	if got != want {
		t.Fatalf("GetTask returned %+v, want %+v", got, want)
	}
}

func TestIntegrationPatchTaskNotFound(t *testing.T) {
	repository := newIntegrationRepository(t)

	completed := true

	_, err := repository.PatchTask(
		context.Background(),
		uniqueID(t, "patch-not-found"),
		model.TaskUpdate{Completed: &completed},
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf(
			"PatchTask returned error %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}
}

func TestIntegrationUpdateTaskNotFound(t *testing.T) {
	repository := newIntegrationRepository(t)

	task := model.Task{
		ID:          uniqueID(t, "update-not-found"),
		Title:       "Updated title",
		Description: "Updated description",
		Completed:   true,
	}

	_, err := repository.UpdateTask(context.Background(), task)

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

	task := createIntegrationTask(t, repository, model.Task{
		Title:       "Task to delete",
		Description: "Will be deleted",
		Completed:   false,
	})

	if err := repository.DeleteTask(context.Background(), task.ID); err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}

	_, err := repository.GetTask(
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
		createIntegrationTask(t, repository, model.Task{
			Title:       "First task",
			Description: "First task description",
			Completed:   false,
		}),
		createIntegrationTask(t, repository, model.Task{
			Title:       "Second task",
			Description: "Second task description",
			Completed:   true,
		}),
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
