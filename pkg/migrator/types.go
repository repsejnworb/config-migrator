package migrator

type MigrationStep struct {
	Op       string                 `json:"op"`
	From     string                 `json:"from,omitempty"`
	To       string                 `json:"to,omitempty"`
	Path     string                 `json:"path,omitempty"`
	WrapAs   string                 `json:"wrapAs,omitempty"`
	UnwrapTo string                 `json:"unwrapTo,omitempty"`
	Rule     map[string]interface{} `json:"rule,omitempty"`
}

type Migration struct {
	From  string          `json:"from"`
	To    string          `json:"to"`
	Steps []MigrationStep `json:"steps"`
}
