package mcpserve

import (
	"fmt"
	"os"
)

func getToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "clawide_notify",
			Description: "Send a notification to the ClawIDE dashboard. Use this to notify the user when a task is complete, needs attention, or encounters an error. The notification will appear in ClawIDE's notification bell and can deep-link back to this session.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Short notification title (e.g. 'Task Complete', 'Build Failed')",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Optional longer description with details about what happened",
					},
					"level": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"info", "success", "warning", "error"},
						"description": "Notification severity level. Defaults to 'info'",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source identifier (e.g. 'claude', 'build', 'test'). Defaults to 'claude'",
					},
				},
				Required: []string{"title"},
			},
		},
	}
}

func dispatchTool(name string, args map[string]interface{}, client *Client) (*toolCallResult, error) {
	switch name {
	case "clawide_notify":
		return handleNotify(args, client)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func handleNotify(args map[string]interface{}, client *Client) (*toolCallResult, error) {
	title, _ := args["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body, _ := args["body"].(string)
	level, _ := args["level"].(string)
	source, _ := args["source"].(string)

	if level == "" {
		level = "info"
	}
	if source == "" {
		source = "claude"
	}

	// Always send CWD so the server can resolve project + feature from the
	// worktree path. Only send explicit IDs for session/pane (which can't be
	// derived from the filesystem) and for project/feature when the env vars
	// are actually set.
	req := notificationRequest{
		Title:     title,
		Body:      body,
		Level:     level,
		Source:    source,
		SessionID: os.Getenv("CLAWIDE_SESSION_ID"),
		PaneID:    os.Getenv("CLAWIDE_PANE_ID"),
		CWD:       getCWD(),
	}
	// Only include project/feature IDs if explicitly set — otherwise let
	// the server resolve them from CWD (which also picks up the feature).
	if v := os.Getenv("CLAWIDE_FEATURE_ID"); v != "" {
		req.FeatureID = v
		req.ProjectID = os.Getenv("CLAWIDE_PROJECT_ID")
	} else if v := os.Getenv("CLAWIDE_PROJECT_ID"); v != "" {
		// Project ID set but no feature ID — omit project_id so the server
		// resolves both from CWD (which correctly identifies the feature).
		// This ensures feature worktrees are detected even when the terminal
		// session only has the project ID in its environment.
	}

	if err := client.PostNotification(req); err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	return &toolCallResult{
		Content: []contentBlock{
			{Type: "text", Text: fmt.Sprintf("Notification sent: %s", title)},
		},
	}, nil
}

func getCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}
