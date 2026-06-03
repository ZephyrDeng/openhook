package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ZephyrDeng/openhook/internal/config"
	"github.com/ZephyrDeng/openhook/internal/store"
)

func TestRouteDeliverForwardsRenderedPayload(t *testing.T) {
	var received map[string]any
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer target.Close()

	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{RequestTimeout: 0})

	templateID := createTemplate(t, app)
	routeID := createRoute(t, app, templateID, target.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/routes/"+routeID+"/deliver", bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if received["content"] != "title: hello" {
		t.Fatalf("unexpected forwarded body: %#v", received)
	}
}

func TestAdminTokenProtectsWriteAPIs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{AdminToken: "secret"})

	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewBufferString(`{"templateName":"blocked","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewBufferString(`{"templateName":"allowed","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenHook-Admin-Token", "secret")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func createTemplate(t *testing.T, app http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewBufferString(`{"templateName":"test","msgType":"markdown","content":"title: {{data.title}}"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("template status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data struct {
			TemplateID string `json:"templateId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data.TemplateID
}

func createRoute(t *testing.T, app http.Handler, templateID, targetURL string) string {
	t.Helper()
	body := `{"name":"test","templateId":"` + templateID + `","targetUrls":["` + targetURL + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("route status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data struct {
			RouteID string `json:"routeId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data.RouteID
}
