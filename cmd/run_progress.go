package cmd

import (
	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
	"github.com/quiet-circles/hyperlocalise/internal/progressui"
)

func applyRunProgressEvent(renderer *progressui.Renderer, event runsvc.Event) {
	switch event.Kind {
	case runsvc.EventPhase:
		renderer.Phase(runPhaseMessage(event.Phase))
	case runsvc.EventPlanned:
		renderer.Plan(event.ExecutableTotal)
	case runsvc.EventTaskStart:
		renderer.TaskStarted(event.TargetPath, event.EntryKey)
	case runsvc.EventTaskDone:
		renderer.TaskStatus(event.TargetPath, event.EntryKey, event.TaskSucceeded, event.FailureReason)
		renderer.TaskDone(event.Succeeded, event.Failed, event.ExecutableTotal)
		renderer.TokenUsage(event.PromptTokens, event.CompletionTokens, event.TotalTokens)
	case runsvc.EventCompleted:
		renderer.TaskDone(event.Succeeded, event.Failed, event.ExecutableTotal)
		renderer.TokenUsage(event.PromptTokens, event.CompletionTokens, event.TotalTokens)
	}
}

func runPhaseMessage(phase string) string {
	switch phase {
	case runsvc.PhasePlanning:
		return "Planning tasks..."
	case runsvc.PhaseScanningPrune:
		return "Scanning prune candidates..."
	case runsvc.PhaseExecuting:
		return "Translating entries..."
	case runsvc.PhaseFinalizingOutput:
		return "Finalizing output..."
	default:
		return "Working..."
	}
}
