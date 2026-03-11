// Package verify checks whether a catalog tool is installed and functional.
package verify

import "time"

// VerifyResult captures the outcome of a tool verification attempt.
//
//nolint:revive // The implementation plan specifies the exported name VerifyResult.
type VerifyResult struct {
	ToolID    string
	Found     bool
	Verified  bool
	Version   string
	Error     string
	CheckedAt time.Time
}
