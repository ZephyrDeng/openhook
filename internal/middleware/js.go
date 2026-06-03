package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZephyrDeng/openhook/internal/model"
	"github.com/dop251/goja"
)

type State struct {
	Ctx     map[string]any
	Global  map[string]any
	Headers map[string]string
}

type Result struct {
	Rejected bool
	Message  string
}

func RunInline(code string, state *State) (Result, error) {
	if code == "" {
		return Result{}, nil
	}
	return run("inline", code, state)
}

func RunAll(items []model.CustomMiddleware, state *State) (Result, error) {
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		result, err := run(item.Name, item.Code, state)
		if err != nil {
			return Result{}, err
		}
		if result.Rejected {
			return result, nil
		}
	}
	return Result{}, nil
}

func run(name, code string, state *State) (Result, error) {
	vm := goja.New()
	_ = vm.Set("ctx", state.Ctx)
	_ = vm.Set("global", state.Global)
	_ = vm.Set("headers", state.Headers)
	_ = vm.Set("now", func() int64 { return time.Now().UnixMilli() })
	_ = vm.Set("json", func(value any) string {
		raw, _ := json.Marshal(value)
		return string(raw)
	})

	wrapped := fmt.Sprintf(`(function() { %s })()`, code)
	value, err := vm.RunString(wrapped)
	if err != nil {
		return Result{}, fmt.Errorf("%s middleware: %w", name, err)
	}
	state.Ctx = exportObject(vm.Get("ctx"))
	state.Global = exportObject(vm.Get("global"))
	state.Headers = exportStringMap(vm.Get("headers"))

	if goja.IsUndefined(value) || goja.IsNull(value) {
		return Result{}, nil
	}
	exported := value.Export()
	switch typed := exported.(type) {
	case bool:
		if !typed {
			return Result{Rejected: true, Message: "middleware returned false"}, nil
		}
	case map[string]any:
		if reject, _ := typed["reject"].(bool); reject {
			message, _ := typed["message"].(string)
			if message == "" {
				message = "middleware rejected request"
			}
			return Result{Rejected: true, Message: message}, nil
		}
	}
	return Result{}, nil
}

func exportObject(value goja.Value) map[string]any {
	exported := value.Export()
	if object, ok := exported.(map[string]any); ok && object != nil {
		return object
	}
	return map[string]any{}
}

func exportStringMap(value goja.Value) map[string]string {
	exported := value.Export()
	result := map[string]string{}
	switch object := exported.(type) {
	case map[string]string:
		return object
	case map[string]any:
		for key, value := range object {
			result[key] = fmt.Sprint(value)
		}
	}
	return result
}
