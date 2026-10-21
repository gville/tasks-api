package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"tasks-api/internal/model"
)

type stubHTTPClient struct {
	statusCode int
	body       string
	err        error

	requests []*http.Request
	bodies   []string
}

func (s *stubHTTPClient) Do(request *http.Request) (*http.Response, error) {
	s.requests = append(s.requests, request)

	var body string

	if request.Body != nil {
		raw, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			return nil, readErr
		}

		body = string(raw)

		request.Body = io.NopCloser(bytes.NewReader(raw))
	}

	s.bodies = append(s.bodies, body)

	if s.err != nil {
		return nil, s.err
	}

	return &http.Response{
		StatusCode: s.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(s.body)),
		Header:     http.Header{"Content-Type": []string{"application/x-amz-json-1.0"}},
	}, nil
}

func (s *stubHTTPClient) lastRequest() *http.Request {
	if len(s.requests) == 0 {
		return nil
	}

	return s.requests[len(s.requests)-1]
}

func (s *stubHTTPClient) lastBody() string {
	if len(s.bodies) == 0 {
		return ""
	}

	return s.bodies[len(s.bodies)-1]
}

func newStubRepository(client *stubHTTPClient) *TaskRepository {
	dynamoClient := dynamodb.New(dynamodb.Options{
		Region:             "us-east-2",
		Credentials:        aws.AnonymousCredentials{},
		HTTPClient:         client,
		EndpointResolverV2: nil,
	})

	return NewTaskRepository(dynamoClient, "stub-table")
}

func successBody(attributes map[string]interface{}) string {
	payload := map[string]interface{}{
		"Attributes": attributes,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	return string(raw)
}

func TestUpdateTaskMapsConditionalFailureToNotFound(t *testing.T) {
	client := &stubHTTPClient{
		statusCode: http.StatusBadRequest,
		body:       `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`,
	}

	repository := newStubRepository(client)

	_, err := repository.UpdateTask(
		context.Background(),
		model.Task{ID: "01TEST", Title: "t", Description: "d"},
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPatchTaskMapsConditionalFailureToNotFound(t *testing.T) {
	client := &stubHTTPClient{
		statusCode: http.StatusBadRequest,
		body:       `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`,
	}

	repository := newStubRepository(client)

	completed := true

	_, err := repository.PatchTask(
		context.Background(),
		"01TEST",
		model.TaskUpdate{Completed: &completed},
	)

	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTaskPassesThroughOtherErrors(t *testing.T) {
	client := &stubHTTPClient{
		statusCode: http.StatusInternalServerError,
		body:       `{"__type":"com.amazonaws.dynamodb.v20120810#InternalServerError","message":"Internal failure"}`,
	}

	repository := newStubRepository(client)

	_, err := repository.UpdateTask(
		context.Background(),
		model.Task{ID: "01TEST", Title: "t", Description: "d"},
	)

	if err == nil {
		t.Fatal("expected an error")
	}

	if errors.Is(err, ErrTaskNotFound) {
		t.Fatal("expected a non-not-found error to be preserved, not mapped to 404")
	}

	var apiErr interface{ ErrorCode() string }

	if !errors.As(err, &apiErr) || apiErr.ErrorCode() == "" {
		t.Fatalf("expected a typed DynamoDB error, got %v", err)
	}
}

func TestPatchTaskPassesThroughOtherErrors(t *testing.T) {
	client := &stubHTTPClient{
		statusCode: http.StatusInternalServerError,
		body:       `{"__type":"com.amazonaws.dynamodb.v20120810#InternalServerError","message":"Internal failure"}`,
	}

	repository := newStubRepository(client)

	completed := true

	_, err := repository.PatchTask(
		context.Background(),
		"01TEST",
		model.TaskUpdate{Completed: &completed},
	)

	if err == nil {
		t.Fatal("expected an error")
	}

	if errors.Is(err, ErrTaskNotFound) {
		t.Fatal("expected a non-not-found error to be preserved, not mapped to 404")
	}

	var apiErr interface{ ErrorCode() string }

	if !errors.As(err, &apiErr) || apiErr.ErrorCode() == "" {
		t.Fatalf("expected a typed DynamoDB error, got %v", err)
	}
}

func decodeRequest(t *testing.T, client *stubHTTPClient) map[string]json.RawMessage {
	t.Helper()

	request := client.lastRequest()
	if request == nil {
		t.Fatal("expected the stub to receive a request")
	}

	if request.Method != http.MethodPost {
		t.Fatalf("expected method POST, got %s", request.Method)
	}

	target := request.Header.Get("X-Amz-Target")

	expectedTarget := "DynamoDB_20120810.UpdateItem"
	if target != expectedTarget {
		t.Fatalf("expected X-Amz-Target %q, got %q", expectedTarget, target)
	}

	var payload map[string]json.RawMessage

	if err := json.Unmarshal([]byte(client.lastBody()), &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	return payload
}

func attributeNames(t *testing.T, payload map[string]json.RawMessage) map[string]string {
	t.Helper()

	raw, ok := payload["ExpressionAttributeNames"]
	if !ok {
		t.Fatal("expected ExpressionAttributeNames in the request")
	}

	var names map[string]string

	if err := json.Unmarshal(raw, &names); err != nil {
		t.Fatalf("failed to decode ExpressionAttributeNames: %v", err)
	}

	return names
}

func attributeValues(t *testing.T, payload map[string]json.RawMessage) map[string]json.RawMessage {
	t.Helper()

	raw, ok := payload["ExpressionAttributeValues"]
	if !ok {
		t.Fatal("expected ExpressionAttributeValues in the request")
	}

	var values map[string]json.RawMessage

	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatalf("failed to decode ExpressionAttributeValues: %v", err)
	}

	return values
}

func updateExpression(t *testing.T, payload map[string]json.RawMessage) string {
	t.Helper()

	raw, ok := payload["UpdateExpression"]
	if !ok {
		t.Fatal("expected UpdateExpression in the request")
	}

	var expression string

	if err := json.Unmarshal(raw, &expression); err != nil {
		t.Fatalf("failed to decode UpdateExpression: %v", err)
	}

	return expression
}

func assertStringAttribute(t *testing.T, values map[string]json.RawMessage, alias string, want string) {
	t.Helper()

	raw, ok := values[alias]
	if !ok {
		t.Fatalf("expected value %q in ExpressionAttributeValues", alias)
	}

	var attribute map[string]string

	if err := json.Unmarshal(raw, &attribute); err != nil {
		t.Fatalf("failed to decode value %q: %v", alias, err)
	}

	if attribute["S"] != want {
		t.Fatalf("expected %s S=%q, got %+v", alias, want, attribute)
	}
}

func assertBoolAttribute(t *testing.T, values map[string]json.RawMessage, alias string, want bool) {
	t.Helper()

	raw, ok := values[alias]
	if !ok {
		t.Fatalf("expected value %q in ExpressionAttributeValues", alias)
	}

	var attribute map[string]bool

	if err := json.Unmarshal(raw, &attribute); err != nil {
		t.Fatalf("failed to decode value %q: %v", alias, err)
	}

	if attribute["BOOL"] != want {
		t.Fatalf("expected %s BOOL=%t, got %+v", alias, want, attribute)
	}
}

func assertSharedContract(t *testing.T, payload map[string]json.RawMessage) {
	t.Helper()

	key, ok := payload["Key"]
	if !ok {
		t.Fatal("expected Key in the request")
	}

	var keyMap map[string]map[string]string

	if err := json.Unmarshal(key, &keyMap); err != nil {
		t.Fatalf("failed to decode Key: %v", err)
	}

	if keyMap["id"]["S"] == "" {
		t.Fatalf("expected Key id to be present, got %s", string(key))
	}

	condition, ok := payload["ConditionExpression"]
	if !ok {
		t.Fatal("expected ConditionExpression in the request")
	}

	var conditionValue string

	if err := json.Unmarshal(condition, &conditionValue); err != nil {
		t.Fatalf("failed to decode ConditionExpression: %v", err)
	}

	if conditionValue != "attribute_exists(id)" {
		t.Fatalf("expected existence condition, got %q", conditionValue)
	}

	returnValues, ok := payload["ReturnValues"]
	if !ok {
		t.Fatal("expected ReturnValues in the request")
	}

	var returnValue string

	if err := json.Unmarshal(returnValues, &returnValue); err != nil {
		t.Fatalf("failed to decode ReturnValues: %v", err)
	}

	if returnValue != "ALL_NEW" {
		t.Fatalf("expected ReturnValues ALL_NEW, got %q", returnValue)
	}
}

func TestUpdateTaskSendsFullReplacementRequest(t *testing.T) {
	client := &stubHTTPClient{
		statusCode: http.StatusOK,
		body: successBody(map[string]interface{}{
			"id":          map[string]string{"S": "01TEST"},
			"title":       map[string]string{"S": "Comprar leche"},
			"description": map[string]string{"S": "Entera"},
			"completed":   map[string]bool{"BOOL": true},
		}),
	}

	repository := newStubRepository(client)

	result, err := repository.UpdateTask(
		context.Background(),
		model.Task{
			ID:          "01TEST",
			Title:       "Comprar leche",
			Description: "Entera",
			Completed:   true,
		},
	)
	if err != nil {
		t.Fatalf("UpdateTask returned error: %v", err)
	}

	payload := decodeRequest(t, client)
	assertSharedContract(t, payload)

	names := attributeNames(t, payload)

	if names["#title"] != "title" ||
		names["#description"] != "description" ||
		names["#completed"] != "completed" {
		t.Fatalf("expected all attribute names aliased, got %+v", names)
	}

	expression := updateExpression(t, payload)

	for _, fragment := range []string{"#title = :title", "#description = :description", "#completed = :completed"} {
		if !strings.Contains(expression, fragment) {
			t.Fatalf("expected UpdateExpression to contain %q, got %q", fragment, expression)
		}
	}

	values := attributeValues(t, payload)

	assertStringAttribute(t, values, ":title", "Comprar leche")
	assertStringAttribute(t, values, ":description", "Entera")
	assertBoolAttribute(t, values, ":completed", true)

	want := model.Task{
		ID:          "01TEST",
		Title:       "Comprar leche",
		Description: "Entera",
		Completed:   true,
	}

	if result != want {
		t.Fatalf("expected deserialized task %+v, got %+v", want, result)
	}
}

func TestPatchTaskSendsOnlyProvidedAttributes(t *testing.T) {
	cases := []struct {
		name            string
		update          model.TaskUpdate
		expectedAliases map[string]string
		absentAliases   []string
		check           func(*testing.T, map[string]json.RawMessage)
	}{
		{
			name: "only title",
			update: model.TaskUpdate{
				Title: stringPtr("Nuevo titulo"),
			},
			expectedAliases: map[string]string{"#title": "title"},
			absentAliases:   []string{"#description", "#completed", ":description", ":completed"},
			check: func(t *testing.T, values map[string]json.RawMessage) {
				assertStringAttribute(t, values, ":title", "Nuevo titulo")
			},
		},
		{
			name: "only description",
			update: model.TaskUpdate{
				Description: stringPtr("Nueva descripcion"),
			},
			expectedAliases: map[string]string{"#description": "description"},
			absentAliases:   []string{"#title", "#completed", ":title", ":completed"},
			check: func(t *testing.T, values map[string]json.RawMessage) {
				assertStringAttribute(t, values, ":description", "Nueva descripcion")
			},
		},
		{
			name: "only completed false",
			update: model.TaskUpdate{
				Completed: boolPtr(false),
			},
			expectedAliases: map[string]string{"#completed": "completed"},
			absentAliases:   []string{"#title", "#description", ":title", ":description"},
			check: func(t *testing.T, values map[string]json.RawMessage) {
				assertBoolAttribute(t, values, ":completed", false)
			},
		},
		{
			name: "title and completed",
			update: model.TaskUpdate{
				Title:     stringPtr("Combinado"),
				Completed: boolPtr(true),
			},
			expectedAliases: map[string]string{
				"#title":     "title",
				"#completed": "completed",
			},
			absentAliases: []string{"#description", ":description"},
			check: func(t *testing.T, values map[string]json.RawMessage) {
				assertStringAttribute(t, values, ":title", "Combinado")
				assertBoolAttribute(t, values, ":completed", true)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubHTTPClient{
				statusCode: http.StatusOK,
				body: successBody(map[string]interface{}{
					"id":          map[string]string{"S": "01TEST"},
					"title":       map[string]string{"S": "Persistido"},
					"description": map[string]string{"S": "Persistida"},
					"completed":   map[string]bool{"BOOL": false},
				}),
			}

			repository := newStubRepository(client)

			result, err := repository.PatchTask(
				context.Background(),
				"01TEST",
				tc.update,
			)
			if err != nil {
				t.Fatalf("PatchTask returned error: %v", err)
			}

			payload := decodeRequest(t, client)
			assertSharedContract(t, payload)

			names := attributeNames(t, payload)

			if len(names) != len(tc.expectedAliases) {
				t.Fatalf(
					"expected exactly %d attribute names, got %d (%+v)",
					len(tc.expectedAliases),
					len(names),
					names,
				)
			}

			for alias, attribute := range tc.expectedAliases {
				if names[alias] != attribute {
					t.Fatalf("expected alias %q -> %q, got %+v", alias, attribute, names)
				}
			}

			values := attributeValues(t, payload)

			for _, alias := range tc.absentAliases {
				if _, ok := names[alias]; ok {
					t.Fatalf("did not expect attribute name %q, got %+v", alias, names)
				}

				if _, ok := values[alias]; ok {
					t.Fatalf("did not expect attribute value %q, got %+v", alias, values)
				}
			}

			expression := updateExpression(t, payload)

			for alias := range tc.expectedAliases {
				if !strings.Contains(expression, alias) {
					t.Fatalf("expected UpdateExpression to contain %q, got %q", alias, expression)
				}
			}

			for _, alias := range tc.absentAliases {
				if strings.Contains(expression, alias) {
					t.Fatalf("did not expect UpdateExpression to contain %q, got %q", alias, expression)
				}
			}

			tc.check(t, values)

			want := model.Task{
				ID:          "01TEST",
				Title:       "Persistido",
				Description: "Persistida",
				Completed:   false,
			}

			if result != want {
				t.Fatalf("expected deserialized task %+v, got %+v", want, result)
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
