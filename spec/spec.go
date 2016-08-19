package spec

type Spec struct {
	Name               string
	Replicas           int    `json:"replicas,omitempty"`
	MaxReplicasPerHost int    `json:"maxReplicasPerHost,omitempty"`
	Engine             string `json:"engine,omitempty"`

	EnvFiles []string `json:"envFiles,omitempty"`
	Env      []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"env,omitempty"`

	Volumes []struct {
		ID      string `json:"id"`
		Driver  string `json:"driver"`
		Path    string `json:"path"`
		Options string `json:"options"`
	} `json:"volumes,omitempty"`

	Net string `json:"net,omitempty"`

	Apps []struct {
		Image string   `json:"image"`
		Args  []string `json:"args,omitempty"`
	}

	Args []string `json:"args,omitempty"`

	Machine   []string `json:"machine,omitempty"`
	MachineID string   `json:"machineId,omitempty"`
	Global    bool     `json:"global,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}
