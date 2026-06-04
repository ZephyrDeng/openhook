package render

import "testing"

func TestTemplateRendersHandlebarsStylePaths(t *testing.T) {
	got, err := Template("hello {{data.user.name}} from {{global.routeId}}", map[string]any{
		"user": map[string]any{"name": "Ada"},
	}, map[string]any{"routeId": "rt_1"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello Ada from rt_1" {
		t.Fatalf("unexpected render: %#v", got)
	}
}

func TestTemplateReturnsJSONObject(t *testing.T) {
	got, err := Template(`{"title":"{{data.title}}","count":{{data.count}}}`, map[string]any{
		"title": "alarm",
		"count": 2,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	object, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if object["title"] != "alarm" {
		t.Fatalf("unexpected object: %#v", object)
	}
}

func TestTemplateJSONPlaceholderEscapesStringValues(t *testing.T) {
	got, err := Template(`{"text":{{json data.text}}}`, map[string]any{
		"text": "line one\nline \"two\"",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	object, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if object["text"] != "line one\nline \"two\"" {
		t.Fatalf("unexpected object: %#v", object)
	}
}
