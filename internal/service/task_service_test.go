package service

import (
	"context"
	"errors"
	"testing"

	"tasks-api/internal/model"
)

type mockTaskRepository struct {
	createdTask   model.Task
	updatedTask   model.Task
	createError   error
	updateError   error
	task          model.Task
	getError      error
	tasks         []model.Task
	getTasksError error
	deleteError   error
	deletedID     string
}

func (m *mockTaskRepository) CreateTask(
	ctx context.Context,
	task model.Task,
) error {
	m.createdTask = task
	return m.createError
}

func (m *mockTaskRepository) GetTask(
	ctx context.Context,
	id string,
) (model.Task, error) {
	return m.task, m.getError
}

func (m *mockTaskRepository) GetTasks(
	ctx context.Context,
) ([]model.Task, error) {
	return m.tasks, m.getTasksError
}

func (m *mockTaskRepository) UpdateTask(
	ctx context.Context,
	task model.Task,
) error {
	m.updatedTask = task
	return m.updateError
}

func (m *mockTaskRepository) DeleteTask(
	ctx context.Context,
	id string,
) error {
	m.deletedID = id
	return m.deleteError
}

func TestCreateTaskEmptyTitle(t *testing.T) {
	repository := &mockTaskRepository{}
	service := NewTaskService(repository)

	_, err := service.CreateTask(
		context.Background(),
		"",
		"Description",
	)

	if err != ErrInvalidTask {
		t.Fatalf(
			"expected ErrInvalidTask, got %v",
			err,
		)
	}
}

func TestCreateTask(t *testing.T) {
	repository := &mockTaskRepository{}
	service := NewTaskService(repository)

	task, err := service.CreateTask(
		context.Background(),
		"Comprar leche",
		"Ir al supermercado",
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if task.ID == "" {
		t.Fatal("expected task ID to be generated")
	}

	if task.Title != "Comprar leche" {
		t.Fatalf(
			"expected title %q, got %q",
			"Comprar leche",
			task.Title,
		)
	}

	if task.Description != "Ir al supermercado" {
		t.Fatalf(
			"expected description %q, got %q",
			"Ir al supermercado",
			task.Description,
		)
	}

	if task.Completed {
		t.Fatal("expected task to be incomplete")
	}

	if repository.createdTask != task {
		t.Fatalf(
			"expected repository to receive task %+v, got %+v",
			task,
			repository.createdTask,
		)
	}
}

func TestCreateTaskRepositoryError(t *testing.T) {
	repositoryError := errors.New("database error")

	repository := &mockTaskRepository{
		createError: repositoryError,
	}

	service := NewTaskService(repository)

	_, err := service.CreateTask(
		context.Background(),
		"Comprar leche",
		"Ir al supermercado",
	)

	if err != repositoryError {
		t.Fatalf(
			"expected error %v, got %v",
			repositoryError,
			err,
		)
	}
}

func TestUpdateTaskEmptyTitle(t *testing.T) {
	repository := &mockTaskRepository{}
	service := NewTaskService(repository)

	err := service.UpdateTask(
		context.Background(),
		model.Task{
			ID:          "123",
			Title:       "",
			Description: "Description",
			Completed:   false,
		},
	)

	if err != ErrInvalidTask {
		t.Fatalf(
			"expected ErrInvalidTask, got %v",
			err,
		)
	}
}

func TestUpdateTask(t *testing.T) {
	repository := &mockTaskRepository{}
	service := NewTaskService(repository)

	task := model.Task{
		ID:          "123",
		Title:       "Comprar leche",
		Description: "Ir al supermercado",
		Completed:   true,
	}

	err := service.UpdateTask(
		context.Background(),
		task,
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if repository.updatedTask != task {
		t.Fatalf(
			"expected repository to receive task %+v, got %+v",
			task,
			repository.createdTask,
		)
	}
}

func TestUpdateTaskRepositoryError(t *testing.T) {
	repositoryError := errors.New("database error")

	repository := &mockTaskRepository{
		updateError: repositoryError,
	}

	service := NewTaskService(repository)

	err := service.UpdateTask(
		context.Background(),
		model.Task{
			ID:          "123",
			Title:       "Comprar leche",
			Description: "Ir al supermercado",
			Completed:   true,
		},
	)

	if err != repositoryError {
		t.Fatalf(
			"expected error %v, got %v",
			repositoryError,
			err,
		)
	}
}

func TestGetTask(t *testing.T) {
	task := model.Task{
		ID:          "123",
		Title:       "Comprar leche",
		Description: "Ir al supermercado",
		Completed:   true,
	}

	repository := &mockTaskRepository{
		task: task,
	}

	service := NewTaskService(repository)

	result, err := service.GetTask(
		context.Background(),
		"123",
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result != task {
		t.Fatalf(
			"expected task %+v, got %+v",
			task,
			result,
		)
	}
}
func TestGetTaskRepositoryError(t *testing.T) {
	repositoryError := errors.New("database error")

	repository := &mockTaskRepository{
		getError: repositoryError,
	}

	service := NewTaskService(repository)

	_, err := service.GetTask(
		context.Background(),
		"123",
	)

	if err != repositoryError {
		t.Fatalf(
			"expected error %v, got %v",
			repositoryError,
			err,
		)
	}
}

func TestGetTasks(t *testing.T) {
	tasks := []model.Task{
		{
			ID:          "123",
			Title:       "Comprar leche",
			Description: "Ir al supermercado",
			Completed:   false,
		},
		{
			ID:          "456",
			Title:       "Estudiar Go",
			Description: "Practicar interfaces",
			Completed:   true,
		},
	}

	repository := &mockTaskRepository{
		tasks: tasks,
	}

	service := NewTaskService(repository)

	result, err := service.GetTasks(
		context.Background(),
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(result) != len(tasks) {
		t.Fatalf(
			"expected %d tasks, got %d",
			len(tasks),
			len(result),
		)
	}

	for i := range tasks {
		if result[i] != tasks[i] {
			t.Fatalf(
				"expected task %+v, got %+v",
				tasks[i],
				result[i],
			)
		}
	}
}

func TestGetTasksRepositoryError(t *testing.T) {
	repositoryError := errors.New("database error")

	repository := &mockTaskRepository{
		getTasksError: repositoryError,
	}

	service := NewTaskService(repository)

	_, err := service.GetTasks(
		context.Background(),
	)

	if err != repositoryError {
		t.Fatalf(
			"expected error %v, got %v",
			repositoryError,
			err,
		)
	}
}

func TestDeleteTask(t *testing.T) {
	repository := &mockTaskRepository{}
	service := NewTaskService(repository)

	err := service.DeleteTask(
		context.Background(),
		"123",
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if repository.deletedID != "123" {
		t.Fatalf(
			"expected repository to receive ID %q, got %q",
			"123",
			repository.deletedID,
		)
	}
}

func TestDeleteTaskRepositoryError(t *testing.T) {
	repositoryError := errors.New("database error")

	repository := &mockTaskRepository{
		deleteError: repositoryError,
	}

	service := NewTaskService(repository)

	err := service.DeleteTask(
		context.Background(),
		"123",
	)

	if err != repositoryError {
		t.Fatalf(
			"expected error %v, got %v",
			repositoryError,
			err,
		)
	}
}
