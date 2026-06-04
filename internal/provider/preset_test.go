package provider

import (
	"encoding/json"
	"testing"

	jsmw "github.com/ZephyrDeng/openhook/internal/middleware"
	"github.com/ZephyrDeng/openhook/internal/render"
)

func TestProviderPresetsRenderJSONObjects(t *testing.T) {
	for _, preset := range All() {
		var simulation map[string]any
		if err := json.Unmarshal(preset.Simulation, &simulation); err != nil {
			t.Fatalf("%s simulation json: %v", preset.ID, err)
		}
		state := &jsmw.State{Ctx: simulation, Global: map[string]any{}, Headers: map[string]string{}}
		if _, err := jsmw.RunInline(preset.Script, state); err != nil {
			t.Fatalf("%s script failed: %v", preset.ID, err)
		}
		rendered, err := render.Template(preset.Content, state.Ctx, state.Global)
		if err != nil {
			t.Fatalf("%s render failed: %v", preset.ID, err)
		}
		if _, ok := rendered.(map[string]any); !ok {
			t.Fatalf("%s rendered as %T, want object: %#v", preset.ID, rendered, rendered)
		}
	}
}
