package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var errInvalidBody = errors.New("invalid request body")

// allowedUpdateFields lists the exact JSON keys accepted by PUT and PATCH.
//
// encoding/json matches struct fields case-insensitively, so a key such as
// "Title" would otherwise be accepted as "title" even with
// DisallowUnknownFields enabled. The raw key check below rejects every key
// that is not an exact match for one of these three.
var allowedUpdateFields = map[string]struct{}{
	"title":       {},
	"description": {},
	"completed":   {},
}

func parseUpdateBody(body string) (UpdateTaskRequest, error) {
	var input UpdateTaskRequest

	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return input, errInvalidBody
	}

	if trimmed[0] != '{' {
		return input, errInvalidBody
	}

	var raw map[string]json.RawMessage

	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return input, errInvalidBody
	}

	for key, value := range raw {
		if _, ok := allowedUpdateFields[key]; !ok {
			return input, errInvalidBody
		}

		if string(bytes.TrimSpace(value)) == "null" {
			return input, errInvalidBody
		}
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		return input, errInvalidBody
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return input, errInvalidBody
	}

	return input, nil
}
