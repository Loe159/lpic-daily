package learning

import "github.com/Loe159/lpic-daily/internal/curriculum"

// EligibleObjectives returns objectives that may be introduced as new material.
//
// readiness represents prerequisite evidence, not lesson completion. This lets
// an initial assessment unlock known material without fabricating history.
func EligibleObjectives(graph curriculum.PrerequisitesFile, readiness map[string]bool) []string {
	eligible := make([]string, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if readiness[node.ObjectiveID] {
			continue
		}
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
