package plan

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

// Verifier checks whether a tool is functional after a lifecycle action.
type Verifier interface {
	Check(ctx context.Context, tool catalog.Tool) (verify.VerifyResult, error)
}

// Summary describes the outcome of a plan execution.
type Summary struct {
	External    []string
	Failed      []FailedTool
	Installed   []string
	Skipped     []string
	Uninstalled []string
	Updated     []string
}

// FailedTool records a tool-level failure during plan execution.
type FailedTool struct {
	ToolID string
	Error  string
}

// ExecuteInstallPlan runs install actions and persists resulting receipts.
func ExecuteInstallPlan(ctx context.Context, plan Plan, st *state.State, verifier Verifier) (Summary, error) {
	summary := Summary{}

	for _, action := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return summary, fmt.Errorf("install plan canceled: %w", err)
		}

		switch action.Type {
		case ActionAlreadyInstalled:
			recordAlreadyInstalled(&summary, action)
		case ActionSkip:
			summary.Skipped = append(summary.Skipped, action.Tool.ID)
		case ActionInstall:
			if err := action.Manager.Install(ctx, action.Method); err != nil {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: err.Error()})

				continue
			}

			receipt := receiptForAction(action)
			if err := st.AddReceipt(receipt); err != nil {
				return summary, fmt.Errorf("persist install receipt for %s: %w", action.Tool.ID, err)
			}

			verifyResult, verifyErr := verifier.Check(ctx, action.Tool)
			updateReceiptVerification(st, action.Tool.ID, verifyResult)
			if verifyErr != nil {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: verifyErr.Error()})

				continue
			}

			if !verifyResult.Verified {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: "installed but verification failed"})

				continue
			}

			summary.Installed = append(summary.Installed, action.Tool.ID)
		}
	}

	return summary, nil
}

// ExecuteUpdatePlan runs update actions for managed tools.
func ExecuteUpdatePlan(ctx context.Context, plan Plan, st *state.State, verifier Verifier) (Summary, error) {
	summary := Summary{}

	for _, action := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return summary, fmt.Errorf("update plan canceled: %w", err)
		}

		switch action.Type {
		case ActionSkip:
			summary.Skipped = append(summary.Skipped, action.Tool.ID)
		case ActionUpdate:
			receipt, ok := st.Tool(action.Tool.ID)
			if !ok {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: "missing managed receipt"})

				continue
			}

			receipt.LastUpdateAttemptAt = time.Now().UTC()
			if err := st.AddReceipt(*receipt); err != nil {
				return summary, fmt.Errorf("persist update receipt for %s: %w", action.Tool.ID, err)
			}

			if err := action.Manager.Update(ctx, action.Method); err != nil {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: err.Error()})

				continue
			}

			verifyResult, verifyErr := verifier.Check(ctx, action.Tool)
			updateReceiptVerification(st, action.Tool.ID, verifyResult)
			if verifyErr != nil {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: verifyErr.Error()})

				continue
			}

			if !verifyResult.Verified {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: "updated but verification failed"})

				continue
			}

			summary.Updated = append(summary.Updated, action.Tool.ID)
		}
	}

	return summary, nil
}

// ExecuteUninstallPlan runs uninstall actions for managed tools.
func ExecuteUninstallPlan(ctx context.Context, plan Plan, st *state.State) (Summary, error) {
	summary := Summary{}

	for _, action := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return summary, fmt.Errorf("uninstall plan canceled: %w", err)
		}

		switch action.Type {
		case ActionSkip:
			if action.Reason == "tool is not managed by atb" {
				summary.External = append(summary.External, action.Tool.ID)

				continue
			}

			summary.Skipped = append(summary.Skipped, action.Tool.ID)
		case ActionUninstall:
			if err := action.Manager.Uninstall(ctx, action.Method); err != nil {
				summary.Failed = append(summary.Failed, FailedTool{ToolID: action.Tool.ID, Error: err.Error()})

				continue
			}

			st.Remove(action.Tool.ID)
			summary.Uninstalled = append(summary.Uninstalled, action.Tool.ID)
		}
	}

	return summary, nil
}

func recordAlreadyInstalled(summary *Summary, action Action) {
	if action.Reason == "already installed externally" {
		summary.External = append(summary.External, action.Tool.ID)

		return
	}

	summary.Skipped = append(summary.Skipped, action.Tool.ID)
}

func receiptForAction(action Action) state.ToolState {
	return state.ToolState{
		ToolID:           action.Tool.ID,
		Bin:              action.Tool.Bin,
		Ownership:        state.OwnershipManaged,
		InstallManager:   action.Method.Manager,
		InstallPackage:   action.Method.Package,
		InstallCommand:   slices.Clone(action.Method.Command),
		UpdateCommand:    slices.Clone(action.Method.UpdateCommand),
		UninstallCommand: slices.Clone(action.Method.UninstallCommand),
		InstalledAt:      time.Now().UTC(),
	}
}

func updateReceiptVerification(st *state.State, toolID string, result verify.VerifyResult) {
	receipt, ok := st.Tool(toolID)
	if !ok {
		return
	}

	receipt.LastVerifyAt = result.CheckedAt
	receipt.LastVerifyOK = result.Verified
	receipt.LastVerifyError = result.Error
	receipt.Version = result.Version
	if err := st.SetTool(*receipt); err != nil {
		return
	}
}
