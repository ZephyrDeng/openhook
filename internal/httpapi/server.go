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
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/ZephyrDeng/openhook/internal/config"
	"github.com/ZephyrDeng/openhook/internal/forward"
	jsmw "github.com/ZephyrDeng/openhook/internal/middleware"
	"github.com/ZephyrDeng/openhook/internal/model"
	"github.com/ZephyrDeng/openhook/internal/provider"
	"github.com/ZephyrDeng/openhook/internal/render"
	"github.com/ZephyrDeng/openhook/internal/response"
	"github.com/ZephyrDeng/openhook/internal/store"
)

type Server struct {
	store  *store.Store
	cfg    config.Config
	sender *forward.Sender
	static http.Handler
}

const sessionCookieName = "openhook_session"
const oauthStateCookieName = "openhook_oauth_state"
const oauthReturnCookieName = "openhook_oauth_return"
const defaultRepositoryURL = "https://github.com/ZephyrDeng/openhook"

type actor struct {
	admin bool
	user  model.User
}

func NewServer(st *store.Store, cfg config.Config, staticHandler http.Handler) http.Handler {
	server := &Server{
		store:  st,
		cfg:    cfg,
		sender: forward.New(cfg.RequestTimeout),
		static: staticHandler,
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
	if path == "health" {
		response.OK(w, map[string]any{"status": "ok"})
		return
	}
	if path == "" && s.static == nil {
		response.OK(w, map[string]any{"status": "ok"})
		return
	}

	switch {
	case len(parts) == 3 && parts[0] == "auth" && parts[1] == "github" && parts[2] == "start":
		s.handleGitHubStart(w, r)
	case len(parts) == 2 && (parts[0] == "login" || parts[0] == "register") && parts[1] == "github":
		s.handleGitHubStart(w, r)
	case len(parts) == 3 && parts[0] == "auth" && parts[1] == "github" && parts[2] == "callback":
		s.handleGitHubCallback(w, r)
	case len(parts) == 2 && parts[0] == "webhook" && parts[1] == "gitlab":
		s.handleGitlab(w, r)
	case len(parts) == 2 && parts[0] == "webhook" && parts[1] == "sentry":
		s.handleSentry(w, r)
	case len(parts) == 3 && parts[0] == "webhook" && parts[1] == "routes":
		s.handleRouteDeliver(w, r, parts[2])
	case len(parts) >= 1 && parts[0] == "api":
		s.routeAPI(w, r, parts[1:])
	case len(parts) == 2 && parts[0] == "webhook":
		s.handleTemplateWebhook(w, r, parts[1])
	default:
		if s.static != nil {
			s.static.ServeHTTP(w, r)
		} else {
			response.Error(w, http.StatusNotFound, "route not found")
		}
	}
}

func (s *Server) routeAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		response.OK(w, map[string]any{"name": "openhook"})
		return
	}
	switch parts[0] {
	case "meta":
		s.handleMeta(w, r)
	case "auth":
		s.handleAuthAPI(w, r, parts[1:])
	case "templates", "message-template":
		s.handleTemplates(w, r, parts[1:])
	case "providers":
		s.handleProviders(w, r, parts[1:])
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

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	version, commit, modified := buildVersion()
	response.OK(w, map[string]any{
		"name":       "OpenHook",
		"repository": firstNonEmpty(s.cfg.RepositoryURL, defaultRepositoryURL),
		"version":    version,
		"commit":     commit,
		"modified":   modified,
	})
}

func buildVersion() (string, string, bool) {
	version := "dev"
	commit := ""
	modified := false
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, modified
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if commit != "" {
		version = commit
		if len(version) > 7 {
			version = version[:7]
		}
		if modified {
			version += "+dirty"
		}
	}
	return version, commit, modified
}

func (s *Server) handleAuthAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 1 && parts[0] == "me" && r.Method == http.MethodGet {
		act, ok := s.actorFromRequest(r)
		if !ok {
			response.OK(w, map[string]any{
				"authenticated": false,
				"authRequired":  s.authEnabled(),
				"githubEnabled": s.githubEnabled(),
			})
			return
		}
		response.OK(w, map[string]any{
			"authenticated": true,
			"authRequired":  s.authEnabled(),
			"admin":         act.admin,
			"user":          act.user,
			"githubEnabled": s.githubEnabled(),
		})
		return
	}
	if len(parts) == 1 && parts[0] == "logout" && r.Method == http.MethodPost {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			_ = s.store.DeleteSession(cookie.Value)
		}
		http.SetCookie(w, s.expiredCookie(sessionCookieName))
		response.OK(w, map[string]any{"loggedOut": true})
		return
	}
	response.Error(w, http.StatusNotFound, "auth route not found")
}

func (s *Server) handleGitHubStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	if !s.githubEnabled() {
		response.Error(w, http.StatusServiceUnavailable, "github login is not configured")
		return
	}
	state := randomToken("oauth")
	returnTo := r.URL.Query().Get("returnTo")
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/"
	}
	http.SetCookie(w, s.cookie(oauthStateCookieName, state, 10*time.Minute))
	http.SetCookie(w, s.cookie(oauthReturnCookieName, returnTo, 10*time.Minute))

	authURL, err := url.Parse(s.cfg.GitHubAuthURL)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "invalid github auth url")
		return
	}
	query := authURL.Query()
	query.Set("client_id", s.cfg.GitHubClientID)
	query.Set("redirect_uri", strings.TrimRight(s.cfg.PublicBaseURL, "/")+"/auth/github/callback")
	query.Set("scope", "read:user")
	query.Set("state", state)
	authURL.RawQuery = query.Encode()
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "")
		return
	}
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		response.Error(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "missing oauth code")
		return
	}
	user, err := s.exchangeGitHubUser(r.Context(), code)
	if err != nil {
		response.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := s.ensureStarterTemplates(user); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	session, err := s.store.CreateSession(user.UserID, s.cfg.SessionTTL)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, s.cookie(sessionCookieName, session.Token, s.cfg.SessionTTL))
	http.SetCookie(w, s.expiredCookie(oauthStateCookieName))
	http.SetCookie(w, s.expiredCookie(oauthReturnCookieName))
	returnTo := "/"
	if cookie, err := r.Cookie(oauthReturnCookieName); err == nil && strings.HasPrefix(cookie.Value, "/") && !strings.HasPrefix(cookie.Value, "//") {
		returnTo = cookie.Value
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (s *Server) exchangeGitHubUser(ctx context.Context, code string) (model.User, error) {
	form := url.Values{}
	form.Set("client_id", s.cfg.GitHubClientID)
	form.Set("client_secret", s.cfg.GitHubClientSecret)
	form.Set("code", code)
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.GitHubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return model.User{}, err
	}
	tokenReq.Header.Set("Accept", "application/json")
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return model.User{}, err
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode < 200 || tokenResp.StatusCode >= 300 {
		return model.User{}, fmt.Errorf("github token exchange failed: %s", tokenResp.Status)
	}
	var tokenBody struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.NewDecoder(io.LimitReader(tokenResp.Body, 1<<20)).Decode(&tokenBody); err != nil {
		return model.User{}, err
	}
	if tokenBody.AccessToken == "" {
		if tokenBody.Error != "" {
			return model.User{}, fmt.Errorf("github token exchange failed: %s", firstNonEmpty(tokenBody.Description, tokenBody.Error))
		}
		return model.User{}, fmt.Errorf("github token exchange returned no access token")
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.GitHubUserURL, nil)
	if err != nil {
		return model.User{}, err
	}
	userReq.Header.Set("Accept", "application/json")
	userReq.Header.Set("Authorization", "Bearer "+tokenBody.AccessToken)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return model.User{}, err
	}
	defer userResp.Body.Close()
	if userResp.StatusCode < 200 || userResp.StatusCode >= 300 {
		return model.User{}, fmt.Errorf("github user fetch failed: %s", userResp.Status)
	}
	var userBody struct {
		ID        any    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(io.LimitReader(userResp.Body, 1<<20)).Decode(&userBody); err != nil {
		return model.User{}, err
	}
	if userBody.Login == "" {
		return model.User{}, fmt.Errorf("github user response missing login")
	}
	return s.store.UpsertUser(store.UserInput{
		Provider:   "github",
		ProviderID: fmt.Sprint(userBody.ID),
		Login:      userBody.Login,
		Name:       userBody.Name,
		AvatarURL:  userBody.AvatarURL,
	})
}

func (s *Server) ensureStarterTemplates(user model.User) error {
	if user.UserID == "" {
		return nil
	}
	_, total, err := s.store.ListTemplatesForOwner("", user.UserID, 1, 0, "create_at", "desc")
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}
	for _, input := range starterTemplates(user) {
		if _, err := s.store.CreateTemplate(input); err != nil {
			return err
		}
	}
	return nil
}

func starterTemplates(user model.User) []model.TemplateInput {
	owner := user.UserID
	createBy := userActorName(user)
	ids := []string{"wecom-markdown", "telegram-html"}
	items := make([]model.TemplateInput, 0, len(ids)+1)
	for _, id := range ids {
		input, err := provider.TemplateInput(id, createBy, owner)
		if err == nil {
			items = append(items, input)
		}
	}
	items = append(items, model.TemplateInput{
		TemplateName: "QQ-Webhook 文本",
		MsgType:      "markdown",
		Content:      "{\"msg_type\":\"text\",\"content\":{{json data.text}}}",
		Script:       `ctx.text = (ctx.title || "OpenHook") + "\n级别: " + (ctx.severity || "info") + "\n服务: " + (ctx.service || "-") + "\n环境: " + (ctx.environment || "-") + "\n" + (ctx.description || ""); return true;`,
		Simulation:   json.RawMessage(`{"title":"OpenHook alert","severity":"info","service":"openhook","environment":"prod","description":"QQ webhook bridge text payload"}`),
		CreateBy:     createBy,
		CurrentOwner: owner,
	})
	return items
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 && r.Method == http.MethodGet {
		if _, ok := s.authorizedActor(w, r); !ok {
			return
		}
		response.OK(w, provider.All())
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		if _, ok := s.authorizedActor(w, r); !ok {
			return
		}
		preset, ok := provider.Find(parts[0])
		if !ok {
			writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
			return
		}
		response.OK(w, preset)
		return
	}
	if len(parts) == 2 && parts[1] == "templates" && r.Method == http.MethodPost {
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		createBy := ""
		owner := ""
		if !act.admin {
			createBy = userActorName(act.user)
			owner = act.user.UserID
		}
		input, err := provider.TemplateInput(parts[0], createBy, owner)
		if err != nil {
			writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
			return
		}
		var override struct {
			TemplateName string `json:"templateName"`
			Visibility   string `json:"visibility"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			if !decodeJSON(w, r, &override) {
				return
			}
		}
		if override.TemplateName != "" {
			input.TemplateName = override.TemplateName
		}
		if override.Visibility != "" {
			input.Visibility = override.Visibility
		}
		item, err := s.store.CreateTemplate(input)
		writeStoreResult(w, item, err, http.StatusCreated)
		return
	}
	response.Error(w, http.StatusNotFound, "provider route not found")
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			act, ok := s.authorizedActor(w, r)
			if !ok {
				return
			}
			s.listTemplates(w, r, act)
		case http.MethodPost:
			act, ok := s.authorizedActor(w, r)
			if !ok {
				return
			}
			var input model.TemplateInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if !act.admin {
				input.CreateBy = userActorName(act.user)
				input.CurrentOwner = act.user.UserID
			}
			item, err := s.store.CreateTemplate(input)
			writeStoreResult(w, item, err, http.StatusCreated)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "")
		}
		return
	}
	if len(parts) == 1 && parts[0] == "preview" && r.Method == http.MethodPost {
		if _, ok := s.authorizedActor(w, r); !ok {
			return
		}
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
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		s.listTemplates(w, r, act)
		return
	}
	templateID := parts[0]
	if len(parts) == 2 && parts[1] == "render" && r.Method == http.MethodPost {
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		s.renderTemplateByID(w, r, templateID, act)
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
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		item, err := s.store.GetTemplate(templateID)
		if err == nil && !s.canUseTemplate(act, item) {
			err = store.ErrNotFound
		}
		if err == nil {
			item = s.templateForActor(act, item)
		}
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		if !s.canEditTemplateID(act, templateID) {
			writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
			return
		}
		var input model.TemplateInput
		if !decodeJSON(w, r, &input) {
			return
		}
		updateBy := input.CreateBy
		if !act.admin {
			updateBy = userActorName(act.user)
			input.CurrentOwner = act.user.UserID
		}
		item, err := s.store.UpdateTemplate(templateID, input, updateBy)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		if !s.canEditTemplateID(act, templateID) {
			writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
			return
		}
		err := s.store.DeleteTemplate(templateID)
		writeStoreResult(w, map[string]string{"templateId": templateID}, err, http.StatusOK)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request, act actor) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "size", 100)
	if page < 1 {
		page = 1
	}
	owner := ""
	if !act.admin {
		owner = act.user.UserID
	}
	items, total, err := s.store.ListTemplatesForOwner(r.URL.Query().Get("search"), owner, size, (page-1)*size, toColumn(r.URL.Query().Get("sortBy")), r.URL.Query().Get("sortOrder"))
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	for i := range items {
		items[i] = s.templateForActor(act, items[i])
	}
	if strings.Contains(r.URL.Path, "paginated") {
		response.OK(w, map[string]any{"items": items, "total": total, "page": page, "size": size})
		return
	}
	response.OK(w, items)
}

func (s *Server) renderTemplateByID(w http.ResponseWriter, r *http.Request, templateID string, act actor) {
	template, err := s.store.GetTemplate(templateID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	if !s.canUseTemplate(act, template) {
		writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
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
		if !s.authorizedAdmin(w, r) {
			return
		}
		items, err := s.store.ListTokens()
		writeStoreResult(w, items, err, http.StatusOK)
		return
	}
	if len(parts) == 1 && parts[0] == "create" && r.Method == http.MethodPost {
		if !s.authorizedAdmin(w, r) {
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
		if !s.authorizedAdmin(w, r) {
			return
		}
		item, err := s.store.GetToken(token)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPost:
		if !s.authorizedAdmin(w, r) {
			return
		}
		var input model.TokenInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateToken(token, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorizedAdmin(w, r) {
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
			act, ok := s.authorizedActor(w, r)
			if !ok {
				return
			}
			if act.admin {
				items, err := s.store.ListRoutes()
				writeStoreResult(w, items, err, http.StatusOK)
				return
			}
			items, err := s.store.ListRoutesForOwner(act.user.UserID)
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			act, ok := s.authorizedActor(w, r)
			if !ok {
				return
			}
			var input model.RouteInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if !act.admin {
				if !s.canUseTemplateID(act, input.TemplateID) {
					writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
					return
				}
				input.OwnerUserID = act.user.UserID
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
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		item, err := s.routeForActor(act, routeID)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		current, err := s.routeForActor(act, routeID)
		if err != nil {
			writeStoreResult(w, nil, err, http.StatusOK)
			return
		}
		var input model.RouteInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if !act.admin {
			templateID := input.TemplateID
			if templateID == "" {
				templateID = current.TemplateID
			}
			if !s.canUseTemplateID(act, templateID) {
				writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
				return
			}
			input.OwnerUserID = current.OwnerUserID
		}
		item, err := s.store.UpdateRoute(routeID, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		act, ok := s.authorizedActor(w, r)
		if !ok {
			return
		}
		if _, err := s.routeForActor(act, routeID); err != nil {
			writeStoreResult(w, nil, err, http.StatusOK)
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
			if !s.authorizedAdmin(w, r) {
				return
			}
			items, err := s.store.ListMiddlewares()
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			if !s.authorizedAdmin(w, r) {
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
		if !s.authorizedAdmin(w, r) {
			return
		}
		item, err := s.store.GetMiddleware(id)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodPut:
		if !s.authorizedAdmin(w, r) {
			return
		}
		var input model.CustomMiddlewareInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateMiddleware(id, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorizedAdmin(w, r) {
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
			if !s.authorizedAdmin(w, r) {
				return
			}
			items, err := s.store.ListRuleSets(kind, r.URL.Query())
			writeStoreResult(w, items, err, http.StatusOK)
		case http.MethodPost:
			if !s.authorizedAdmin(w, r) {
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
		if !s.authorizedAdmin(w, r) {
			return
		}
		item, err := s.store.GetActiveRuleSet(kind, r.URL.Query())
		writeStoreResult(w, item, err, http.StatusOK)
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodPut:
		if !s.authorizedAdmin(w, r) {
			return
		}
		var input model.RuleSetInput
		if !decodeJSON(w, r, &input) {
			return
		}
		item, err := s.store.UpdateRuleSet(kind, id, input)
		writeStoreResult(w, item, err, http.StatusOK)
	case http.MethodDelete:
		if !s.authorizedAdmin(w, r) {
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
	if !s.authorizedAdmin(w, r) {
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
	act, ok := s.authorizedActor(w, r)
	if !ok {
		return
	}
	template, err := s.store.GetTemplate(templateID)
	if err != nil {
		writeStoreResult(w, nil, err, http.StatusOK)
		return
	}
	if !s.canUseTemplate(act, template) {
		writeStoreResult(w, nil, store.ErrNotFound, http.StatusOK)
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
		if !route.Enabled {
			response.Error(w, http.StatusForbidden, "route is disabled")
			return
		}
		targets = route.TargetURLs
	} else if !s.authorizedAdmin(w, r) {
		return
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
	_, ok := s.authorizedActor(w, r)
	return ok
}

func (s *Server) authorizedAdmin(w http.ResponseWriter, r *http.Request) bool {
	act, ok := s.authorizedActor(w, r)
	if !ok {
		return false
	}
	if act.admin {
		return true
	}
	response.Error(w, http.StatusForbidden, "admin token required")
	return false
}

func (s *Server) authorizedActor(w http.ResponseWriter, r *http.Request) (actor, bool) {
	act, ok := s.actorFromRequest(r)
	if ok {
		return act, true
	}
	response.Error(w, http.StatusUnauthorized, "login required")
	return actor{}, false
}

func (s *Server) actorFromRequest(r *http.Request) (actor, bool) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if session, err := s.store.GetSession(cookie.Value); err == nil {
			return actor{user: session.User}, true
		}
	}
	if s.cfg.AdminToken != "" && r.Header.Get("X-OpenHook-Admin-Token") == s.cfg.AdminToken {
		return actor{admin: true}, true
	}
	if !s.authEnabled() {
		return actor{admin: true}, true
	}
	return actor{}, false
}

func (s *Server) authEnabled() bool {
	return s.cfg.AdminToken != "" || s.cfg.GitHubClientID != "" || s.cfg.GitHubClientSecret != ""
}

func (s *Server) githubEnabled() bool {
	return s.cfg.GitHubClientID != "" && s.cfg.GitHubClientSecret != ""
}

func (s *Server) canUseTemplateID(act actor, templateID string) bool {
	template, err := s.store.GetTemplate(templateID)
	return err == nil && s.canUseTemplate(act, template)
}

func (s *Server) canUseTemplate(act actor, template model.Template) bool {
	return act.admin || template.Visibility == "public" || (act.user.UserID != "" && template.CurrentOwner == act.user.UserID)
}

func (s *Server) canEditTemplateID(act actor, templateID string) bool {
	template, err := s.store.GetTemplate(templateID)
	return err == nil && s.canEditTemplate(act, template)
}

func (s *Server) canEditTemplate(act actor, template model.Template) bool {
	return act.admin || (act.user.UserID != "" && template.CurrentOwner == act.user.UserID)
}

func (s *Server) templateForActor(act actor, template model.Template) model.Template {
	template.CanEdit = s.canEditTemplate(act, template)
	template.CanDel = template.CanEdit
	return template
}

func (s *Server) routeForActor(act actor, routeID string) (model.Route, error) {
	if act.admin {
		return s.store.GetRoute(routeID)
	}
	return s.store.GetRouteForOwner(routeID, act.user.UserID)
}

func userActorName(user model.User) string {
	if user.Login != "" {
		return "github:" + user.Login
	}
	return user.UserID
}

func (s *Server) cookie(name, value string, ttl time.Duration) *http.Cookie {
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.PublicBaseURL, "https://"),
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	}
}

func (s *Server) expiredCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.PublicBaseURL, "https://"),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func randomToken(prefix string) string {
	var buf [24]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return newIDLike(prefix)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(buf[:])
}

func newIDLike(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 36)
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
