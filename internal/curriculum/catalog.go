package curriculum

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type ObjectiveManifest struct {
	Certification     string             `json:"certification"`
	SyllabusVersion   string             `json:"syllabus_version"`
	ExamCodes         []string           `json:"exam_codes"`
	ResearchDate      string             `json:"research_date"`
	CanonicalSource   string             `json:"canonical_scope_source"`
	RemovedObjectives []RemovedObjective `json:"removed_objectives"`
	Objectives        []Objective        `json:"objectives"`
}

type RemovedObjective struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Objective struct {
	ID                  string   `json:"id"`
	Exam                string   `json:"exam"`
	Topic               string   `json:"topic"`
	Weight              int      `json:"weight"`
	TitleFR             string   `json:"title_fr"`
	Active              bool     `json:"active"`
	SyllabusVersion     string   `json:"syllabus_version"`
	ExamCode            string   `json:"exam_code"`
	Concepts            []string `json:"concepts"`
	TermsFilesUtilities []string `json:"terms_files_utilities"`
	RecommendedBackend  string   `json:"recommended_backend"`
	AssessmentEvidence  []string `json:"assessment_evidence"`
	Notes               string   `json:"notes"`
	Source              string   `json:"source"`
}

type PrerequisiteGraph struct {
	SchemaVersion             string          `json:"schema_version"`
	Semantics                 GraphSemantics  `json:"semantics"`
	Certification             string          `json:"certification"`
	SyllabusVersion           string          `json:"syllabus_version"`
	RecommendedEntryObjective string          `json:"recommended_entry_objective"`
	Nodes                     []ObjectiveNode `json:"nodes"`
	ReferenceTopologicalOrder []string        `json:"reference_topological_order"`
}

type GraphSemantics struct {
	Hard        string `json:"hard"`
	Recommended string `json:"recommended"`
}

type ObjectiveNode struct {
	ObjectiveID              string   `json:"objective_id"`
	HardPrerequisites        []string `json:"hard_prerequisites"`
	RecommendedPrerequisites []string `json:"recommended_prerequisites"`
	RationaleFR              string   `json:"rationale_fr"`
}

type ConceptManifest struct {
	SchemaVersion   string    `json:"schema_version"`
	Certification   string    `json:"certification"`
	SyllabusVersion string    `json:"syllabus_version"`
	IDPolicy        string    `json:"id_policy"`
	Concepts        []Concept `json:"concepts"`
}

type Concept struct {
	ID                                  string   `json:"id"`
	ObjectiveID                         string   `json:"objective_id"`
	TitleFR                             string   `json:"title_fr"`
	PedagogyOrder                       int      `json:"pedagogy_order"`
	InheritedHardObjectivePrerequisites []string `json:"inherited_hard_objective_prerequisites"`
	Classification                      []string `json:"classification"`
	Active                              bool     `json:"active"`
	PrerequisiteConceptIDs              []string `json:"prerequisite_concept_ids,omitempty"`
}

type Phase1Slice struct {
	SchemaVersion      string              `json:"schema_version"`
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	SelectedObjectives []string            `json:"selected_objectives"`
	RationaleFR        string              `json:"rationale_fr"`
	ObjectiveConcepts  map[string][]string `json:"objective_concepts"`
	LearningOrder      SliceLearningOrder  `json:"learning_order"`
	ReferenceScenarios []SliceScenario     `json:"reference_scenarios"`
	ExplicitNonGoals   []string            `json:"explicit_non_goals"`
}

type SliceLearningOrder struct {
	Foundation     []string `json:"foundation"`
	ThenInterleave []string `json:"then_interleave"`
}

type SliceScenario struct {
	ID             string   `json:"id"`
	ObjectiveIDs   []string `json:"objective_ids"`
	GoalFR         string   `json:"goal_fr"`
	Backend        string   `json:"backend"`
	EvidenceTarget string   `json:"evidence_target"`
}

type Catalog struct {
	Certification             string
	SyllabusVersion           string
	Objectives                map[string]Objective
	Graph                     map[string]ObjectiveNode
	Concepts                  map[string]Concept
	ConceptsByObjective       map[string][]Concept
	ReferenceTopologicalOrder []string
	RecommendedEntryObjective string
	Phase1Slice               Phase1Slice
}

func LoadDir(dir string) (*Catalog, error) {
	var objectives ObjectiveManifest
	if err := decodeFile(filepath.Join(dir, "objectives.json"), &objectives); err != nil {
		return nil, fmt.Errorf("load objectives: %w", err)
	}
	var graph PrerequisiteGraph
	if err := decodeFile(filepath.Join(dir, "prerequisites.json"), &graph); err != nil {
		return nil, fmt.Errorf("load prerequisites: %w", err)
	}
	var concepts ConceptManifest
	if err := decodeFile(filepath.Join(dir, "concepts.json"), &concepts); err != nil {
		return nil, fmt.Errorf("load concepts: %w", err)
	}
	var phase1 Phase1Slice
	if err := decodeFile(filepath.Join(dir, "phase1-slice.json"), &phase1); err != nil {
		return nil, fmt.Errorf("load phase1 slice: %w", err)
	}

	if objectives.Certification != graph.Certification || objectives.Certification != concepts.Certification {
		return nil, errors.New("curriculum files disagree on certification")
	}
	if objectives.SyllabusVersion != graph.SyllabusVersion || objectives.SyllabusVersion != concepts.SyllabusVersion {
		return nil, errors.New("curriculum files disagree on syllabus version")
	}

	catalog := &Catalog{
		Certification:             objectives.Certification,
		SyllabusVersion:           objectives.SyllabusVersion,
		Objectives:                make(map[string]Objective),
		Graph:                     make(map[string]ObjectiveNode),
		Concepts:                  make(map[string]Concept),
		ConceptsByObjective:       make(map[string][]Concept),
		ReferenceTopologicalOrder: append([]string(nil), graph.ReferenceTopologicalOrder...),
		RecommendedEntryObjective: graph.RecommendedEntryObjective,
		Phase1Slice:               phase1,
	}
	for _, objective := range objectives.Objectives {
		if !objective.Active {
			continue
		}
		if _, exists := catalog.Objectives[objective.ID]; exists {
			return nil, fmt.Errorf("duplicate active objective %s", objective.ID)
		}
		catalog.Objectives[objective.ID] = objective
	}
	for _, node := range graph.Nodes {
		if _, exists := catalog.Graph[node.ObjectiveID]; exists {
			return nil, fmt.Errorf("duplicate graph node %s", node.ObjectiveID)
		}
		catalog.Graph[node.ObjectiveID] = node
	}
	for _, concept := range concepts.Concepts {
		if !concept.Active {
			continue
		}
		if _, exists := catalog.Concepts[concept.ID]; exists {
			return nil, fmt.Errorf("duplicate active concept %s", concept.ID)
		}
		catalog.Concepts[concept.ID] = concept
		catalog.ConceptsByObjective[concept.ObjectiveID] = append(catalog.ConceptsByObjective[concept.ObjectiveID], concept)
	}
	for objectiveID := range catalog.ConceptsByObjective {
		sort.Slice(catalog.ConceptsByObjective[objectiveID], func(i, j int) bool {
			return catalog.ConceptsByObjective[objectiveID][i].PedagogyOrder < catalog.ConceptsByObjective[objectiveID][j].PedagogyOrder
		})
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	return catalog, nil
}

func decodeFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

func (c *Catalog) Validate() error {
	if c.Certification == "" || c.SyllabusVersion == "" {
		return errors.New("curriculum identity is incomplete")
	}
	if len(c.Objectives) == 0 {
		return errors.New("no active objectives")
	}
	if _, ok := c.Objectives[c.RecommendedEntryObjective]; !ok {
		return fmt.Errorf("recommended entry objective %s is unknown", c.RecommendedEntryObjective)
	}
	if len(c.Graph) != len(c.Objectives) {
		return fmt.Errorf("graph/objective cardinality mismatch: graph=%d objectives=%d", len(c.Graph), len(c.Objectives))
	}
	for id, objective := range c.Objectives {
		if objective.ID == "" || objective.Weight <= 0 {
			return fmt.Errorf("objective %s has invalid identity or weight", id)
		}
		node, ok := c.Graph[id]
		if !ok {
			return fmt.Errorf("objective %s missing graph node", id)
		}
		overlap := make(map[string]struct{})
		for _, dep := range node.HardPrerequisites {
			overlap[dep] = struct{}{}
		}
		for _, dep := range node.RecommendedPrerequisites {
			if _, exists := overlap[dep]; exists {
				return fmt.Errorf("objective %s lists %s as hard and recommended", id, dep)
			}
		}
		for _, dep := range append(append([]string(nil), node.HardPrerequisites...), node.RecommendedPrerequisites...) {
			if dep == id {
				return fmt.Errorf("objective %s depends on itself", id)
			}
			if _, ok := c.Objectives[dep]; !ok {
				return fmt.Errorf("objective %s references unknown prerequisite %s", id, dep)
			}
		}
	}
	if err := c.validateAcyclic(false); err != nil {
		return err
	}
	if err := c.validateAcyclic(true); err != nil {
		return err
	}
	if err := c.validateReferenceOrder(); err != nil {
		return err
	}
	for id, concept := range c.Concepts {
		if _, ok := c.Objectives[concept.ObjectiveID]; !ok {
			return fmt.Errorf("concept %s references unknown objective %s", id, concept.ObjectiveID)
		}
		if concept.PedagogyOrder <= 0 {
			return fmt.Errorf("concept %s has invalid pedagogy order", id)
		}
		for _, dep := range concept.PrerequisiteConceptIDs {
			if _, ok := c.Concepts[dep]; !ok {
				return fmt.Errorf("concept %s references unknown concept prerequisite %s", id, dep)
			}
		}
	}
	for objectiveID, objective := range c.Objectives {
		concepts := c.ConceptsByObjective[objectiveID]
		if len(concepts) == 0 {
			return fmt.Errorf("objective %s has no active concepts", objectiveID)
		}
		if len(concepts) != len(objective.Concepts) {
			return fmt.Errorf("objective %s concept count drift: objective=%d manifest=%d", objectiveID, len(objective.Concepts), len(concepts))
		}
		for i, concept := range concepts {
			if concept.PedagogyOrder != i+1 {
				return fmt.Errorf("objective %s concept %s has non-contiguous pedagogy order %d", objectiveID, concept.ID, concept.PedagogyOrder)
			}
			if concept.TitleFR != objective.Concepts[i] {
				return fmt.Errorf("objective %s concept wording/order drift at %s", objectiveID, concept.ID)
			}
			if !sameStrings(concept.InheritedHardObjectivePrerequisites, c.Graph[objectiveID].HardPrerequisites) {
				return fmt.Errorf("concept %s inherited hard prerequisites drift", concept.ID)
			}
		}
	}
	if err := c.validatePhase1Slice(); err != nil {
		return err
	}
	return nil
}

func (c *Catalog) validateAcyclic(includeRecommended bool) error {
	indegree := make(map[string]int, len(c.Objectives))
	reverse := make(map[string][]string, len(c.Objectives))
	for id := range c.Objectives {
		deps := make(map[string]struct{})
		for _, dep := range c.Graph[id].HardPrerequisites {
			deps[dep] = struct{}{}
		}
		if includeRecommended {
			for _, dep := range c.Graph[id].RecommendedPrerequisites {
				deps[dep] = struct{}{}
			}
		}
		indegree[id] = len(deps)
		for dep := range deps {
			reverse[dep] = append(reverse[dep], id)
		}
	}
	queue := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	seen := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		seen++
		for _, next := range reverse[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if seen != len(c.Objectives) {
		kind := "hard"
		if includeRecommended {
			kind = "hard+recommended"
		}
		return fmt.Errorf("%s prerequisite graph contains a cycle", kind)
	}
	return nil
}

func (c *Catalog) validateReferenceOrder() error {
	if len(c.ReferenceTopologicalOrder) != len(c.Objectives) {
		return fmt.Errorf("reference topological order has %d entries, want %d", len(c.ReferenceTopologicalOrder), len(c.Objectives))
	}
	positions := make(map[string]int, len(c.ReferenceTopologicalOrder))
	for i, id := range c.ReferenceTopologicalOrder {
		if _, ok := c.Objectives[id]; !ok {
			return fmt.Errorf("reference topological order contains unknown objective %s", id)
		}
		if _, exists := positions[id]; exists {
			return fmt.Errorf("reference topological order contains duplicate objective %s", id)
		}
		positions[id] = i
	}
	for id, node := range c.Graph {
		for _, dep := range node.HardPrerequisites {
			if positions[dep] >= positions[id] {
				return fmt.Errorf("reference topological order violates %s -> %s", dep, id)
			}
		}
	}
	return nil
}

func (c *Catalog) validatePhase1Slice() error {
	if c.Phase1Slice.ID == "" || len(c.Phase1Slice.SelectedObjectives) == 0 {
		return errors.New("phase1 slice is incomplete")
	}
	selected := make(map[string]struct{}, len(c.Phase1Slice.SelectedObjectives))
	for _, id := range c.Phase1Slice.SelectedObjectives {
		if _, exists := selected[id]; exists {
			return fmt.Errorf("phase1 slice repeats objective %s", id)
		}
		if _, ok := c.Objectives[id]; !ok {
			return fmt.Errorf("phase1 slice references unknown objective %s", id)
		}
		selected[id] = struct{}{}
	}
	for objectiveID := range selected {
		for _, dep := range c.Graph[objectiveID].HardPrerequisites {
			if _, ok := selected[dep]; !ok {
				return fmt.Errorf("phase1 slice is not closed over hard prerequisite %s -> %s", dep, objectiveID)
			}
		}
		expected := make(map[string]struct{}, len(c.ConceptsByObjective[objectiveID]))
		for _, concept := range c.ConceptsByObjective[objectiveID] {
			expected[concept.ID] = struct{}{}
		}
		actual := c.Phase1Slice.ObjectiveConcepts[objectiveID]
		if len(actual) != len(expected) {
			return fmt.Errorf("phase1 slice concept count drift for %s", objectiveID)
		}
		seen := make(map[string]struct{}, len(actual))
		for _, conceptID := range actual {
			if _, duplicate := seen[conceptID]; duplicate {
				return fmt.Errorf("phase1 slice repeats concept %s", conceptID)
			}
			seen[conceptID] = struct{}{}
			if _, ok := expected[conceptID]; !ok {
				return fmt.Errorf("phase1 slice references unexpected concept %s for %s", conceptID, objectiveID)
			}
		}
	}
	return nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (c *Catalog) ObjectiveIDs() []string {
	ids := make([]string, 0, len(c.Objectives))
	for id := range c.Objectives {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (c *Catalog) ScopeConcepts(objectiveIDs []string) ([]Concept, error) {
	var result []Concept
	for _, objectiveID := range objectiveIDs {
		if _, ok := c.Objectives[objectiveID]; !ok {
			return nil, fmt.Errorf("unknown objective %s", objectiveID)
		}
		result = append(result, c.ConceptsByObjective[objectiveID]...)
	}
	return result, nil
}
