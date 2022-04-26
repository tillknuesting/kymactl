package helm

import (
	"io/fs"
	"testing"

	"github.com/kyma-incubator/kymactl/manifests"
	"gopkg.in/yaml.v2"
)

func TestAllChartsCanBeRendered(t *testing.T) {

	components := map[interface{}]interface{}{}
	by, err := fs.ReadFile(manifests.FS, "components.yaml")
	if err != nil {
		t.Errorf("Cannot read component.yaml: %s", err)
	}
	var list []interface{}
	yaml.Unmarshal(by, &components)
	for _, c := range components["components"].([]interface{}) {
		list = append(list, c)
	}
	for _, c := range components["prerequisites"].([]interface{}) {
		list = append(list, c)
	}

	for _, c := range list {
		var component = c.(map[interface{}]interface{})

		name := component["name"].(string)
		namespace := "kyma-system"
		if component["namespace"] != nil {
			namespace = component["namespace"].(string)
		}

		t.Logf("Component %s (%s)", name, namespace)
		r := NewGenericRenderer(manifests.FS, "charts/"+name, name, namespace)

		err = r.Run()
		if err != nil {
			t.Error(err)
		}
		evaluation, err := LoadValues("evaluation", "charts/"+name)
		if err != nil {
			t.Logf("No evaluation profile for %s", name)
		}

		_, err = r.RenderManifest(evaluation)
		if err != nil {
			t.Error(err)
		}

	}
}
