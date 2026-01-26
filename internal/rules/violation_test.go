package rules

import (
	"encoding/json"
	"testing"
)

func TestNewViolation(t *testing.T) {
	loc := NewLineLocation("Dockerfile", 5)
	v := NewViolation(loc, "test-rule", "test message", SeverityWarning)

	if v.RuleCode != "test-rule" {
		t.Errorf("RuleCode = %q, want %q", v.RuleCode, "test-rule")
	}
	if v.Message != "test message" {
		t.Errorf("Message = %q, want %q", v.Message, "test message")
	}
	if v.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", v.Severity, SeverityWarning)
	}
	if v.File() != "Dockerfile" {
		t.Errorf("File() = %q, want %q", v.File(), "Dockerfile")
	}
	if v.Line() != 5 {
		t.Errorf("Line() = %d, want 5", v.Line())
	}
}

func TestViolation_WithMethods(t *testing.T) {
	loc := NewLineLocation("Dockerfile", 1)
	v := NewViolation(loc, "rule", "msg", SeverityError).
		WithDetail("extra detail").
		WithDocURL("https://example.com/doc").
		WithSourceCode("FROM alpine")

	if v.Detail != "extra detail" {
		t.Errorf("Detail = %q, want %q", v.Detail, "extra detail")
	}
	if v.DocURL != "https://example.com/doc" {
		t.Errorf("DocURL = %q, want %q", v.DocURL, "https://example.com/doc")
	}
	if v.SourceCode != "FROM alpine" {
		t.Errorf("SourceCode = %q, want %q", v.SourceCode, "FROM alpine")
	}
}

func TestViolation_WithSuggestedFix(t *testing.T) {
	loc := NewRangeLocation("Dockerfile", 1, 1, 1, 12)
	fix := &SuggestedFix{
		Description: "Use specific tag",
		Edits: []TextEdit{
			{
				Location: loc,
				NewText:  "FROM alpine:3.18",
			},
		},
	}

	v := NewViolation(loc, "DL3006", "Always specify tag", SeverityWarning).
		WithSuggestedFix(fix)

	if v.SuggestedFix == nil {
		t.Fatal("SuggestedFix is nil")
	}
	if v.SuggestedFix.Description != "Use specific tag" {
		t.Errorf("SuggestedFix.Description = %q", v.SuggestedFix.Description)
	}
	if len(v.SuggestedFix.Edits) != 1 {
		t.Fatalf("len(Edits) = %d, want 1", len(v.SuggestedFix.Edits))
	}
	if v.SuggestedFix.Edits[0].NewText != "FROM alpine:3.18" {
		t.Errorf("Edit.NewText = %q", v.SuggestedFix.Edits[0].NewText)
	}
}

func TestViolation_JSON(t *testing.T) {
	loc := NewLineLocation("Dockerfile", 10)
	v := NewViolation(loc, "max-lines", "file too long", SeverityError).
		WithDocURL("https://example.com")

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Violation
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.RuleCode != v.RuleCode {
		t.Errorf("RuleCode = %q, want %q", parsed.RuleCode, v.RuleCode)
	}
	if parsed.Message != v.Message {
		t.Errorf("Message = %q, want %q", parsed.Message, v.Message)
	}
	if parsed.Severity != v.Severity {
		t.Errorf("Severity = %v, want %v", parsed.Severity, v.Severity)
	}
	if parsed.Line() != v.Line() {
		t.Errorf("Line() = %d, want %d", parsed.Line(), v.Line())
	}
}

func TestViolation_JSON_WithFix(t *testing.T) {
	loc := NewLineLocation("Dockerfile", 1)
	fix := &SuggestedFix{
		Description: "Fix the issue",
		Edits: []TextEdit{
			{Location: loc, NewText: "new text"},
		},
	}
	v := NewViolation(loc, "rule", "msg", SeverityWarning).WithSuggestedFix(fix)

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Violation
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.SuggestedFix == nil {
		t.Fatal("SuggestedFix is nil after unmarshal")
	}
	if parsed.SuggestedFix.Description != "Fix the issue" {
		t.Errorf("SuggestedFix.Description = %q", parsed.SuggestedFix.Description)
	}
}
