package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ZephyrDeng/openhook/internal/model"
)

func TestTemplateRouteTokenRuleAndDeliveryPersistence(t *testing.T) {
	db := openStoreTestDB(t)
	defer db.Close()
	st := New(db)

	template, err := st.CreateTemplate(model.TemplateInput{
		TemplateName: "checkout-alert",
		Content:      "title: {{data.title}}",
		MsgType:      "markdown",
		CreateBy:     "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if template.TemplateID == "" || template.TemplateName != "checkout-alert" || template.MsgType != "markdown" {
		t.Fatalf("unexpected template: %#v", template)
	}

	route, err := st.CreateRoute(model.RouteInput{
		Name:       "checkout-route",
		TemplateID: template.TemplateID,
		TargetURLs: []string{"https://example.com/hook"},
		Headers:    map[string]string{"X-Route": "yes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.Mode != "envelope" || !route.Enabled || route.Headers["X-Route"] != "yes" {
		t.Fatalf("unexpected route defaults: %#v", route)
	}

	disabled := false
	route, err = st.UpdateRoute(route.RouteID, model.RouteInput{Enabled: &disabled, TargetURLs: []string{"https://example.com/updated"}})
	if err != nil {
		t.Fatal(err)
	}
	if route.Enabled || route.TargetURLs[0] != "https://example.com/updated" || route.Name != "checkout-route" {
		t.Fatalf("unexpected updated route: %#v", route)
	}

	token, err := st.CreateToken(model.TokenInput{Name: "scoped", TemplateIDs: []string{template.TemplateID}})
	if err != nil {
		t.Fatal(err)
	}
	canEdit, err := st.TokenCanEditTemplate(token.Token, template.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	if !canEdit {
		t.Fatal("expected token to edit scoped template")
	}
	canEdit, err = st.TokenCanEditTemplate(token.Token, "tpl_other")
	if err != nil {
		t.Fatal(err)
	}
	if canEdit {
		t.Fatal("expected token to reject unscoped template")
	}

	expired, err := st.CreateToken(model.TokenInput{
		Name:        "expired",
		TemplateIDs: []string{template.TemplateID},
		ExpireAt:    time.Now().Add(-time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	canEdit, err = st.TokenCanEditTemplate(expired.Token, template.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	if canEdit {
		t.Fatal("expected expired token to reject edits")
	}

	filter, err := st.CreateRuleSet("filter", model.RuleSetInput{
		Name:     "critical-only",
		Status:   true,
		Domain:   []string{"checkout", "payment"},
		Platform: "generic",
		Payload:  json.RawMessage(`{"severity":"critical"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := st.ListRuleSets("filter", map[string][]string{"domain": {"checkout"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != filter.ID {
		t.Fatalf("unexpected domain matches: %#v", matches)
	}
	matches, err = st.ListRuleSets("filter", map[string][]string{"domain": {"inventory"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected filtered rules: %#v", matches)
	}

	if err := st.CreateDelivery(model.Delivery{
		RequestID:    "req_store",
		RouteID:      route.RouteID,
		TemplateID:   template.TemplateID,
		TargetURL:    "https://example.com/***",
		StatusCode:   httpStatusOK,
		Success:      true,
		RequestBody:  json.RawMessage(`{"ok":true}`),
		ResponseBody: `{"received":true}`,
	}); err != nil {
		t.Fatal(err)
	}
	deliveries, err := st.ListDeliveries(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].RequestID != "req_store" || !deliveries[0].Success {
		t.Fatalf("unexpected deliveries: %#v", deliveries)
	}

	if err := st.DeleteRoute(route.RouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetRoute(route.RouteID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing route, got %v", err)
	}
}

const httpStatusOK = 200

func openStoreTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}
