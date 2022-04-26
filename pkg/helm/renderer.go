package helm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/kyma-incubator/kymactl/manifests"
)

// Helm reenderer is based on istio operator implementation: https://github.com/istio/istio/blob/master/operator/pkg/helm/renderer.go

const (

	// YAMLSeparator is a separator for multi-document YAML files.
	YAMLSeparator = "\n---\n"

	// NotesFileNameSuffix is the file name suffix for helm notes.
	// see https://helm.sh/docs/chart_template_guide/notes_files/
	NotesFileNameSuffix = ".txt"
)

// TemplateRenderer defines a helm template renderer interface.
type TemplateRenderer interface {
	// Run starts the renderer and should be called before using it.
	Run() error
	// RenderManifest renders the associated helm charts with the given values YAML string and returns the resulting
	// string.
	RenderManifest(values string) (string, error)
}

// Renderer is a helm template renderer for a fs.FS.
type Renderer struct {
	namespace     string
	componentName string
	chart         *chart.Chart
	started       bool
	files         fs.FS
	dir           string
}

// NewFileTemplateRenderer creates a TemplateRenderer with the given parameters and returns a pointer to it.
// helmChartDirPath must be an absolute file path to the root of the helm charts.
func NewGenericRenderer(files fs.FS, dir, componentName, namespace string) *Renderer {
	return &Renderer{
		namespace:     namespace,
		componentName: componentName,
		dir:           dir,
		files:         files,
	}
}

// Run implements the TemplateRenderer interface.
func (h *Renderer) Run() error {
	if err := h.loadChart(); err != nil {
		return err
	}

	h.started = true
	return nil
}

// RenderManifest renders the current helm templates with the current values and returns the resulting YAML manifest string.
func (h *Renderer) RenderManifest(values string) (string, error) {
	if !h.started {
		return "", fmt.Errorf("fileTemplateRenderer for %s not started in renderChart", h.componentName)
	}
	return renderChart(h.componentName, h.namespace, values, h.chart)
}

func GetFilesRecursive(f fs.FS, root string) ([]string, error) {
	res := []string{}
	err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		res = append(res, path)
		return nil
	})
	return res, err
}

// loadChart implements the TemplateRenderer interface.
func (h *Renderer) loadChart() error {
	fnames, err := GetFilesRecursive(h.files, h.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("component %q does not exist", h.componentName)
		}
		return fmt.Errorf("list files: %v", err)
	}
	var bfs []*loader.BufferedFile
	for _, fname := range fnames {
		b, err := fs.ReadFile(h.files, fname)
		if err != nil {
			return fmt.Errorf("read file: %v", err)
		}
		// Helm expects unix / separator, but on windows this will be \
		name := strings.ReplaceAll(stripPrefix(fname, h.dir), string(filepath.Separator), "/")
		bf := &loader.BufferedFile{
			Name: name,
			Data: b,
		}
		bfs = append(bfs, bf)
	}
	h.chart, err = loader.LoadFiles(bfs)
	if err != nil {
		return fmt.Errorf("load files: %v", err)
	}
	return nil
}

func builtinProfileToFilename(name string) string {
	return "profile-" + name + ".yaml"
}

func LoadValues(profileName string, chartsDir string) (string, error) {
	path := strings.Join([]string{chartsDir, builtinProfileToFilename(profileName)}, "/")
	by, err := fs.ReadFile(manifests.FS, path)
	if err != nil {
		return "", err
	}
	return string(by), nil
}

// stripPrefix removes the the given prefix from prefix.
func stripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}

func convertNestedToStringInterfaceMap(input map[interface{}]interface{}, dest map[string]interface{}) {
	for k, v := range input {
		if nested, ok := v.(map[interface{}]interface{}); ok {
			destMap := make(map[string]interface{})
			dest[k.(string)] = destMap
			convertNestedToStringInterfaceMap(nested, destMap)
		} else {
			dest[k.(string)] = v
		}
	}

}

// renderChart renders the given chart with the given values and returns the resulting YAML manifest string.
func renderChart(name, namespace, values string, chrt *chart.Chart) (string, error) {
	options := chartutil.ReleaseOptions{
		Name:      name,
		Namespace: namespace,
		IsInstall: true,
	}
	valuesMap := map[interface{}]interface{}{}
	if err := yaml.Unmarshal([]byte(values), &valuesMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal values: %v", err)
	}
	convertedMap := map[string]interface{}{}
	convertNestedToStringInterfaceMap(valuesMap, convertedMap)

	caps := *chartutil.DefaultCapabilities
	vals, err := chartutil.ToRenderValues(chrt, convertedMap, options, &caps)
	if err != nil {
		fmt.Printf("Error dupa1: %s", err)
		return "", err
	}

	files, err := engine.Render(chrt, vals)
	crdFiles := chrt.CRDObjects()
	if err != nil {
		return "", err
	}

	// Create sorted array of keys to iterate over, to stabilize the order of the rendered templates
	keys := make([]string, 0, len(files))
	for k := range files {
		if strings.HasSuffix(k, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i := 0; i < len(keys); i++ {
		f := files[keys[i]]
		// add yaml separator if the rendered file doesn't have one at the end
		f = strings.TrimSpace(f) + "\n"
		if !strings.HasSuffix(f, YAMLSeparator) {
			f += YAMLSeparator
		}
		_, err := sb.WriteString(f)
		if err != nil {
			return "", err
		}
	}

	// Sort crd files by name to ensure stable manifest output
	sort.Slice(crdFiles, func(i, j int) bool { return crdFiles[i].Name < crdFiles[j].Name })
	for _, crdFile := range crdFiles {
		f := string(crdFile.File.Data)
		// add yaml separator if the rendered file doesn't have one at the end
		f = strings.TrimSpace(f) + "\n"
		if !strings.HasSuffix(f, YAMLSeparator) {
			f += YAMLSeparator
		}
		_, err := sb.WriteString(f)
		if err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}
