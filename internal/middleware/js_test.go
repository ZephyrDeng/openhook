package middleware

import "testing"

func TestRunInlineMutatesContext(t *testing.T) {
	state := &State{
		Ctx:     map[string]any{"severity": "warning"},
		Global:  map[string]any{},
		Headers: map[string]string{},
	}
	result, err := RunInline(`ctx.severity = "critical"; headers["X-Test"] = "ok"; return true;`, state)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rejected {
		t.Fatalf("unexpected reject: %#v", result)
	}
	if state.Ctx["severity"] != "critical" {
		t.Fatalf("ctx was not mutated: %#v", state.Ctx)
	}
	if state.Headers["X-Test"] != "ok" {
		t.Fatalf("headers were not mutated: %#v", state.Headers)
	}
}

func TestRunInlineRejects(t *testing.T) {
	state := &State{Ctx: map[string]any{}, Global: map[string]any{}, Headers: map[string]string{}}
	result, err := RunInline(`return { reject: true, message: "ignored" };`, state)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Rejected || result.Message != "ignored" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
