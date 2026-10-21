package handler

import (
	"testing"
)

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func TestParseUpdateBodyAcceptsExactFieldNames(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		title       *string
		description *string
		completed   *bool
	}{
		{
			name:  "title only",
			body:  `{"title":"Comprar leche"}`,
			title: stringPtr("Comprar leche"),
		},
		{
			name:        "description only",
			body:        `{"description":"Entera"}`,
			description: stringPtr("Entera"),
		},
		{
			name:      "completed only",
			body:      `{"completed":true}`,
			completed: boolPtr(true),
		},
		{
			name:      "completed explicit false",
			body:      `{"completed":false}`,
			completed: boolPtr(false),
		},
		{
			name:        "all three exact fields",
			body:        `{"title":"Comprar leche","description":"Entera","completed":true}`,
			title:       stringPtr("Comprar leche"),
			description: stringPtr("Entera"),
			completed:   boolPtr(true),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := parseUpdateBody(tc.body)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			assertOptionalString(t, "title", input.Title, tc.title)
			assertOptionalString(t, "description", input.Description, tc.description)
			assertOptionalBool(t, "completed", input.Completed, tc.completed)
		})
	}
}

func TestParseUpdateBodyRejectsCaseVariantFieldNames(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"title capitalized", `{"Title":"Comprar leche"}`},
		{"description capitalized", `{"Description":"Entera"}`},
		{"completed capitalized", `{"Completed":true}`},
		{"title uppercase", `{"TITLE":"Comprar leche"}`},
		{"description uppercase", `{"DESCRIPTION":"Entera"}`},
		{"completed uppercase", `{"COMPLETED":true}`},
		{
			name: "capitalized title with exact other fields",
			body: `{"Title":"Comprar leche","description":"Entera","completed":true}`,
		},
		{
			name: "exact fields with capitalized description",
			body: `{"title":"Comprar leche","Description":"Entera","completed":true}`,
		},
		{
			name: "exact fields with capitalized completed",
			body: `{"title":"Comprar leche","description":"Entera","Completed":true}`,
		},
		{
			name: "mixed casing with exact fields",
			body: `{"title":"Comprar leche","Description":"Entera","COMPLETED":true}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseUpdateBody(tc.body)
			if err == nil {
				t.Fatalf("expected an error for body %s", tc.body)
			}
		})
	}
}

func TestParseUpdateBodyDistinguishesAbsentCompletedFromFalse(t *testing.T) {
	absent, err := parseUpdateBody(`{"title":"Comprar leche"}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if absent.Completed != nil {
		t.Fatalf("expected completed to be absent, got %v", *absent.Completed)
	}

	explicit, err := parseUpdateBody(`{"completed":false}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if explicit.Completed == nil {
		t.Fatal("expected completed to be present")
	}

	if *explicit.Completed {
		t.Fatal("expected completed to be false")
	}
}

func TestParseUpdateBodyRejectsInvalidBodies(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty body", ``},
		{"whitespace body", `   `},
		{"root null", `null`},
		{"root array", `[]`},
		{"root string", `"title"`},
		{"root number", `42`},
		{"root boolean", `false`},
		{"trailing object", `{"title":"Comprar leche"}{}`},
		{"trailing text", `{"title":"Comprar leche"} extra`},
		{"invalid json", `{"title":`},
		{"null title", `{"title":null}`},
		{"null description", `{"description":null}`},
		{"null completed", `{"completed":null}`},
		{"unknown field", `{"foo":1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseUpdateBody(tc.body)
			if err == nil {
				t.Fatalf("expected an error for body %s", tc.body)
			}
		})
	}
}

func assertOptionalString(
	t *testing.T,
	field string,
	got *string,
	want *string,
) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("expected %s to be absent, got %q", field, *got)
		}

		return
	}

	if got == nil {
		t.Fatalf("expected %s to be present", field)
	}

	if *got != *want {
		t.Fatalf("expected %s %q, got %q", field, *want, *got)
	}
}

func assertOptionalBool(
	t *testing.T,
	field string,
	got *bool,
	want *bool,
) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("expected %s to be absent, got %t", field, *got)
		}

		return
	}

	if got == nil {
		t.Fatalf("expected %s to be present", field)
	}

	if *got != *want {
		t.Fatalf("expected %s %t, got %t", field, *want, *got)
	}
}
