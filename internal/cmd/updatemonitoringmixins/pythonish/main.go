package pythonish

import (
	"text/template"
)

func NewRenderer() *template.Template {
	return template.New("pythonish").Delims("%(", ")s")
}
