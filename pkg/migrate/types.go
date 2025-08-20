package migrate

// Migration describes a version-to-version set of steps.
type Migration struct {
	Name  string          `json:"name,omitempty"`
	From  string          `json:"from"`
	To    string          `json:"to"`
	Steps []MigrationStep `json:"steps"`
}

// MigrationStep is a single operation.
type MigrationStep struct {
	Op         string                 `json:"op"` // move|wrap|unwrap|mapArray|set|delete (set/delete are non-reversible by default)
	From       string                 `json:"from,omitempty"`
	To         string                 `json:"to,omitempty"`
	Path       string                 `json:"path,omitempty"`
	WrapAs     string                 `json:"wrapAs,omitempty"`
	UnwrapTo   string                 `json:"unwrapTo,omitempty"`
	Rule       map[string]interface{} `json:"rule,omitempty"`       // for mapArray
	Reversible *bool                  `json:"reversible,omitempty"` // nil=>auto; false=>do not invert
}
