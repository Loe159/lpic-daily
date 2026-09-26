package content

type SourceRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Lesson struct {
	SchemaVersion          string      `json:"schema_version"`
	ID                     string      `json:"id"`
	TitleFR                string      `json:"title_fr"`
	ObjectiveIDs           []string    `json:"objective_ids"`
	ConceptIDs             []string    `json:"concept_ids"`
	PrerequisiteConceptIDs []string    `json:"prerequisite_concept_ids,omitempty"`
	Stage                  string      `json:"stage"`
	EstimatedMinutes       int         `json:"estimated_minutes"`
	BodyMarkdown           string      `json:"body_markdown"`
	Labels                 []string    `json:"labels"`
	Distribution           string      `json:"distribution,omitempty"`
	SourceRefs             []SourceRef `json:"source_refs,omitempty"`
}

type Choice struct {
	ID      string `json:"id"`
	LabelFR string `json:"label_fr"`
}

type Grading struct {
	Strategy          string   `json:"strategy"`
	AcceptedAnswers   []string `json:"accepted_answers,omitempty"`
	CaseSensitive     bool     `json:"case_sensitive,omitempty"`
	Pattern           string   `json:"pattern,omitempty"`
	AcceptedChoiceIDs []string `json:"accepted_choice_ids,omitempty"`
	ExpectedOrder     []string `json:"expected_order,omitempty"`
}

type Question struct {
	SchemaVersion         string   `json:"schema_version"`
	ID                    string   `json:"id"`
	ObjectiveIDs          []string `json:"objective_ids"`
	ConceptIDs            []string `json:"concept_ids"`
	Type                  string   `json:"type"`
	PromptFR              string   `json:"prompt_fr"`
	Choices               []Choice `json:"choices,omitempty"`
	Grading               Grading  `json:"grading"`
	EvidenceKindOnSuccess string   `json:"evidence_kind_on_success"`
	Labels                []string `json:"labels"`
	Distribution          string   `json:"distribution,omitempty"`
	ExplanationFR         string   `json:"explanation_fr,omitempty"`
}

type Bundle struct {
	Lessons   []Lesson
	Questions []Question
}
