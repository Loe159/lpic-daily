package lab

type Definition struct {
	SchemaVersion        string            `json:"schema_version"`
	ID                   string            `json:"id"`
	TitleFR              string            `json:"title_fr"`
	BriefFR              string            `json:"brief_fr"`
	SuccessCriteriaFR    []string          `json:"success_criteria_fr"`
	DebriefFR            string            `json:"debrief_fr"`
	ObjectiveIDs         []string          `json:"objective_ids"`
	ConceptIDs           []string          `json:"concept_ids"`
	Labels               []string          `json:"labels"`
	EstimatedMinutes     int               `json:"estimated_minutes"`
	Environment          Environment       `json:"environment"`
	Resources            Resources         `json:"resources"`
	Setup                Setup             `json:"setup"`
	Checks               []CheckDefinition `json:"checks"`
	HintIDs              []string          `json:"hint_ids"`
	ReferenceSolutionRef string            `json:"reference_solution_ref"`
	ResetPolicy          string            `json:"reset_policy"`
}

type Environment struct {
	Backend            string   `json:"backend"`
	ImageRef           string   `json:"image_ref"`
	Distribution       string   `json:"distribution"`
	Network            string   `json:"network"`
	CapabilityProfile  string   `json:"capability_profile"`
	WritableGuestPaths []string `json:"writable_guest_paths"`
	Machine            *Machine `json:"machine,omitempty"`
}

type Machine struct {
	Firmware   string        `json:"firmware"`
	ExtraDisks []MachineDisk `json:"extra_disks,omitempty"`
}

type MachineDisk struct {
	ID     string `json:"id"`
	SizeMB int    `json:"size_mb"`
}

type Resources struct {
	MemoryMB       int `json:"memory_mb"`
	CPUPercent     int `json:"cpu_percent"`
	PIDs           int `json:"pids"`
	TimeoutSeconds int `json:"timeout_seconds"`
}

type Setup struct {
	ExecutionScope string `json:"execution_scope"`
	ScriptRef      string `json:"script_ref"`
}

type CheckDefinition struct {
	Type         string   `json:"type"`
	Path         string   `json:"path,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	User         string   `json:"user,omitempty"`
	Group        string   `json:"group,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	Match        string   `json:"match,omitempty"`
	Argv         []string `json:"argv,omitempty"`
	ExpectedExit   int      `json:"expected_exit,omitempty"`
	DeviceType     string   `json:"device_type,omitempty"`
	Filesystem     string   `json:"filesystem,omitempty"`
	PartitionTable string   `json:"partition_table,omitempty"`
	Mountpoint     string   `json:"mountpoint,omitempty"`
	SwapActive     *bool    `json:"swap_active,omitempty"`
}

type Hint struct {
	SchemaVersion  string `json:"schema_version"`
	ID             string `json:"id"`
	LabID          string `json:"lab_id"`
	Level          int    `json:"level"`
	ContentFR      string `json:"content_fr"`
	EvidenceImpact string `json:"evidence_impact"`
}

type Lab struct {
	Definition  Definition
	Hints       []Hint
	SetupScript string

	// ReferenceSolutionRef is retained for authoring/test traceability.
	// The referenced file is deliberately not loaded into runtime memory.
	ReferenceSolutionRef string
}
