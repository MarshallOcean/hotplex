package server

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/hrygo/hotplex"
	"github.com/hrygo/hotplex/event"
)

type ExecutionController struct {
	engine hotplex.HotPlexClient
	logger *slog.Logger
}

func NewExecutionController(engine hotplex.HotPlexClient, logger *slog.Logger) *ExecutionController {
	return &ExecutionController{
		engine: engine,
		logger: logger,
	}
}

type ExecutionRequest struct {
	SessionID    string
	Prompt       string
	Instructions string
	WorkDir      string
	Timeout      time.Duration
}

func (c *ExecutionController) Execute(ctx context.Context, req ExecutionRequest, cb event.Callback) error {
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	workDir := req.WorkDir
	if workDir == "" {
		workDir = "/tmp/hotplex_sandbox"
	}
	workDir = filepath.Clean(workDir)

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 15 * time.Minute
	}

	taskCtx, taskCancel := context.WithTimeout(ctx, timeout)
	defer taskCancel()

	cfg := &hotplex.Config{
		SessionID:        sessionID,
		WorkDir:          workDir,
		TaskInstructions: req.Instructions,
	}

	c.logger.Info("Controller: starting engine execution", "session_id", sessionID)

	if err := c.engine.ValidateConfig(cfg); err != nil {
		c.logger.Error("Controller: config validation failed", "session_id", sessionID, "error", err)
		return err
	}

	err := c.engine.Execute(taskCtx, cfg, req.Prompt, cb)
	if err != nil {
		if taskCtx.Err() == nil {
			c.logger.Error("Controller: execution failed", "session_id", sessionID, "error", err)
		} else {
			c.logger.Info("Controller: execution cancelled or timed out", "session_id", sessionID, "reason", taskCtx.Err())
		}
		return err
	}

	c.logger.Info("Controller: execution completed successfully", "session_id", sessionID)
	return nil
}
