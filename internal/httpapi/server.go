package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZephyrDeng/openhook/internal/config"
	"github.com/ZephyrDeng/openhook/internal/forward"
	jsmw "github.com/ZephyrDeng/openhook/internal/middleware"
	"github.com/ZephyrDeng/openhook/internal/model"
	"github.com/ZephyrDeng/openhook/internal/render"
	"github.com/ZephyrDeng/openhook/internal/response"
	"github.com/ZephyrDeng/openhook/internal/store"
)

type Server struct {
	store  *store.Store
	cfg    config.Config
	sender *forward.Sender
}

func NewServer(st *store.Store, cfg config.Config) http.Handler {
	server := &Server{
		store:  st,
		cfg:    cfg,
		sender: forward.New(cfg.RequestTimeout),
	}
	return server.withCORS(http.HandlerFunc(server.route))
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := splitPath(path)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if path == "" || path == "health" {
		response.OK(w, map[string]any{"status": "ok"})
		return
	}

	switch {
	case len(parts) == 2 && parts[0] == "webhook" && parts[1] == "gitlab":
		s.handleGitlab(w, r)
	case len(parts) == 2 && parts[0] == "webhook" && parts[1] == "sentry":
		s.handleSentry(w, r)
	case len(parts) >= 1 && parts[0] == "api":
		s.routeAPI(w, r, parts[1:])
	case len(parts) == 2 && parts[0] == "webhook":
		s.handleTemplateWebhook(w, r, parts[1])
	default:
		response.Error(w, http.StatusNotFound, "route not found")
	}
}

func (s *Server) routeAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		response.OK(w, map[string]any{"name": "openhook"})
		return
	}
	switch parts[0] {
	case "templates", "message-template":
		s.handleTemplates(w, r, parts[1:])
	case "tokens", "token":
		s.handleTokens(w, r, parts[1:])
	case "routes":
		s.handleRoutes(w, r, parts[1:])
	case "middlewares":
		s.handleMiddlewares(w, r, parts[1:])
	case "filters":
		s.handleRuleSets("filter", w, r, parts[1:])
	case "dedup-rule", "dedup-rules":
		s.handleRuleSets("dedup", w, r, parts[1:])
	case "deliveries":
		s.handleDeliveries(w, r)
	default:
		response.Error(w, http.StatusNotFound, "api route not found")
	}
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.listTemplates(w, r)
		case http.MethodPost:
			if !s.authorized(w, r) {
				return
			}
			var input model.TemplateInput
			if !decodeJSON(w, r, &input) {
				return
			}
			item, err := s.store.CreateTemplate(input)
			writeStoreResult(w, item, err, http.StatusCreated)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "")
		}
		return
	}
	if len(parts) == 1 && parts[0] == "preview" && r.Method == http.MethodPost {
		var input struct {
			Content     string         `json:"content"`
			Simulation  map[string]any `json:"simulation"`
			Script      string         `json:"script"`
			AsyncScript string         `json:"asyncScript"`
			MsgType     string         `json:"msgType"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		content, _, err := s.renderWithInline(r.Context(), model.Template{Content: input.Content, MsgType: input.MsgType, Script: firstNonEmpty(input.AsyncScript, input.Script)}, input.Simulation, map[string]any{})
		writeStoreResult(w, content, err, http.StatusOK)
		return
	}
	if len(parts) == 1 && parts[0] == "paginated" && r.Method == http.MethodGet {
		s.listTemplates(w, r)
		return
	}
	templateID := parts[0]
	if len(parts) == 2 && parts[1] == "render" && r.Method == http.MethodPost {
		s.renderTemplateByID(w, r, templateID)
		return
	}
	if len(parts) == 3 && parts[1] == "token" && r.Method == http.MethodPut {
		var input model.TemplateInput
		if !decodeJSON(w, r, &input) {
			return
		}
		ok, err := s.store.TokenCanEditTemplate(parts[2], templateID)
		if err != nil || !ok {
			response.Error(w, http.StatusForbidden, "token cannot edit template")
			return
		}
		item, err := s.store.UpdateTemplate(templateID, input, "token:"+parts[2])
		writeStoreResult(w, item, err, http.StatusOK)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetTemplate(templateID)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		if !s.authorized(w, r) {
			return
		}
		var input model.TemplateInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateTemplate(templateID, input, input.CreateBy)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorized(w, r) {
			return
		}
		err := s.store.DeleteTemplate(templateID)
		writeStoreResult(w, map[string]string{"templateId": templateID}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "size", 100)
	if page < 1 {
		page = 1
	}
	items, total, err := s.store.ListTemplates(r.URL.Query().Get("search"), size, (page-1)*size, toColumn(r.URL.Query().Get("sortBy")), r.URL.Query().Get("sortOrder"))
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	if strings.Contains(r.URL.Path, "paginated") {
		response.OK(w, map[string]any{"items": items, "total": total, "page": page, "size": size})
		return
	}
	response.OK(w, items)
}

func (s *Server) renderTemplateByID(w http.ResponseWriter, r *http.Request, templateID string) {
	template, err := s.store.GetTemplate(templateID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	body, ok := decodeBodyMap(w, r)
	if !ok {
		return
	}
	content, _, err := s.renderWithInline(r.Context(), template, body, map[string]any{})
	writeStoreResult(w, content, err, http.StatusOK)
}

func (s *Server) handleTokens(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "")
			return
		}
		items, err := s.store.ListTokens()
		writeStoreResult(w, items, err, http.StatusOK)
		return
	}
	if len(parts) == 1 && parts[0] == "create" && r.Method == http.MethodPost {
		if !s.authorized(w, r) {
			return
		}
		var input model.TokenInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.CreateToken(input)
		writeStoreResult(w, item, err, http.StatusCreated)
		return
	}
	token := parts[0]
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetToken(token)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPost:
		if !s.authorized(w, r) {
			return
		}
		var input model.TokenInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateToken(token, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorized(w, r) {
			return
		}
		err := s.store.DeleteToken(token)
		writeStoreResult(w, map[string]string{"token": token}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListRoutes()
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			if !s.authorized(w, r) {
				return
			}
			var input model.RouteInput
			if !decodeJSON(w, r, &input) {
				return
			}
			item, err := s.store.CreateRoute(input)
			writeStoreResult(w, item, err, http.StatusCreated)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "")
		}
		return
	}
	routeID := parts[0]
	if len(parts) == 2 && parts[1] == "deliver" && r.Method == http.MethodPost {
		s.handleRouteDeliver(w, r, routeID)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetRoute(routeID)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		if !s.authorized(w, r) {
			return
		}
		var input model.RouteInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateRoute(routeID, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorized(w, r) {
			return
		}
		err := s.store.DeleteRoute(routeID)
		writeStoreResult(w, map[string]string{"routeId": routeID}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleMiddlewares(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListMiddlewares()
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			if !s.authorized(w, r) {
				return
			}
			var input model.CustomMiddlewareInput
			if !decodeJSON(w, r, &input) {
				return
			}
			item, err := s.store.CreateMiddleware(input)
			writeStoreResult(w, item, err, http.StatusCreated)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "")
		}
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetMiddleware(id)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		if !s.authorized(w, r) {
			return
		}
		var input model.CustomMiddlewareInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateMiddleware(id, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorized(w, r) {
			return
		}
		err := s.store.DeleteMiddleware(id)
		writeStoreResult(w, map[string]string{"middlewareId": id}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleRuleSets(kind string, w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			items, err := s.store.ListRuleSets(kind, r.URL.Query())
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			if !s.authorized(w, r) {
				return
			}
			var input model.RuleSetInput
			if !decodeJSON(w, r, &input) {
				return
			}
			item, err := s.store.CreateRuleSet(kind, input)
			writeStoreResult(w, item, err, http.StatusCreated)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "")
		}
		return
	}
	if len(parts) == 1 && parts[0] == "one" && r.Method == http.MethodGet {
		item, err := s.store.GetActiveRuleSet(kind, r.URL.Query())
		writeStoreResult(w, item, err, http.StatusOK)
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodPut:
		if !s.authorized(w, r) {
			return
		}
		var input model.RuleSetInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateRuleSet(kind, id, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorized(w, r) {
			return
		}
		err := s.store.DeleteRuleSet(kind, id)
		writeStoreResult(w, map[string]string{"id": id}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleDeliveries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	items, err := s.store.ListDeliveries(queryInt(r, "limit", 50), queryInt(r, "offset", 0))
	writeStoreResult(w, items, err, http.StatusOK)
}

func (s *Server) handleTemplateWebhook(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	body, ok := decodeBodyMap(w, r)
	if !ok {
		return
	}
	targets := splitCSV(r.URL.Query().Get("webhookUrls"))
	if len(targets) == 0 {
		response.Error(w, http.StatusBadRequest, "webhookUrls is required")
		return
	}
	template, err := s.store.GetTemplate(templateID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	requestID := requestID(r)
	content, state, err := s.renderWithInline(r.Context(), template, body, map[string]any{"query": queryObject(r), "requestId": requestID})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	results := s.sendToTargets(r.Context(), requestID, "", template.TemplateID, targets, map[string]string{}, "envelope", template.MsgType, content, state.Ctx)
	response.OK(w, results)
}

func (s *Server) handleRouteDeliver(w http.ResponseWriter, r *http.Request, routeID string) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	body, ok := decodeBodyMap(w, r)
	if !ok {
		return
	}
	route, err := s.store.GetRoute(routeID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	if !route.Enabled {
		response.Error(w, http.StatusForbidden, "route is disabled")
		return
	}
	template, err := s.store.GetTemplate(route.TemplateID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	requestID := requestID(r)
	global := map[string]any{"query": queryObject(r), "requestId": requestID, "routeId": route.RouteID}
	content, state, err := s.renderWithInline(r.Context(), template, body, global)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	state.Headers = route.Headers
	items, err := s.store.GetMiddlewares(route.MiddlewareIDs)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	result, err := jsmw.RunAll(items, state)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if result.Rejected {
		response.OK(w, []model.SendResult{{Code: -1, Message: result.Message, Rejected: true}})
		return
	}
	content, err = render.Template(template.Content, state.Ctx, state.Global)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	results := s.sendToTargets(r.Context(), requestID, route.RouteID, template.TemplateID, route.TargetURLs, state.Headers, route.Mode, template.MsgType, content, state.Ctx)
	response.OK(w, results)
}

func (s *Server) renderWithInline(ctx context.Context, template model.Template, data map[string]any, global map[string]any) (any, *jsmw.State, error) {
	state := &jsmw.State{Ctx: data, Global: global, Headers: map[string]string{}}
	code := firstNonEmpty(template.AsyncScript, template.Script)
	result, err := jsmw.RunInline(code, state)
	if err != nil {
		return nil, state, err
	}
	if result.Rejected {
		return nil, state, fmt.Errorf(result.Message)
	}
	content, err := render.Template(template.Content, state.Ctx, state.Global)
	return content, state, err
}

func (s *Server) sendToTargets(ctx context.Context, requestID, routeID, templateID string, targets []string, headers map[string]string, mode, msgType string, content any, messageContent map[string]any) []model.SendResult {
	results := make([]model.SendResult, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		result, requestBody := s.sender.Send(ctx, forward.Request{
			URL:            target,
			Headers:        headers,
			Mode:           mode,
			MsgType:        msgType,
			Content:        content,
			MessageContent: messageContent,
			RequestID:      requestID,
		})
		results = append(results, result)
		_ = s.store.CreateDelivery(model.Delivery{
			RequestID:    requestID,
			RouteID:      routeID,
			TemplateID:   templateID,
			TargetURL:    forward.RedactURL(target),
			StatusCode:   result.StatusCode,
			Success:      result.Code == 0,
			Message:      result.Message,
			RequestBody:  requestBody,
			ResponseBody: string(result.Response),
		})
	}
	return results
}

func (s *Server) handleGitlab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	body, ok := decodeBodyMap(w, r)
	if !ok {
		return
	}
	content := compatGitlabMarkdown(body)
	s.compatSend(w, r, "gitlab", body, content)
}

func (s *Server) handleSentry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	body, ok := decodeBodyMap(w, r)
	if !ok {
		return
	}
	content := compatSentryText(body)
	s.compatSend(w, r, "sentry", body, content)
}

func (s *Server) compatSend(w http.ResponseWriter, r *http.Request, source string, body map[string]any, content string) {
	targets := splitCSV(r.URL.Query().Get("webhookUrls"))
	routeID := r.URL.Query().Get("routeId")
	if routeID != "" {
		route, err := s.store.GetRoute(routeID)
		if err != nil {
			writeStoreResult(w, nil, err, http.StatusOK)
			return
		}
		targets = route.TargetURLs
	}
	if len(targets) == 0 {
		response.Error(w, http.StatusBadRequest, "webhookUrls or routeId is required")
		return
	}
	requestID := requestID(r)
	results := s.sendToTargets(r.Context(), requestID, routeID, source, targets, map[string]string{}, "envelope", "markdown", content, body)
	response.OK(w, results)
}

func (s *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	if s.cfg.AdminToken == "" {
		return true
	}
	if r.Header.Get("X-OpenHook-Admin-Token") == s.cfg.AdminToken {
		return true
	}
	response.Error(w, http.StatusUnauthorized, "invalid admin token")
	return false
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-OpenHook-Admin-Token, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 8<<20))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func decodeBodyMap(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return nil, false
	}
	data, err := render.DecodeObject(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return nil, false
	}
	return data, true
}

func writeStoreResult(w http.ResponseWriter, data any, err error, status int) {
	if err == nil {
		response.JSON(w, status, data)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "resource not found")
		return
	}
	response.Error(w, http.StatusBadRequest, err.Error())
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryObject(r *http.Request) map[string]any {
	result := map[string]any{}
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			items := make([]any, len(values))
			for i, value := range values {
				items[i] = value
			}
			result[key] = items
		}
	}
	return result
}

func requestID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req_fallback"
	}
	return "req_" + base64.RawURLEncoding.EncodeToString(buf[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func toColumn(value string) string {
	switch value {
	case "updateAt":
		return "update_at"
	case "templateName":
		return "template_name"
	default:
		return "create_at"
	}
}
