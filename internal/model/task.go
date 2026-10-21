package model

type Task struct {
	ID          string `dynamodbav:"id" json:"id"`
	Title       string `dynamodbav:"title" json:"title"`
	Description string `dynamodbav:"description" json:"description"`
	Completed   bool   `dynamodbav:"completed" json:"completed"`
}

type TaskUpdate struct {
	Title       *string
	Description *string
	Completed   *bool
}

func (u TaskUpdate) HasAnyField() bool {
	return u.Title != nil || u.Description != nil || u.Completed != nil
}
