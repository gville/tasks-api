package model

type Task struct {
	ID          string `dynamodbav:"id" json:"id"`
	Title       string `dynamodbav:"title" json:"title"`
	Description string `dynamodbav:"description" json:"description"`
	Completed   bool   `dynamodbav:"completed" json:"completed"`
}
