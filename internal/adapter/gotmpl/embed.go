package gotmpl

import "embed"

//go:embed templates/*.tmpl templates/domain/*.tmpl templates/app/*.tmpl templates/adapter/*.tmpl templates/cmd/*.tmpl
var templatesFS embed.FS
