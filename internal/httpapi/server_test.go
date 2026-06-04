package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ZephyrDeng/openhook/internal/config"
	"github.com/ZephyrDeng/openhook/internal/model"
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
	app := NewServer(store.New(db), config.Config{RequestTimeout: 0}, nil)

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

func TestRouteDeliverRedactsTargetURLInResponse(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer target.Close()

	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{RequestTimeout: 0}, nil)

	targetURL := target.URL + "/webhook/send?key=secret-robot-key"
	templateID := createTemplate(t, app)
	routeID := createRoute(t, app, templateID, targetURL)

	req := httptest.NewRequest(http.MethodPost, "/api/routes/"+routeID+"/deliver", bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, targetURL) || strings.Contains(body, "secret-robot-key") {
		t.Fatalf("target url leaked in response: %s", body)
	}
	if !strings.Contains(body, `"targetUrl"`) {
		t.Fatalf("response missing targetUrl field: %s", body)
	}
}

func TestAdminTokenProtectsWriteAPIs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{AdminToken: "secret"}, nil)

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

func TestAdminTokenProtectsSensitiveReadAPIs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{AdminToken: "secret"}, nil)

	templateID := createTemplateWithToken(t, app, "secret")
	routeID := createRouteWithToken(t, app, templateID, "https://example.com/webhook", "secret")

	for _, path := range []string{
		"/api/routes",
		"/api/routes/" + routeID,
		"/api/tokens",
		"/api/deliveries",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s without token status=%d body=%s", path, rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-OpenHook-Admin-Token", "secret")
		rec = httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s with token status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAuthProtectsDirectTemplateWebhookWhenEnabled(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer target.Close()

	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{AdminToken: "secret"}, nil)

	templateID := createTemplateWithToken(t, app, "secret")

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+templateID+"?webhookUrls="+url.QueryEscape(target.URL), bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("without token status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/webhook/"+templateID+"?webhookUrls="+url.QueryEscape(target.URL), bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenHook-Admin-Token", "secret")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("with token status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthenticatedUserCannotUseAnotherUsersTemplateWebhook(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer target.Close()

	db := openTestDB(t)
	defer db.Close()
	st := store.New(db)
	app := NewServer(st, config.Config{GitHubClientID: "client", GitHubClientSecret: "secret"}, nil)

	aliceCookie := createTestSession(t, st, "1001", "alice")
	bobCookie := createTestSession(t, st, "1002", "bob")
	aliceTemplateID := createTemplateWithCookie(t, app, aliceCookie, "alice-template")

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+aliceTemplateID+"?webhookUrls="+url.QueryEscape(target.URL), bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(bobCookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob using alice template status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/webhook/"+aliceTemplateID+"?webhookUrls="+url.QueryEscape(target.URL), bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(aliceCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice using own template status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompatibilityWebhookRouteIDRespectsDisabledRoute(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	st := store.New(db)
	template, err := st.CreateTemplate(model.TemplateInput{
		TemplateName: "disabled-route-template",
		Content:      "title: {{data.title}}",
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := false
	route, err := st.CreateRoute(model.RouteInput{
		Name:       "disabled-route",
		TemplateID: template.TemplateID,
		TargetURLs: []string{"https://example.com/webhook"},
		Enabled:    &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	app := NewServer(st, config.Config{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook/gitlab?routeId="+route.RouteID, bytes.NewBufferString(`{"object_kind":"merge_request"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthMeReportsAuthRequirement(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{AdminToken: "secret"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"authRequired":true`) || !strings.Contains(rec.Body.String(), `"authenticated":false`) {
		t.Fatalf("unexpected unauthenticated body: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("X-OpenHook-Admin-Token", "secret")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"authRequired":true`) || !strings.Contains(rec.Body.String(), `"authenticated":true`) || !strings.Contains(rec.Body.String(), `"admin":true`) {
		t.Fatalf("unexpected authenticated body: %s", rec.Body.String())
	}
}

func TestAuthenticatedUsersSeeOnlyOwnedTemplatesAndRoutes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	st := store.New(db)
	app := NewServer(st, config.Config{GitHubClientID: "client", GitHubClientSecret: "secret"}, nil)

	aliceCookie := createTestSession(t, st, "1001", "alice")
	bobCookie := createTestSession(t, st, "1002", "bob")

	aliceTemplateID := createTemplateWithCookie(t, app, aliceCookie, "alice-template")
	createRouteWithCookie(t, app, aliceCookie, aliceTemplateID, "https://example.com/alice")
	bobTemplateID := createTemplateWithCookie(t, app, bobCookie, "bob-template")
	createRouteWithCookie(t, app, bobCookie, bobTemplateID, "https://example.com/bob")

	assertListContainsOnly(t, app, aliceCookie, "/api/templates", "alice-template")
	assertListContainsOnly(t, app, aliceCookie, "/api/routes", "alice")
	assertListContainsOnly(t, app, bobCookie, "/api/templates", "bob-template")
	assertListContainsOnly(t, app, bobCookie, "/api/routes", "bob")
}

func TestPublicTemplatesAreReusableButEditableOnlyByOwner(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	st := store.New(db)
	app := NewServer(st, config.Config{GitHubClientID: "client", GitHubClientSecret: "secret"}, nil)

	aliceCookie := createTestSession(t, st, "1001", "alice")
	bobCookie := createTestSession(t, st, "1002", "bob")

	privateID := createTemplateWithCookieAndBody(t, app, aliceCookie, `{"templateName":"alice-private","msgType":"markdown","content":"private: {{data.title}}"}`)
	publicID := createTemplateWithCookieAndBody(t, app, aliceCookie, `{"templateName":"alice-public","msgType":"markdown","content":"public: {{data.title}}","visibility":"public"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	req.AddCookie(bobCookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob templates status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "alice-private") || strings.Contains(rec.Body.String(), privateID) {
		t.Fatalf("private template leaked to bob: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "alice-public") || !strings.Contains(rec.Body.String(), `"visibility":"public"`) {
		t.Fatalf("public template missing from bob list: %s", rec.Body.String())
	}

	createRouteWithCookie(t, app, bobCookie, publicID, "https://example.com/bob-public")

	req = httptest.NewRequest(http.MethodPut, "/api/templates/"+publicID, bytes.NewBufferString(`{"templateName":"bob-edited","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(bobCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob editing public template status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/templates/"+publicID, bytes.NewBufferString(`{"templateName":"alice-public-updated","content":"public: {{data.title}}","visibility":"public"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(aliceCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice editing public template status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthenticatedUserEmptyRoutesReturnsArray(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	st := store.New(db)
	app := NewServer(st, config.Config{GitHubClientID: "client", GitHubClientSecret: "secret"}, nil)
	cookie := createTestSession(t, st, "1003", "carol")

	req := httptest.NewRequest(http.MethodGet, "/api/routes", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Fatalf("empty route list should be an array: %s", rec.Body.String())
	}
}

func TestRouteMiddlewareTokenAndDeliveryUseCases(t *testing.T) {
	var received map[string]any
	var receivedHeader string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-OpenHook-Test")
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
	st := store.New(db)
	app := NewServer(st, config.Config{}, nil)

	template, err := st.CreateTemplate(model.TemplateInput{
		TemplateName: "middleware-template",
		Content:      "{\"title\":{{json data.title}}}",
		MsgType:      "markdown",
	})
	if err != nil {
		t.Fatal(err)
	}
	mw, err := st.CreateMiddleware(model.CustomMiddlewareInput{
		Name: "append-header",
		Code: `ctx.title = ctx.title + "-mw"; headers["X-OpenHook-Test"] = "yes"; return true;`,
	})
	if err != nil {
		t.Fatal(err)
	}
	route, err := st.CreateRoute(model.RouteInput{
		Name:          "middleware-route",
		TemplateID:    template.TemplateID,
		TargetURLs:    []string{target.URL + "/webhook?key=secret"},
		MiddlewareIDs: []string{mw.MiddlewareID},
		Mode:          "raw",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/routes/"+route.RouteID+"/deliver", bytes.NewBufferString(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deliver status=%d body=%s", rec.Code, rec.Body.String())
	}
	if receivedHeader != "yes" || received["title"] != "hello-mw" {
		t.Fatalf("middleware was not applied, header=%q body=%#v", receivedHeader, received)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/deliveries?limit=10", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deliveries status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), route.RouteID) || strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("delivery log missing route or leaked target secret: %s", rec.Body.String())
	}

	other, err := st.CreateTemplate(model.TemplateInput{TemplateName: "other-template", Content: "x"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := st.CreateToken(model.TokenInput{Name: "scoped-token", TemplateIDs: []string{template.TemplateID}})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, "/api/templates/"+template.TemplateID+"/token/"+token.Token, bytes.NewBufferString(`{"templateName":"middleware-template-updated","content":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("scoped token update status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, "/api/templates/"+other.TemplateID+"/token/"+token.Token, bytes.NewBufferString(`{"templateName":"other-updated","content":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("token updating out-of-scope template status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGitHubOAuthCallbackCreatesSession(t *testing.T) {
	var tokenRequested bool
	var userRequested bool
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			tokenRequested = true
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("client_id") != "client" || r.Form.Get("client_secret") != "secret" || r.Form.Get("code") != "oauth-code" {
				t.Fatalf("unexpected token form: %v", r.Form)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"github-token","token_type":"bearer"}`))
		case "/user":
			userRequested = true
			if r.Header.Get("Authorization") != "Bearer github-token" {
				t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1001,"login":"alice","name":"Alice","avatar_url":"https://example.com/alice.png"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer github.Close()

	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{
		PublicBaseURL:      "https://openhook.test",
		GitHubClientID:     "client",
		GitHubClientSecret: "secret",
		GitHubAuthURL:      github.URL + "/login/oauth/authorize",
		GitHubTokenURL:     github.URL + "/login/oauth/access_token",
		GitHubUserURL:      github.URL + "/user",
		SessionTTL:         time.Hour,
	}, nil)

	startReq := httptest.NewRequest(http.MethodGet, "/auth/github/start?returnTo=%2Froutes", nil)
	startRec := httptest.NewRecorder()
	app.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusFound {
		t.Fatalf("start status=%d body=%s", startRec.Code, startRec.Body.String())
	}
	redirectURL, err := url.Parse(startRec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := redirectURL.Query().Get("state")
	if state == "" || redirectURL.Query().Get("client_id") != "client" {
		t.Fatalf("unexpected redirect: %s", redirectURL.String())
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=oauth-code&state="+url.QueryEscape(state), nil)
	for _, cookie := range startRec.Result().Cookies() {
		callbackReq.AddCookie(cookie)
	}
	callbackRec := httptest.NewRecorder()
	app.ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", callbackRec.Code, callbackRec.Body.String())
	}
	if callbackRec.Header().Get("Location") != "/routes" {
		t.Fatalf("unexpected callback redirect: %q", callbackRec.Header().Get("Location"))
	}
	if !tokenRequested || !userRequested {
		t.Fatalf("github tokenRequested=%v userRequested=%v", tokenRequested, userRequested)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	for _, cookie := range callbackRec.Result().Cookies() {
		meReq.AddCookie(cookie)
	}
	meRec := httptest.NewRecorder()
	app.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", meRec.Code, meRec.Body.String())
	}
	if !strings.Contains(meRec.Body.String(), `"login":"alice"`) {
		t.Fatalf("unexpected me body: %s", meRec.Body.String())
	}

	templatesReq := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	for _, cookie := range callbackRec.Result().Cookies() {
		templatesReq.AddCookie(cookie)
	}
	templatesRec := httptest.NewRecorder()
	app.ServeHTTP(templatesRec, templatesReq)
	if templatesRec.Code != http.StatusOK {
		t.Fatalf("templates status=%d body=%s", templatesRec.Code, templatesRec.Body.String())
	}
	for _, want := range []string{"企微-机器人 Markdown", "Telegram-sendMessage", "QQ-Webhook 文本"} {
		if !strings.Contains(templatesRec.Body.String(), want) {
			t.Fatalf("templates response missing %q: %s", want, templatesRec.Body.String())
		}
	}
}

func TestGitHubLoginAndRegisterAliasesRedirectToOAuth(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	app := NewServer(store.New(db), config.Config{
		PublicBaseURL:      "https://openhook.test",
		GitHubClientID:     "client",
		GitHubClientSecret: "secret",
		GitHubAuthURL:      "https://github.test/login/oauth/authorize",
		SessionTTL:         time.Hour,
	}, nil)

	for _, path := range []string{"/login/github?returnTo=%2Froutes", "/register/github?returnTo=%2Ftemplates"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		redirectURL, err := url.Parse(rec.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		if redirectURL.Host != "github.test" || redirectURL.Query().Get("client_id") != "client" {
			t.Fatalf("%s unexpected redirect: %s", path, redirectURL.String())
		}
	}
}

func TestStarterTemplatesRenderRawJSONObjectsWithUnsafeText(t *testing.T) {
	user := storeUserForStarterTest()
	data := map[string]any{
		"chatId":      "123456789",
		"title":       `quote " title`,
		"severity":    "info",
		"service":     "openhook",
		"environment": "prod",
		"time":        "2026-06-04 00:00:00",
		"description": "line one\nline two with \"quote\"",
	}
	server := &Server{}
	for _, input := range starterTemplates(user) {
		rendered, _, err := server.renderWithInline(context.Background(), model.Template{
			Content: input.Content,
			Script:  input.Script,
			MsgType: input.MsgType,
		}, data, map[string]any{})
		if err != nil {
			t.Fatalf("%s render failed: %v", input.TemplateName, err)
		}
		if _, ok := rendered.(map[string]any); !ok {
			t.Fatalf("%s rendered as %T, want JSON object: %#v", input.TemplateName, rendered, rendered)
		}
	}
}

func storeUserForStarterTest() model.User {
	return model.User{
		UserID:     "usr_test",
		Provider:   "github",
		ProviderID: "1001",
		Login:      "alice",
		Name:       "Alice",
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

func createTestSession(t *testing.T, st *store.Store, providerID, login string) *http.Cookie {
	t.Helper()
	user, err := st.UpsertUser(store.UserInput{
		Provider:   "github",
		ProviderID: providerID,
		Login:      login,
		Name:       login,
		AvatarURL:  "https://example.com/" + login + ".png",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateSession(user.UserID, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: sessionCookieName, Value: session.Token}
}

func createTemplateWithCookie(t *testing.T, app http.Handler, cookie *http.Cookie, name string) string {
	return createTemplateWithCookieAndBody(t, app, cookie, `{"templateName":"`+name+`","msgType":"markdown","content":"title: {{data.title}}"}`)
}

func createTemplateWithCookieAndBody(t *testing.T, app http.Handler, cookie *http.Cookie, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
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

func createRouteWithCookie(t *testing.T, app http.Handler, cookie *http.Cookie, templateID, targetURL string) string {
	t.Helper()
	body := `{"name":"` + targetURL + `","templateId":"` + templateID + `","targetUrls":["` + targetURL + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
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

func assertListContainsOnly(t *testing.T, app http.Handler, cookie *http.Cookie, path, want string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("%s response missing %q: %s", path, want, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "alice") && strings.Contains(rec.Body.String(), "bob") {
		t.Fatalf("%s response contains multiple owners: %s", path, rec.Body.String())
	}
}

func createTemplate(t *testing.T, app http.Handler) string {
	return createTemplateWithToken(t, app, "")
}

func createTemplateWithToken(t *testing.T, app http.Handler, token string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewBufferString(`{"templateName":"test","msgType":"markdown","content":"title: {{data.title}}"}`))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-OpenHook-Admin-Token", token)
	}
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
	return createRouteWithToken(t, app, templateID, targetURL, "")
}

func createRouteWithToken(t *testing.T, app http.Handler, templateID, targetURL, token string) string {
	t.Helper()
	body := `{"name":"test","templateId":"` + templateID + `","targetUrls":["` + targetURL + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-OpenHook-Admin-Token", token)
	}
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
