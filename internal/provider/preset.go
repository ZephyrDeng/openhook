package provider

import (
	"encoding/json"
	"fmt"

	"github.com/ZephyrDeng/openhook/internal/model"
)

type Field struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
}

type Preset struct {
	ID            string          `json:"id"`
	Provider      string          `json:"provider"`
	TemplateName  string          `json:"templateName"`
	DisplayName   string          `json:"displayName"`
	Description   string          `json:"description"`
	MsgType       string          `json:"msgType"`
	RouteMode     string          `json:"routeMode"`
	TargetURLHint string          `json:"targetUrlHint"`
	Content       string          `json:"content"`
	Script        string          `json:"script"`
	Simulation    json.RawMessage `json:"simulation"`
	Fields        []Field         `json:"fields"`
	DocsURL       string          `json:"docsUrl,omitempty"`
}

func All() []Preset {
	presets := allPresets()
	result := make([]Preset, len(presets))
	copy(result, presets)
	return result
}

func Find(id string) (Preset, bool) {
	for _, preset := range allPresets() {
		if preset.ID == id {
			return preset, true
		}
	}
	return Preset{}, false
}

func TemplateInput(id, createBy, owner string) (model.TemplateInput, error) {
	preset, ok := Find(id)
	if !ok {
		return model.TemplateInput{}, fmt.Errorf("provider preset not found")
	}
	input := preset.TemplateInput()
	input.CreateBy = createBy
	input.CurrentOwner = owner
	return input, nil
}

func (p Preset) TemplateInput() model.TemplateInput {
	return model.TemplateInput{
		TemplateName: p.TemplateName,
		Content:      p.Content,
		MsgType:      p.MsgType,
		Script:       p.Script,
		Simulation:   cloneRaw(p.Simulation),
	}
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func allPresets() []Preset {
	return []Preset{
		{
			ID:            "wecom-markdown",
			Provider:      "wecom",
			TemplateName:  "企微-机器人 Markdown",
			DisplayName:   "企微 Markdown",
			Description:   "企业微信群机器人 markdown 消息，适合告警、发布和巡检通知。",
			MsgType:       "markdown",
			RouteMode:     "raw",
			TargetURLHint: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...",
			Content:       `{"msgtype":"markdown","markdown":{"content":{{json data.text}}}}`,
			Script:        `const level = String(ctx.severity || "info").toLowerCase(); const color = level === "critical" || level === "error" ? "warning" : level === "warning" ? "comment" : "info"; ctx.text = "# " + (ctx.title || "OpenHook") + "\n\n- 级别: <font color=\"" + color + "\">" + (ctx.severity || "info") + "</font>\n- 服务: " + (ctx.service || "-") + "\n- 环境: " + (ctx.environment || "-") + "\n- 时间: " + (ctx.time || "") + "\n\n" + (ctx.description || ""); return true;`,
			Simulation:    json.RawMessage(`{"title":"OpenHook 告警","severity":"info","service":"openhook","environment":"prod","time":"2026-06-04 00:00:00","description":"企微机器人 Markdown 测试"}`),
			Fields: []Field{
				{Name: "title", Label: "标题", Type: "string", Required: true, Default: "OpenHook 告警"},
				{Name: "severity", Label: "级别", Type: "string", Required: false, Default: "info"},
				{Name: "service", Label: "服务", Type: "string", Required: false, Default: "openhook"},
				{Name: "environment", Label: "环境", Type: "string", Required: false, Default: "prod"},
				{Name: "time", Label: "时间", Type: "string", Required: false},
				{Name: "description", Label: "描述", Type: "string", Required: false},
			},
			DocsURL: "https://developer.work.weixin.qq.com/document/path/91770",
		},
		{
			ID:            "wecom-text",
			Provider:      "wecom",
			TemplateName:  "企微-机器人 Text",
			DisplayName:   "企微 Text",
			Description:   "企业微信群机器人 text 消息，支持 userid 和手机号 @ 成员。",
			MsgType:       "text",
			RouteMode:     "raw",
			TargetURLHint: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...",
			Content:       `{"msgtype":"text","text":{"content":{{json data.text}},"mentioned_list":{{json data.mentionedList}},"mentioned_mobile_list":{{json data.mentionedMobileList}}}}`,
			Script:        `ctx.text = (ctx.title || "OpenHook") + "\n级别: " + (ctx.severity || "info") + "\n服务: " + (ctx.service || "-") + "\n环境: " + (ctx.environment || "-") + "\n" + (ctx.description || ""); ctx.mentionedList = ctx.mentionedList || []; ctx.mentionedMobileList = ctx.mentionedMobileList || []; return true;`,
			Simulation:    json.RawMessage(`{"title":"OpenHook 告警","severity":"info","service":"openhook","environment":"prod","description":"企微机器人文本测试","mentionedList":[],"mentionedMobileList":[]}`),
			Fields: []Field{
				{Name: "title", Label: "标题", Type: "string", Required: true, Default: "OpenHook 告警"},
				{Name: "severity", Label: "级别", Type: "string", Required: false, Default: "info"},
				{Name: "service", Label: "服务", Type: "string", Required: false, Default: "openhook"},
				{Name: "environment", Label: "环境", Type: "string", Required: false, Default: "prod"},
				{Name: "description", Label: "描述", Type: "string", Required: false},
				{Name: "mentionedList", Label: "@用户 ID", Type: "string[]", Required: false, Description: "支持 @all"},
				{Name: "mentionedMobileList", Label: "@手机号", Type: "string[]", Required: false, Description: "支持 @all"},
			},
			DocsURL: "https://developer.work.weixin.qq.com/document/path/91770",
		},
		{
			ID:            "telegram-html",
			Provider:      "telegram",
			TemplateName:  "Telegram-sendMessage",
			DisplayName:   "Telegram HTML",
			Description:   "Telegram Bot API sendMessage，使用 HTML parse_mode 并自动转义用户输入。",
			MsgType:       "html",
			RouteMode:     "raw",
			TargetURLHint: "https://api.telegram.org/bot<TOKEN>/sendMessage",
			Content:       `{"chat_id":{{json data.chatId}},"text":{{json data.text}},"parse_mode":"HTML","link_preview_options":{"is_disabled":true}}`,
			Script:        `const esc = (v) => String(v ?? "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;"); ctx.text = "<b>" + esc(ctx.title || "OpenHook") + "</b>\n\nSeverity: " + esc(ctx.severity || "info") + "\nService: " + esc(ctx.service || "-") + "\nEnvironment: " + esc(ctx.environment || "-") + "\n\n" + esc(ctx.description || ""); return true;`,
			Simulation:    json.RawMessage(`{"chatId":"123456789","title":"OpenHook alert","severity":"info","service":"openhook","environment":"prod","description":"Telegram sendMessage HTML 测试"}`),
			Fields: []Field{
				{Name: "chatId", Label: "Chat ID", Type: "string", Required: true, Default: "123456789"},
				{Name: "title", Label: "Title", Type: "string", Required: true, Default: "OpenHook alert"},
				{Name: "severity", Label: "Severity", Type: "string", Required: false, Default: "info"},
				{Name: "service", Label: "Service", Type: "string", Required: false, Default: "openhook"},
				{Name: "environment", Label: "Environment", Type: "string", Required: false, Default: "prod"},
				{Name: "description", Label: "Description", Type: "string", Required: false},
			},
			DocsURL: "https://core.telegram.org/bots/api#sendmessage",
		},
		{
			ID:            "telegram-text",
			Provider:      "telegram",
			TemplateName:  "Telegram-sendMessage Text",
			DisplayName:   "Telegram Text",
			Description:   "Telegram Bot API sendMessage 纯文本消息，适合外部系统已经拼好文本的场景。",
			MsgType:       "text",
			RouteMode:     "raw",
			TargetURLHint: "https://api.telegram.org/bot<TOKEN>/sendMessage",
			Content:       `{"chat_id":{{json data.chatId}},"text":{{json data.text}},"link_preview_options":{"is_disabled":true}}`,
			Script:        `ctx.text = ctx.text || ((ctx.title || "OpenHook") + "\nSeverity: " + (ctx.severity || "info") + "\nService: " + (ctx.service || "-") + "\nEnvironment: " + (ctx.environment || "-") + "\n\n" + (ctx.description || "")); return true;`,
			Simulation:    json.RawMessage(`{"chatId":"123456789","title":"OpenHook alert","severity":"info","service":"openhook","environment":"prod","description":"Telegram sendMessage text 测试"}`),
			Fields: []Field{
				{Name: "chatId", Label: "Chat ID", Type: "string", Required: true, Default: "123456789"},
				{Name: "text", Label: "Text", Type: "string", Required: false},
				{Name: "title", Label: "Title", Type: "string", Required: false, Default: "OpenHook alert"},
				{Name: "severity", Label: "Severity", Type: "string", Required: false, Default: "info"},
				{Name: "service", Label: "Service", Type: "string", Required: false, Default: "openhook"},
				{Name: "environment", Label: "Environment", Type: "string", Required: false, Default: "prod"},
				{Name: "description", Label: "Description", Type: "string", Required: false},
			},
			DocsURL: "https://core.telegram.org/bots/api#sendmessage",
		},
	}
}
