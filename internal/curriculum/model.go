package curriculum

type ObjectivesFile struct {
	Certification        string             `json:"certification"`
	SyllabusVersion      string             `json:"syllabus_version"`
	ExamCodes            []string           `json:"exam_codes"`
	ResearchDate         string             `json:"research_date"`
	CanonicalScopeSource string             `json:"canonical_scope_source"`
	RemovedObjectives    []RemovedObjective `json:"removed_objectives"`
	Objectives           []Objective        `json:"objectives"`
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

type PrerequisitesFile struct {
	SchemaVersion             string          `json:"schema_version"`
	Certification             string          `json:"certification"`
	SyllabusVersion           string          `json:"syllabus_version"`
	Semantics                 GraphSemantics  `json:"semantics"`
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

type ConceptsFile struct {
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
	PrerequisiteConceptIDs              []string `json:"prerequisite_concept_ids,omitempty"`
	Classification                      []string `json:"classification"`
	Active                              bool     `json:"active"`
}

type Phase1Slice struct {
	SchemaVersion      string              `json:"schema_version"`
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	SelectedObjectives []string            `json:"selected_objectives"`
	RationaleFR        string              `json:"rationale_fr"`
	ObjectiveConcepts  map[string][]string `json:"objective_concepts"`
	LearningOrder      SliceLearningOrder  `json:"learning_order"`
	ReferenceScenarios []ReferenceScenario `json:"reference_scenarios"`
	ExplicitNonGoals   []string            `json:"explicit_non_goals"`
}

type Phase3Scope struct {
	SchemaVersion      string              `json:"schema_version"`
	ID                 string              `json:"id"`
	Certification      string              `json:"certification"`
	SyllabusVersion    string              `json:"syllabus_version"`
	Exam               string              `json:"exam"`
	ExamCode           string              `json:"exam_code"`
	Topics             []string            `json:"topics"`
	SelectedObjectives []string            `json:"selected_objectives"`
	ObjectiveConcepts  map[string][]string `json:"objective_concepts"`
	ConceptCount       int                 `json:"concept_count"`
	ScopeSource        string              `json:"scope_source"`
	ScopeVerifiedOn    string              `json:"scope_verified_on"`
}

type SliceLearningOrder struct {
	Foundation     []string `json:"foundation"`
	ThenInterleave []string `json:"then_interleave"`
}

type ReferenceScenario struct {
	ID             string   `json:"id"`
	ObjectiveIDs   []string `json:"objective_ids"`
	GoalFR         string   `json:"goal_fr"`
	Backend        string   `json:"backend"`
	EvidenceTarget string   `json:"evidence_target"`
}

type Bundle struct {
	Objectives    ObjectivesFile
	Prerequisites PrerequisitesFile
	Concepts      ConceptsFile
	Phase1        Phase1Slice
	Phase3        Phase3Scope
	Phase4        Phase3Scope
	Schemas       map[string]map[string]any
}

type Summary struct {
	ActiveObjectives int
	Exam101Weight    int
	Exam102Weight    int
	Concepts         int
	Phase1Objectives int
	Phase1Concepts   int
	Phase3Objectives int
	Phase3Concepts   int
	Phase4Objectives int
	Phase4Concepts   int
	SchemaFiles      int
}
