package httpapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func compatGitlabMarkdown(body map[string]any) string {
	kind := stringValue(body, "object_kind")
	project := objectValue(body, "project")
	attrs := objectValue(body, "object_attributes")
	name := stringValue(project, "name")
	if name == "" {
		name = "GitLab"
	}
	switch kind {
	case "merge_request":
		return fmt.Sprintf("# %s Merge Request\n- Title: [%s](%s)\n- Source: %s\n- Target: %s\n- Action: %s",
			name, stringValue(attrs, "title"), stringValue(attrs, "url"), stringValue(attrs, "source_branch"), stringValue(attrs, "target_branch"), stringValue(attrs, "action"))
	case "pipeline":
		commit := objectValue(body, "commit")
		return fmt.Sprintf("# %s Pipeline\n- Commit: [%s](%s)\n- Status: %s\n- Finished at: %s",
			name, stringValue(commit, "title"), stringValue(commit, "url"), stringValue(attrs, "status"), stringValue(attrs, "finished_at"))
	case "note":
		return fmt.Sprintf("# %s Comment\n- Comment: [%s](%s)", name, stringValue(attrs, "note"), stringValue(attrs, "url"))
	case "issue":
		return fmt.Sprintf("# %s Issue\n- Title: [%s](%s)\n- Action: %s", name, stringValue(attrs, "title"), stringValue(attrs, "url"), stringValue(attrs, "action"))
	default:
		return fmt.Sprintf("# %s GitLab Event\n- Kind: %s", name, kind)
	}
}

func compatSentryText(body map[string]any) string {
	event := objectValue(body, "event")
	tags := tagsMap(event["tags"])
	message := stringValue(body, "message")
	metadata := objectValue(event, "metadata")
	if value := stringValue(metadata, "value"); value != "" {
		message = value
	}
	project := stringValue(body, "project_name")
	if project == "" {
		project = "Sentry"
	}
	lines := []string{
		fmt.Sprintf("# %s alert", strings.ToUpper(project)),
		field("message", message),
		field("level", stringValue(event, "level")),
		field("environment", stringValue(event, "environment")),
		field("url", firstNonEmpty(tags["url"], stringValue(body, "web_url"), stringValue(body, "url"))),
		field("event_id", stringValue(event, "event_id")),
		field("time", sentryTime(event["timestamp"])),
	}
	filtered := lines[:0]
	for _, line := range lines {
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

func objectValue(root map[string]any, key string) map[string]any {
	value, _ := root[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func stringValue(root map[string]any, key string) string {
	switch value := root[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func tagsMap(value any) map[string]string {
	result := map[string]string{}
	list, ok := value.([]any)
	if !ok {
		return result
	}
	for _, item := range list {
		pair, ok := item.([]any)
		if ok && len(pair) >= 2 {
			result[fmt.Sprint(pair[0])] = fmt.Sprint(pair[1])
		}
	}
	return result
}

func field(key, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("- %s: %s", key, value)
}

func sentryTime(value any) string {
	switch typed := value.(type) {
	case float64:
		return time.Unix(int64(typed), 0).Format(time.RFC3339)
	case int64:
		return time.Unix(typed, 0).Format(time.RFC3339)
	case interface{ String() string }:
		seconds, err := strconv.ParseFloat(typed.String(), 64)
		if err == nil {
			return time.Unix(int64(seconds), 0).Format(time.RFC3339)
		}
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}
