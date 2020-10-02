package appinterface

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/observatorium/statectl/pkg/project"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SaaSFile string `yaml:"saasFile"`

	ClustersByRef map[string]project.Cluster `yaml:"clusters"`
}

// To check compatibility with project.StateEditor interface.
var _ project.StateCodec = StateCodec{}

type StateCodec struct {
	cfg Config
}

func NewSaaSCodec(cfg Config) StateCodec {
	return StateCodec{cfg}
}

func (s StateCodec) Decode(dir string) ([]project.ServiceState, error) {
	saasPath := filepath.Join(dir, s.cfg.SaaSFile)
	b, err := ioutil.ReadFile(saasPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read %v", saasPath)
	}

	saas := saasFile{}
	if err := yaml.Unmarshal(b, &saas); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %v", saasPath)
	}

	var ret []project.ServiceState
	for _, tmpl := range saas.ResourceTemplates {
		for _, target := range tmpl.Targets {
			envParams := map[string]string{}
			for k, v := range tmpl.Parameters {
				envParams[k] = v
			}
			for k, v := range target.Parameters {
				envParams[k] = v
			}

			cl, ok := s.cfg.ClustersByRef[target.Namespace.Ref]
			if !ok {
				return nil, errors.Errorf("No cluster defined in configuration file for reference %v", target.Namespace.Ref)
			}

			ret = append(ret, project.ServiceState{
				Service:           tmpl.Name,
				ConfigurationPath: tmpl.Path,
				ConfigurationURL:  strings.TrimLeft(tmpl.URL, "https://"),
				Cluster:           cl,
				ConfigurationRef:  project.Ref(target.Ref),
				EnvParameters:     envParams,
			})
		}
	}
	return ret, nil
}

func (s StateCodec) Encode(dir string, states []project.ServiceState) error {
	return errors.New("not implemented")
}

// saasFile represents parsable form of part of SaaS.yaml we are interested in.
type saasFile struct {
	// TODO(bwplotka): Check schema changes from ( string `yaml:"$schema"`) so we can support multiple versions.
	ResourceTemplates []struct {
		Name       string            `yaml:"name"`
		Path       string            `yaml:"path"`
		URL        string            `yaml:"url"`
		Parameters map[string]string `yaml:"parameters,omitempty"`
		Targets    []struct {
			Namespace struct {
				Ref string `yaml:"$ref"`
			} `yaml:"namespace"`
			Ref        string            `yaml:"ref"`
			Parameters map[string]string `yaml:"parameters,omitempty"`
		} `yaml:"targets"`
	} `yaml:"resourceTemplates"`
}
