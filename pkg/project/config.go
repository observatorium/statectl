package project

type Config struct {
	Configuration RepoConfig  `yaml:"configuration"`
	Overview      RepoConfig  `yaml:"overview"`
	State         StateConfig `yaml:"state"`
}

type StateType string

const (
	AppInterface StateType = "app-interface"
)

type RepoConfig struct {
	URL string `yaml:"URL"`
}

type StateConfig struct {
	URL string `yaml:"URL"`

	Type   StateType   `yaml:"type"`
	Config interface{} `yaml:"config"`
	Codec  StateCodec  `yaml:"-"`
}

type Cluster struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
}
