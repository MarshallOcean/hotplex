package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// PermissionRequest represents a permission request from Claude Code.
// Format as described in GitHub Issue #39.
// Note: Claude Code has two permission request formats:
// 1. Legacy format with "permission" object: {"type":"permission_request","permission":{"name":"bash","input":"cmd"}}
// 2. Current format with "decision" object: {"type":"permission_request","decision":{"type":"ask","options":[...]}}
type PermissionRequest struct {
	Type       string            `json:"type"`
	SessionID  string            `json:"session_id,omitempty"`
	MessageID  string            `json:"message_id,omitempty"`
	Decision   *DecisionDetail   `json:"decision,omitempty"`
	Permission *PermissionDetail `json:"permission,omitempty"` // Legacy format
}

// PermissionDetail contains the permission details (legacy format).
// Used when Claude Code requests permission for a specific tool/action.
type PermissionDetail struct {
	Name  string `json:"name"`            // Tool name (e.g., "bash", "Read", "Edit")
	Input string `json:"input,omitempty"` // Tool input (e.g., command to execute)
}

// DecisionDetail contains the permission decision details.
type DecisionDetail struct {
	Type    string `json:"type"` // "ask", "allow", "deny"
	Reason  string `json:"reason,omitempty"`
	Options []struct {
		Name string `json:"name"`
	} `json:"options,omitempty"`
}

// PermissionResponse represents the response sent to Claude Code stdin.
// Format: {"behavior": "allow"} or {"behavior": "deny", "message": "User rejected"}
type PermissionResponse struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message,omitempty"`
}

// PermissionTool is the type of tool requesting permission.
type PermissionTool string

const (
	PermissionToolBash      PermissionTool = "Bash"
	PermissionToolRead      PermissionTool = "Read"
	PermissionToolEdit      PermissionTool = "Edit"
	PermissionToolWrite     PermissionTool = "Write"
	PermissionToolMultiEdit PermissionTool = "MultiEdit"
)

// PermissionBehavior is the user's decision for a permission request.
type PermissionBehavior string

const (
	PermissionBehaviorAllow       PermissionBehavior = "allow"
	PermissionBehaviorDeny        PermissionBehavior = "deny"
	PermissionBehaviorAllowAlways PermissionBehavior = "allow_always"
	PermissionBehaviorDenyAlways  PermissionBehavior = "deny_always"
)

// ParsePermissionRequest parses a permission request from JSON.
// It handles both legacy (permission) and current (decision) formats.
func ParsePermissionRequest(data []byte) (*PermissionRequest, error) {
	var req PermissionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse permission request: %w", err)
	}
	return &req, nil
}

// WritePermissionResponse writes a permission response to stdout/stdin.
// Format: single-line JSON with newline terminator.
func WritePermissionResponse(w io.Writer, behavior PermissionBehavior, message string) error {
	resp := PermissionResponse{
		Behavior: string(behavior),
		Message:  message,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Write single-line JSON with newline terminator
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// GetToolAndInput extracts the tool name and input from a permission request.
// Handles both legacy and current formats.
func (p *PermissionRequest) GetToolAndInput() (tool string, input string) {
	if p.Permission != nil {
		return p.Permission.Name, p.Permission.Input
	}

	// Current format: extract from decision if available
	if p.Decision != nil {
		return p.Decision.Type, p.Decision.Reason
	}

	return "", ""
}

// IsLegacy returns true if this is a legacy format permission request.
func (p *PermissionRequest) IsLegacy() bool {
	return p.Permission != nil
}

// GetDescription returns a human-readable description of the permission request.
func (p *PermissionRequest) GetDescription() string {
	tool, input := p.GetToolAndInput()
	if tool != "" && input != "" {
		return fmt.Sprintf("%s: %s", tool, input)
	}
	if p.Decision != nil && p.Decision.Reason != "" {
		return p.Decision.Reason
	}
	return "Permission request"
}

// PendingPermissionRequest tracks a pending permission request waiting for user response.
type PendingPermissionRequest struct {
	ID             string // Unique ID (messageID or generated)
	SessionID      string // Claude Code session ID
	Request        *PermissionRequest
	ChannelID      string           // Slack channel ID
	MessageTS      string           // Slack message timestamp for update
	UserID         string           // Slack user who triggered the request
	CreatedAt      time.Time        // When the request was created
	ExpiresAt      time.Time        // When the request expires (timeout)
	SlackMessageTS string           // TS of the Slack message with buttons
	Status         PermissionStatus // Current status
}

// PermissionStatus is the status of a pending permission request.
type PermissionStatus string

const (
	PermissionStatusPending  PermissionStatus = "pending"
	PermissionStatusAllowed  PermissionStatus = "allowed"
	PermissionStatusDenied   PermissionStatus = "denied"
	PermissionStatusExpired  PermissionStatus = "expired"
	PermissionStatusTimedOut PermissionStatus = "timed_out"
)

// IsExpired returns true if the pending request has expired.
func (p *PendingPermissionRequest) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// TimeUntilExpiry returns the duration until the request expires.
func (p *PendingPermissionRequest) TimeUntilExpiry() time.Duration {
	return time.Until(p.ExpiresAt)
}
