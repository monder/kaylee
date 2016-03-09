package lib

type Unit struct {
	Name string `json:"name"`
	Spec struct {
		Replicas           int    `json:"replicas"`
		MaxReplicasPerHost int    `json:"maxReplicasPerHost"`
		Image              string `json:"image"`
		Env                []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"env"`
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels"`
		Machine []string `json:"machine"`
	} `json:"spec"`
}
