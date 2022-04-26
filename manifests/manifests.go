package manifests

import (
	"embed"
)

// FS embeds the manifests.
// TODO: _helpers.tpl files are not taken without specific templates. See: https://github.com/golang/go/issues/43854 resolved in go 1.18.1
//
//go:embed charts/* crds components.yaml
//go:embed charts/*/*/_*
//go:embed charts/*/*/*/_*
//go:embed charts/*/*/*/*/_*
//go:embed charts/*/*/*/*/*/_*
//go:embed charts/*/*/*/*/*/*/_*
var FS embed.FS
