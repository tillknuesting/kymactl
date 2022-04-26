package helm

import (
	"testing"

	"github.com/kyma-incubator/kymactl/manifests"
)

func TestAllChartsCanBeRendered(t *testing.T) {

	r := NewGenericRenderer(manifests.FS, "charts/serverless", "serverless", "kyma-system")

	err := r.Run()
	if err != nil {
		t.Error(err)
	}
	evaluation, err := LoadValues("evaluation", "charts/serverless")
	if err != nil {
		t.Error(err)
	}
	t.Logf("Evaluation profile: %s", evaluation)

	evaluationManifest, err := r.RenderManifest(evaluation)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Evaluation manifest:\n%s", evaluationManifest)

}
