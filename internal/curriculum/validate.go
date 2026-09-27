package curriculum

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const expectedSchemaDraft = "https://json-schema.org/draft/2020-12/schema"

func Validate(bundle *Bundle) error {
	var errs []error

	active := make([]Objective, 0, len(bundle.Objectives.Objectives))
	objectiveByID := make(map[string]Objective)
	weights := map[string]int{"101": 0, "102": 0}

	for _, objective := range bundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		if _, exists := objectiveByID[objective.ID]; exists {
			errs = append(errs, fmt.Errorf("duplicate objective %s", objective.ID))
			continue
		}
		objectiveByID[objective.ID] = objective
		active = append(active, objective)
		weights[objective.Exam] += objective.Weight
	}

	if got := len(active); got != 42 {
		errs = append(errs, fmt.Errorf("expected 42 active objectives, got %d", got))
	}
	if weights["101"] != 60 {
		errs = append(errs, fmt.Errorf("exam 101 weight: expected 60, got %d", weights["101"]))
	}
	if weights["102"] != 60 {
		errs = append(errs, fmt.Errorf("exam 102 weight: expected 60, got %d", weights["102"]))
	}
	if _, exists := objectiveByID["104.4"]; exists {
		errs = append(errs, errors.New("104.4 must not be active in LPIC-1 v5"))
	}

	nodeByID := make(map[string]ObjectiveNode)
	for _, node := range bundle.Prerequisites.Nodes {
		if _, exists := nodeByID[node.ObjectiveID]; exists {
			errs = append(errs, fmt.Errorf("duplicate graph node %s", node.ObjectiveID))
			continue
		}
		nodeByID[node.ObjectiveID] = node
		if _, exists := objectiveByID[node.ObjectiveID]; !exists {
			errs = append(errs, fmt.Errorf("graph node %s has no active objective", node.ObjectiveID))
		}

		for _, dependency := range appendCopy(node.HardPrerequisites, node.RecommendedPrerequisites...) {
			if dependency == node.ObjectiveID {
				errs = append(errs, fmt.Errorf("%s depends on itself", node.ObjectiveID))
			}
			if _, exists := objectiveByID[dependency]; !exists {
				errs = append(errs, fmt.Errorf("%s references unknown prerequisite %s", node.ObjectiveID, dependency))
			}
		}
		for _, dependency := range node.HardPrerequisites {
			if slices.Contains(node.RecommendedPrerequisites, dependency) {
				errs = append(errs, fmt.Errorf("%s lists %s as hard and recommended", node.ObjectiveID, dependency))
			}
		}
	}

	if len(nodeByID) != len(objectiveByID) {
		errs = append(errs, fmt.Errorf("graph/objective cardinality mismatch: %d nodes, %d objectives", len(nodeByID), len(objectiveByID)))
	}
	for objectiveID := range objectiveByID {
		if _, exists := nodeByID[objectiveID]; !exists {
			errs = append(errs, fmt.Errorf("active objective %s is missing from graph", objectiveID))
		}
	}

	if err := validateAcyclic(bundle.Prerequisites.Nodes, false); err != nil {
		errs = append(errs, err)
	}
	if err := validateAcyclic(bundle.Prerequisites.Nodes, true); err != nil {
		errs = append(errs, err)
	}
	if err := validateReferenceOrder(bundle.Prerequisites.ReferenceTopologicalOrder, nodeByID); err != nil {
		errs = append(errs, err)
	}

	conceptByID := make(map[string]Concept)
	conceptsByObjective := make(map[string][]Concept)
	for _, concept := range bundle.Concepts.Concepts {
		if _, exists := conceptByID[concept.ID]; exists {
			errs = append(errs, fmt.Errorf("duplicate concept %s", concept.ID))
			continue
		}
		conceptByID[concept.ID] = concept
		if _, exists := objectiveByID[concept.ObjectiveID]; !exists {
			errs = append(errs, fmt.Errorf("concept %s references unknown objective %s", concept.ID, concept.ObjectiveID))
			continue
		}
		conceptsByObjective[concept.ObjectiveID] = append(conceptsByObjective[concept.ObjectiveID], concept)
		if !slices.Equal(concept.InheritedHardObjectivePrerequisites, nodeByID[concept.ObjectiveID].HardPrerequisites) {
			errs = append(errs, fmt.Errorf("concept %s inherited prerequisites drift", concept.ID))
		}
	}
	if got := len(conceptByID); got != 295 {
		errs = append(errs, fmt.Errorf("expected 295 concepts, got %d", got))
	}

	for _, objective := range active {
		concepts := conceptsByObjective[objective.ID]
		slices.SortFunc(concepts, func(a, b Concept) int {
			return a.PedagogyOrder - b.PedagogyOrder
		})
		if len(concepts) != len(objective.Concepts) {
			errs = append(errs, fmt.Errorf("%s concept count drift", objective.ID))
			continue
		}
		for index := range concepts {
			if concepts[index].TitleFR != objective.Concepts[index] {
				errs = append(errs, fmt.Errorf("%s concept order/title drift at position %d", objective.ID, index+1))
			}
		}
	}

	selected := make(map[string]struct{}, len(bundle.Phase1.SelectedObjectives))
	for _, objectiveID := range bundle.Phase1.SelectedObjectives {
		selected[objectiveID] = struct{}{}
		node, exists := nodeByID[objectiveID]
		if !exists {
			errs = append(errs, fmt.Errorf("phase1 references unknown objective %s", objectiveID))
			continue
		}
		for _, dependency := range node.HardPrerequisites {
			if _, included := selected[dependency]; !included && !slices.Contains(bundle.Phase1.SelectedObjectives, dependency) {
				errs = append(errs, fmt.Errorf("phase1 objective %s missing hard prerequisite %s", objectiveID, dependency))
			}
		}

		expected := conceptsByObjective[objectiveID]
		actualIDs := bundle.Phase1.ObjectiveConcepts[objectiveID]
		if len(actualIDs) != len(expected) {
			errs = append(errs, fmt.Errorf("phase1 concept count drift for %s", objectiveID))
			continue
		}
		for _, concept := range expected {
			if !slices.Contains(actualIDs, concept.ID) {
				errs = append(errs, fmt.Errorf("phase1 %s missing concept %s", objectiveID, concept.ID))
			}
		}
	}

	if len(bundle.Schemas) != 10 {
		errs = append(errs, fmt.Errorf("expected 10 schema files, got %d", len(bundle.Schemas)))
	}
	schemaIDs := make(map[string]string)
	for name, schema := range bundle.Schemas {
		draft, _ := schema["$schema"].(string)
		if draft != expectedSchemaDraft {
			errs = append(errs, fmt.Errorf("schema %s uses unsupported draft %q", name, draft))
		}
		id, _ := schema["$id"].(string)
		if id == "" {
			errs = append(errs, fmt.Errorf("schema %s has no $id", name))
		} else if previous, exists := schemaIDs[id]; exists {
			errs = append(errs, fmt.Errorf("schemas %s and %s share $id %s", previous, name, id))
		} else {
			schemaIDs[id] = name
		}
	}

	return errors.Join(errs...)
}

func validateAcyclic(nodes []ObjectiveNode, includeRecommended bool) error {
	dependencies := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		deps := append([]string(nil), node.HardPrerequisites...)
		if includeRecommended {
			deps = append(deps, node.RecommendedPrerequisites...)
		}
		dependencies[node.ObjectiveID] = deps
	}

	state := make(map[string]uint8, len(nodes))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("cycle detected at %s", id)
		case 2:
			return nil
		}
		state[id] = 1
		for _, dependency := range dependencies[id] {
			if _, exists := dependencies[dependency]; !exists {
				continue
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}

	for id := range dependencies {
		if err := visit(id); err != nil {
			kind := "hard"
			if includeRecommended {
				kind = "hard+recommended"
			}
			return fmt.Errorf("%s graph: %w", kind, err)
		}
	}
	return nil
}

func validateReferenceOrder(order []string, nodes map[string]ObjectiveNode) error {
	if len(order) != len(nodes) {
		return fmt.Errorf("reference order contains %d entries, expected %d", len(order), len(nodes))
	}
	position := make(map[string]int, len(order))
	for index, id := range order {
		if _, exists := position[id]; exists {
			return fmt.Errorf("reference order duplicates %s", id)
		}
		position[id] = index
	}
	for id, node := range nodes {
		current, exists := position[id]
		if !exists {
			return fmt.Errorf("reference order missing %s", id)
		}
		for _, dependency := range node.HardPrerequisites {
			depPosition, exists := position[dependency]
			if !exists {
				return fmt.Errorf("reference order missing prerequisite %s", dependency)
			}
			if depPosition >= current {
				return fmt.Errorf("reference order violates %s -> %s", dependency, id)
			}
		}
	}
	return nil
}

func appendCopy(base []string, extra ...string) []string {
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

func Summarize(bundle *Bundle) Summary {
	summary := Summary{
		Concepts:         len(bundle.Concepts.Concepts),
		Phase1Objectives: len(bundle.Phase1.SelectedObjectives),
		SchemaFiles:      len(bundle.Schemas),
	}
	for _, objective := range bundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		summary.ActiveObjectives++
		switch objective.Exam {
		case "101":
			summary.Exam101Weight += objective.Weight
		case "102":
			summary.Exam102Weight += objective.Weight
		}
	}
	for _, ids := range bundle.Phase1.ObjectiveConcepts {
		summary.Phase1Concepts += len(ids)
	}
	return summary
}

func FormatSummary(summary Summary) string {
	return strings.Join([]string{
		fmt.Sprintf("objectives=%d", summary.ActiveObjectives),
		fmt.Sprintf("exam101_weight=%d", summary.Exam101Weight),
		fmt.Sprintf("exam102_weight=%d", summary.Exam102Weight),
		fmt.Sprintf("concepts=%d", summary.Concepts),
		fmt.Sprintf("phase1_objectives=%d", summary.Phase1Objectives),
		fmt.Sprintf("phase1_concepts=%d", summary.Phase1Concepts),
		fmt.Sprintf("schemas=%d", summary.SchemaFiles),
	}, " ")
}
