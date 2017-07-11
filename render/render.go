package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/handler"
	"io"
	"reflect"
	"text/template"
)

// Renderer contains templates to render object.
type Renderer struct {
	defaultTmpl *template.Template
	tmpl        *template.Template
}

func (r *Renderer) Run(dot interface{}, w io.Writer) error {
	tmpl := r.tmpl
	if tmpl == nil {
		tmpl = r.defaultTmpl
	}
	return tmpl.Execute(w, dot)
}

func (r *Renderer) AddTemplate(tmpl_name, tmpl_text string) error {
	tmpl := r.defaultTmpl.New(tmpl_name)
	if _, err := tmpl.Parse(tmpl_text); err != nil {
		return err
	} else {
		if tmpl_name == "" {
			r.tmpl = tmpl
		}
		return nil
	}
}

// Map type -> Renderer
var (
	renderers map[reflect.Type]*Renderer = make(map[reflect.Type]*Renderer)
)

func RegistRenderer(obj interface{}, tmpl_text string) {
	t := reflect.TypeOf(obj)
	if _, ok := renderers[t]; ok {
		panic(fmt.Errorf("RegistRenderer: %T has already registed", obj))
	}

	default_tmpl, err := template.New("default").Parse(tmpl_text)
	if err != nil {
		panic(err)
	}

	renderers[t] = &Renderer{
		defaultTmpl: default_tmpl,
	}

}

func GetRenderer(obj interface{}) *Renderer {
	renderer, ok := renderers[reflect.TypeOf(obj)]
	if ok {
		return renderer
	}
	return nil
}

// Main method to render an obj.
func Render(ctx *context.Context, obj interface{}, w io.Writer) error {
	h := handler.GetHandler(obj)
	if h == nil {
		return fmt.Errorf("Render: GetHandler return nil for %T", obj)
	}

	r := GetRenderer(obj)
	if r == nil {
		return fmt.Errorf("Render: GetRenderer return nil for %T", obj)
	}

	dot, err := h(ctx, obj)
	if err != nil {
		return err
	}

	return r.Run(dot, w)
}
