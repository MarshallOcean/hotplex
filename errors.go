package hotplex

import "errors"

// Standard Sentinel Errors for the HotPlex SDK
var (
	// ErrDangerBlocked indicates that a command or input was intercepted and blocked by the WAF (Detector).
	ErrDangerBlocked = errors.New("danger event blocked: input matched forbidden patterns")

	// ErrInvalidConfig indicates that the provided configuration is invalid or missing required fields.
	ErrInvalidConfig = errors.New("invalid execution configuration")
)
