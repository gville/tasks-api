package service

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"tasks-api/internal/model"

	"github.com/oklog/ulid/v2"
)

var ErrInvalidTask = errors.New("invalid task")

type TaskRepository interface {
	CreateTask(
		ctx context.Context,
		task model.Task,
	) error

	GetTask(
		ctx context.Context,
		id string,
	) (model.Task, error)

	GetTasks(
		ctx context.Context,
	) ([]model.Task, error)

	UpdateTask(
		ctx context.Context,
		task model.Task,
	) (model.Task, error)

	PatchTask(
		ctx context.Context,
		id string,
		update model.TaskUpdate,
	) (model.Task, error)

	DeleteTask(
		ctx context.Context,
		id string,
	) error
}

type TaskService struct {
	repository TaskRepository
}

func NewTaskService(repository TaskRepository) *TaskService {
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
) (model.Task, error) {
	if strings.TrimSpace(task.Title) == "" {
		return model.Task{}, ErrInvalidTask
	}

	return s.repository.UpdateTask(ctx, task)
}

func (s *TaskService) PatchTask(
	ctx context.Context,
	id string,
	update model.TaskUpdate,
) (model.Task, error) {
	if !update.HasAnyField() {
		return model.Task{}, ErrInvalidTask
	}

	if update.Title != nil && strings.TrimSpace(*update.Title) == "" {
		return model.Task{}, ErrInvalidTask
	}

	return s.repository.PatchTask(ctx, id, update)
}

func (s *TaskService) DeleteTask(
	ctx context.Context,
	id string,
) error {
	return s.repository.DeleteTask(ctx, id)
}
