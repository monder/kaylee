package spec

type Spec struct {
	Name               string `json:"name"`
	Replicas           int    `json:"replicas,omitempty"`
	MaxReplicasPerHost int    `json:"maxReplicasPerHost,omitempty"`
	Engine             string `json:"engine,omitempty"`

	EnvFiles []string `json:"envFiles,omitempty"`
	Env      []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"env,omitempty"`

	Volumes []struct {
		ID      string `json:"id,omitempty"`
		Driver  string `json:"driver,omitempty"`
		Source  string `json:"source,omitempty"`
		Path    string `json:"path"`
		Options string `json:"options,omitempty"`
	} `json:"volumes,omitempty"`

	Net string `json:"net,omitempty"`

	Apps []struct {
		Image string   `json:"image"`
		Cmd   string   `json:"cmd,omitempty"`
		Args  []string `json:"args,omitempty"`
	} `json:"apps"`

	Args []string `json:"args,omitempty"`

	Machine   []string `json:"machine,omitempty"`
	MachineID string   `json:"machineId,omitempty"`
	Global    bool     `json:"global,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}
