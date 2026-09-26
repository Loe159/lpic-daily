package learning

import "github.com/Loe159/lpic-daily/internal/curriculum"

// EligibleObjectives returns objectives whose hard prerequisites are ready.
//
// readiness describes prerequisite readiness only. A ready objective may still
// contain unseen or due concepts, so readiness must never be interpreted as
// "objective complete".
func EligibleObjectives(graph curriculum.PrerequisitesFile, readiness map[string]bool) []string {
	eligible := make([]string, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		ready := true
		for _, prerequisite := range node.HardPrerequisites {
			if !readiness[prerequisite] {
				ready = false
				break
			}
		}
		if ready {
			eligible = append(eligible, node.ObjectiveID)
		}
	}
	return eligible
}
