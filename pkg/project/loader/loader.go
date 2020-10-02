package loader

import (
	"github.com/observatorium/statectl/pkg/project"
	"github.com/observatorium/statectl/pkg/state/appinterface"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadProjectConfig loads project configuration from given bytes.
func LoadProjectConfig(file []byte) (*project.Config, error) {
	c := &project.Config{}
	if err := yaml.Unmarshal(file, c); err != nil {
		return nil, errors.Wrapf(err, "unmarshal config %v", file)
	}

	stateConfig, err := yaml.Marshal(c.State.Config)
	if err != nil {
		return nil, errors.Wrap(err, "marshal content of state configuration")
	}

	switch c.State.Type {
	case project.AppInterface:
		sc := appinterface.Config{}
		if err := yaml.Unmarshal(stateConfig, &sc); err != nil {
			return nil, errors.Wrapf(err, "unmarshal app interface config given by %v", file)
		}
		c.State.Codec = appinterface.NewSaaSCodec(sc)
	default:
		return nil, errors.Errorf("not supported state type %v, supported %v", c.State.Type, []project.StateType{project.AppInterface})
	}
	return c, nil
}
