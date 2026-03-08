package runsvc

// Prune planning determines which keys should remain in each target file.

import (
	"fmt"
	"slices"
)

func buildPlannedTargetMetadata(planned []Task) (map[string]stagedOutput, error) {
	metadata := make(map[string]stagedOutput, len(planned))
	for _, task := range planned {
		existing, ok := metadata[task.TargetPath]
		if !ok {
			metadata[task.TargetPath] = stagedOutput{
				entries:      map[string]string{},
				sourcePath:   task.SourcePath,
				sourceLocale: task.SourceLocale,
				targetLocale: task.TargetLocale,
			}
			continue
		}
		if existing.sourcePath != task.SourcePath {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting source paths", task.TargetPath)
		}
		if existing.sourceLocale != "" && existing.sourceLocale != task.SourceLocale {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting source locales", task.TargetPath)
		}
		if existing.targetLocale != "" && existing.targetLocale != task.TargetLocale {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting target locales", task.TargetPath)
		}
	}
	return metadata, nil
}

func buildPlannedTargetKeySet(planned []Task) map[string]map[string]struct{} {
	keep := map[string]map[string]struct{}{}
	for _, task := range planned {
		bucket := keep[task.TargetPath]
		if bucket == nil {
			bucket = map[string]struct{}{}
			keep[task.TargetPath] = bucket
		}
		bucket[task.EntryKey] = struct{}{}
	}
	return keep
}

func (s *Service) planPruneCandidates(pruneTargets map[string]map[string]struct{}) ([]PruneCandidate, error) {
	candidates := make([]PruneCandidate, 0)
	targetPaths := make([]string, 0, len(pruneTargets))
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)

	for _, targetPath := range targetPaths {
		existing, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return nil, err
		}
		for _, key := range sortedEntryKeys(existing) {
			if _, ok := pruneTargets[targetPath][key]; !ok {
				candidates = append(candidates, PruneCandidate{TargetPath: targetPath, EntryKey: key})
			}
		}
	}
	return candidates, nil
}

func validatePruneLimit(in Input, candidates int) error {
	if !in.Prune || in.DryRun || in.PruneForce {
		return nil
	}
	limit := in.PruneLimit
	if limit <= 0 {
		limit = defaultPruneLimit
	}
	if candidates <= limit {
		return nil
	}
	return fmt.Errorf("prune safety limit exceeded: %d keys scheduled for deletion (limit %d). rerun with --prune-max-deletions %d or --prune-force", candidates, limit, candidates)
}
