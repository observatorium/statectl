package loader

import (
	"testing"

	"github.com/observatorium/statectl/pkg/project"
	"github.com/observatorium/statectl/pkg/state/appinterface"
	"github.com/observatorium/statectl/pkg/testutil"
)

func TestLoadConfig(t *testing.T) {
	c, err := LoadProjectConfig([]byte(`
configuration:
  URL: "git@example.com:observatorium/configuration.git"
overview:
  URL: "git@example.com:observatorium/statectl-overview.git"

state:
  URL: "git@example.com:service/app-interface.git"
  type: "app-interface"
  config:
    saasFile: "data/services/telemeter/cicd/saas.yaml"
    clusters:
      /services/telemeter/namespaces/production.yml:
        name: "telemeter-prod-01"
        environment: "production"
      /services/telemeter/namespaces/stage.yml:
        name: "telemeter-stage-01"
        environment: "staging"
`))
	testutil.Ok(t, err)
	testutil.Equals(t, &project.Config{
		Configuration: project.RepoConfig{URL: "git@example.com:observatorium/configuration.git"},
		Overview:      project.RepoConfig{URL: "git@example.com:observatorium/statectl-overview.git"},
		State: project.StateConfig{
			URL:  "git@example.com:service/app-interface.git",
			Type: "app-interface",
			Config: map[string]interface{}{"clusters": map[string]interface{}{
				"/services/telemeter/namespaces/production.yml": map[string]interface{}{
					"environment": "production", "name": "telemeter-prod-01",
				},
				"/services/telemeter/namespaces/stage.yml": map[string]interface{}{
					"environment": "staging", "name": "telemeter-stage-01",
				},
			}, "saasFile": "data/services/telemeter/cicd/saas.yaml",
			}, Codec: appinterface.NewSaaSCodec(appinterface.Config{
				SaaSFile: "data/services/telemeter/cicd/saas.yaml",
				ClustersByRef: map[string]project.Cluster{
					"/services/telemeter/namespaces/production.yml": {Name: "telemeter-prod-01", Environment: "production"},
					"/services/telemeter/namespaces/stage.yml":      {Name: "telemeter-stage-01", Environment: "staging"},
				},
			}),
		},
	}, c)
}
