package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ZephyrDeng/openhook/internal/model"
)

type Sender struct {
	client *http.Client
}

type Request struct {
	URL            string
	Headers        map[string]string
	Mode           string
	MsgType        string
	Content        any
	MessageContent map[string]any
	RequestID      string
}

func New(timeout time.Duration) *Sender {
	return &Sender{client: &http.Client{Timeout: timeout}}
}

func (s *Sender) Send(ctx context.Context, req Request) (model.SendResult, []byte) {
	body, err := buildBody(req)
	if err != nil {
		return model.SendResult{TargetURL: req.URL, Code: -1, Message: err.Error()}, nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.URL, bytes.NewReader(body))
	if err != nil {
		return model.SendResult{TargetURL: req.URL, Code: -1, Message: err.Error()}, body
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "openhook/1.0")
	if req.RequestID != "" {
		httpReq.Header.Set("X-OpenHook-Request-ID", req.RequestID)
	}
	for key, value := range req.Headers {
		if key != "" {
			httpReq.Header.Set(key, value)
		}
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return model.SendResult{TargetURL: req.URL, Code: -1, Message: err.Error()}, body
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	result := model.SendResult{
		TargetURL:  req.URL,
		Code:       resp.StatusCode,
		StatusCode: resp.StatusCode,
		Message:    resp.Status,
		Response:   json.RawMessage(jsonOrString(respBody)),
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Code = 0
		result.Message = "ok"
	}
	return result, body
}

func buildBody(req Request) ([]byte, error) {
	if req.Mode == "raw" {
		return json.Marshal(req.Content)
	}
	payload := map[string]any{
		"msgType":        fallback(req.MsgType, "markdown"),
		"content":        req.Content,
		"messageContent": req.MessageContent,
		"timestamp":      time.Now().UnixMilli(),
	}
	if req.RequestID != "" {
		payload["requestId"] = req.RequestID
	}
	return json.Marshal(payload)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func jsonOrString(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("null")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil {
		return raw
	}
	encoded, _ := json.Marshal(string(raw))
	return encoded
}

func RedactURL(url string) string {
	if len(url) <= 16 {
		return url
	}
	return fmt.Sprintf("%s***%s", url[:8], url[len(url)-8:])
}
