package planexecute

// PlanStep represents a single step in a plan (tool name and arguments).
type PlanStep struct {
	Tool string                 `json:"tool"`
	Args map[string]interface{} `json:"args"`
	// Reason is optional, for observability.
	Reason string `json:"reason,omitempty"`
}

// Plan is the result of the planning LLM call: an ordered list of tool steps.
type Plan struct {
	Steps []PlanStep `json:"steps"`
}
