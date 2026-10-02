package service

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"tasks-api/internal/model"
	"tasks-api/internal/repository"

	"github.com/oklog/ulid/v2"
)

var ErrInvalidTask = errors.New("invalid task")

type TaskService struct {
	repository *repository.TaskRepository
}

func NewTaskService(repository *repository.TaskRepository) *TaskService {
	return &TaskService{
		repository: repository,
	}
}

func (s *TaskService) CreateTask(
	ctx context.Context,
	title string,
	description string,
) (model.Task, error) {
	if strings.TrimSpace(title) == "" {
		return model.Task{}, ErrInvalidTask
	}

	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

	task := model.Task{
		ID:          id.String(),
		Title:       title,
		Description: description,
		Completed:   false,
	}

	err := s.repository.CreateTask(ctx, task)
	if err != nil {
		return model.Task{}, err
	}

	return task, nil
}

func (s *TaskService) GetTask(
	ctx context.Context,
	id string,
) (model.Task, error) {
	return s.repository.GetTask(ctx, id)
}

func (s *TaskService) GetTasks(
	ctx context.Context,
) ([]model.Task, error) {
	return s.repository.GetTasks(ctx)
}

func (s *TaskService) UpdateTask(
	ctx context.Context,
	task model.Task,
) error {
	if strings.TrimSpace(task.Title) == "" {
		return ErrInvalidTask
	}

	return s.repository.UpdateTask(ctx, task)
}

func (s *TaskService) DeleteTask(
	ctx context.Context,
	id string,
) error {
	return s.repository.DeleteTask(ctx, id)
}
