package model

import "encoding/json"

type Template struct {
	ID           int64           `json:"id"`
	TemplateID   string          `json:"templateId"`
	TemplateKey  string          `json:"templateKey,omitempty"`
	TemplateName string          `json:"templateName"`
	Content      string          `json:"content"`
	MsgType      string          `json:"msgType"`
	Script       string          `json:"script,omitempty"`
	AsyncScript  string          `json:"asyncScript,omitempty"`
	Simulation   json.RawMessage `json:"simulation,omitempty"`
	CreateBy     string          `json:"createBy,omitempty"`
	UpdateBy     string          `json:"updateBy,omitempty"`
	CurrentOwner string          `json:"currentOwner,omitempty"`
	CreateAt     int64           `json:"createAt"`
	UpdateAt     int64           `json:"updateAt"`
	CanEdit      bool            `json:"canEdit,omitempty"`
	CanDel       bool            `json:"canDel,omitempty"`
}

type TemplateInput struct {
	TemplateName string          `json:"templateName"`
	Content      string          `json:"content"`
	MsgType      string          `json:"msgType"`
	Script       string          `json:"script"`
	AsyncScript  string          `json:"asyncScript"`
	Simulation   json.RawMessage `json:"simulation"`
	CreateBy     string          `json:"createBy"`
}

type TokenStatus int

const (
	TokenExpired TokenStatus = 0
	TokenEnabled TokenStatus = 1
	TokenDeleted TokenStatus = 2
)

type Token struct {
	ID          int64       `json:"id"`
	Token       string      `json:"token"`
	Name        string      `json:"name"`
	TemplateIDs []string    `json:"templateIds"`
	IsCoverAll  bool        `json:"isCoverAll"`
	Remark      string      `json:"remark,omitempty"`
	ExpireAt    int64       `json:"expireAt,omitempty"`
	UserIDs     []string    `json:"userIds"`
	CreateBy    string      `json:"createBy,omitempty"`
	Status      TokenStatus `json:"status"`
	CreateAt    int64       `json:"createAt"`
	UpdateAt    int64       `json:"updateAt"`
}

type TokenInput struct {
	Name        string   `json:"name"`
	TemplateIDs []string `json:"templateIds"`
	IsCoverAll  bool     `json:"isCoverAll"`
	Remark      string   `json:"remark"`
	ExpireAt    int64    `json:"expireAt"`
	UserIDs     []string `json:"userIds"`
	CreateBy    string   `json:"createBy"`
}

type Route struct {
	ID            int64             `json:"id"`
	RouteID       string            `json:"routeId"`
	Name          string            `json:"name"`
	TemplateID    string            `json:"templateId"`
	TargetURLs    []string          `json:"targetUrls"`
	Headers       map[string]string `json:"headers,omitempty"`
	MiddlewareIDs []string          `json:"middlewareIds,omitempty"`
	Mode          string            `json:"mode"`
	Enabled       bool              `json:"enabled"`
	CreateAt      int64             `json:"createAt"`
	UpdateAt      int64             `json:"updateAt"`
}

type RouteInput struct {
	Name          string            `json:"name"`
	TemplateID    string            `json:"templateId"`
	TargetURLs    []string          `json:"targetUrls"`
	Headers       map[string]string `json:"headers"`
	MiddlewareIDs []string          `json:"middlewareIds"`
	Mode          string            `json:"mode"`
	Enabled       *bool             `json:"enabled"`
}

type CustomMiddleware struct {
	ID           int64  `json:"id"`
	MiddlewareID string `json:"middlewareId"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Enabled      bool   `json:"enabled"`
	CreateAt     int64  `json:"createAt"`
	UpdateAt     int64  `json:"updateAt"`
}

type CustomMiddlewareInput struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Enabled *bool  `json:"enabled"`
}

type RuleSet struct {
	ID       int64           `json:"id"`
	Kind     string          `json:"kind"`
	Name     string          `json:"name"`
	Status   bool            `json:"status"`
	Domain   []string        `json:"domain"`
	Platform string          `json:"platform"`
	Payload  json.RawMessage `json:"payload"`
	CreateBy string          `json:"createBy,omitempty"`
	UpdateBy string          `json:"updateBy,omitempty"`
	CreateAt int64           `json:"createAt"`
	UpdateAt int64           `json:"updateAt"`
}

type RuleSetInput struct {
	Name     string          `json:"name"`
	Status   bool            `json:"status"`
	Domain   []string        `json:"domain"`
	Platform string          `json:"platform"`
	Payload  json.RawMessage `json:"payload"`
	CreateBy string          `json:"createBy"`
}

type Delivery struct {
	ID           int64           `json:"id"`
	RequestID    string          `json:"requestId"`
	RouteID      string          `json:"routeId,omitempty"`
	TemplateID   string          `json:"templateId,omitempty"`
	TargetURL    string          `json:"targetUrl"`
	StatusCode   int             `json:"statusCode"`
	Success      bool            `json:"success"`
	Message      string          `json:"message"`
	RequestBody  json.RawMessage `json:"requestBody,omitempty"`
	ResponseBody string          `json:"responseBody,omitempty"`
	CreateAt     int64           `json:"createAt"`
}

type SendResult struct {
	TargetURL  string          `json:"targetUrl"`
	Code       int             `json:"code"`
	Message    string          `json:"message"`
	Response   json.RawMessage `json:"response,omitempty"`
	Rejected   bool            `json:"rejected,omitempty"`
	StatusCode int             `json:"statusCode,omitempty"`
}
