// Package qwenacp binds the user's own Qwen Code installation to AO's reusable
// ACP Chat transport.
package qwenacp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/qwen"
	acpdriver "github.com/aoagents/agent-orchestrator/backend/internal/adapters/chatdriver/acp"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/chatdriver/nativeacp"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// New launches `qwen --acp` from the exact binary resolved by the existing Qwen
// Code agent plugin. Login, models, settings, and updates remain owned by the
// user's Qwen installation. `--experimental-acp` is the deprecated alias; AO
// launches the stable flag.
func New(plugin nativeacp.Plugin, log *slog.Logger) ports.ChatDriver {
	return newDriver(plugin, versionProbe, log)
}

func newDriver(plugin nativeacp.Plugin, probe nativeacp.VersionProbe, log *slog.Logger) ports.ChatDriver {
	return nativeacp.New(plugin, nativeacp.Config{
		Harness:        domain.HarnessQwen,
		Configure:      configure,
		SessionMode:    sessionMode,
		SessionOptions: sessionOptions,
		VersionProbe:   probe,
		Capabilities: ports.ChatCapabilities{
			// Qwen Code's ACP mode enforces approval modes over the wire: a
			// non-read-only shell under auto-edit emits
			// session/request_permission (verified live on 0.23.4), so the
			// permission selector carries a value the provider honors.
			// Bypass still maps to yolo for users who want no prompts.
			ports.ChatCapabilityApprovals: true,
		},
	}, log)
}

func configure(_ context.Context, cfg acpdriver.LaunchConfig) ([]string, map[string]string, error) {
	args := []string{"--acp"}
	qwen.AppendSessionFlags(&args, cfg.Permissions, cfg.Model)
	if prompt := strings.TrimSpace(cfg.SystemPrompt); prompt != "" {
		args = append(args, "--append-system-prompt", prompt)
	}
	return args, nil, nil
}

// sessionMode maps AO's permission vocabulary onto Qwen Code's advertised ACP
// mode ids (plan, default, auto-edit, auto, yolo). Qwen's out-of-the-box mode
// is auto, so AO's default explicitly selects default (Ask Permissions) to get
// supervised behavior instead of classifier auto-approval.
func sessionMode(permission ports.PermissionMode) string {
	switch ports.NormalizePermissionMode(permission) {
	case ports.PermissionModeDefault:
		return "default"
	case ports.PermissionModeAcceptEdits:
		return "auto-edit"
	case ports.PermissionModeAuto:
		return "auto"
	case ports.PermissionModeBypassPermissions:
		return "yolo"
	default:
		return ""
	}
}

func sessionOptions(settings ports.ChatTurnSettings) []acpdriver.SessionOption {
	if model := strings.TrimSpace(settings.Model); model != "" {
		return []acpdriver.SessionOption{{ID: "model", Value: model}}
	}
	return nil
}
