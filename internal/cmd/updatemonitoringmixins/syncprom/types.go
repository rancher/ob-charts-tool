package syncprom

type chartState struct {
	cwd      string
	mixinDir string
	rawText  string
	alerts   Alerts
	source   string
	url      string
}

type Alerts struct {
	Groups []AlertGroup `json:"groups"`
}

type AlertGroup struct {
	Interval string    `json:"interval,omitempty" yaml:"interval,omitempty"`
	Name     string    `json:"name" yaml:"name"`
	Rules    PromRules `json:"rules" yaml:"rules"`
}

type PromRules []PromRule

type PromRule struct {
	Alert       string            `json:"alert,omitempty" yaml:"alert,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Expr        string            `json:"expr"`
	For         string            `json:"for,omitempty" yaml:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Record      string            `json:"record,omitempty" yaml:"record,omitempty"`
}
