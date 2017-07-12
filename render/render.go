package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/handler"
	"io"
	"reflect"
	"text/template"
)

var (
	default_type_templates map[reflect.Type]string = make(map[reflect.Type]string)
)

// Regist default template for a type.
func RegistDefaultTypeTemplate(obj interface{}, text string) {
	t := reflect.TypeOf(obj)
	if _, ok := default_type_templates[t]; ok {
		panic(fmt.Errorf("RegistTypeTemplate: %T has already registed", obj))
	}

	default_type_templates[t] = text
}

// RenderInfo contains templates to render objects.
type RenderInfo struct {
	// map type -> []*template.Template
	// tmpls[0] is the default template
	// tmpls[-1] is the template used for renderring
	tmpls map[reflect.Type][]*template.Template

	// Extra functions used in templates.
	extraFuncs template.FuncMap

	ctx *context.Context
}

func (r *RenderInfo) defaultTmpl(obj interface{}) *template.Template {
	tmpls, ok := r.tmpls[reflect.TypeOf(obj)]
	if !ok {
		return nil
	}
	return tmpls[0]
}

func (r *RenderInfo) currentTmpl(obj interface{}) *template.Template {
	tmpls, ok := r.tmpls[reflect.TypeOf(obj)]
	if !ok {
		return nil
	}
	return tmpls[len(tmpls)-1]
}

func NewRenderInfo(ctx *context.Context) *RenderInfo {
	ret := &RenderInfo{
		tmpls:      make(map[reflect.Type][]*template.Template),
		extraFuncs: buildExtraFuncs(ctx),
		ctx:        ctx,
	}
	for t, text := range default_type_templates {
		tmpl := template.New("default").Funcs(ret.extraFuncs)
		if _, err := tmpl.Parse(text); err != nil {
			panic(err)
		}
		ret.tmpls[t] = []*template.Template{
			tmpl,
		}
	}
	return ret
}

// Add a new template to the RenderInfo. The lastest added
// template will be used for renderring.
func (r *RenderInfo) AddTemplate(obj interface{}, name string, text string) error {
	default_tmpl := r.defaultTmpl(obj)
	if default_tmpl == nil {
		return fmt.Errorf("AddTemplate: no render info for %T", obj)
	}

	tmpl := default_tmpl.New(name).Funcs(r.extraFuncs)
	if _, err := tmpl.Parse(text); err != nil {
		return err
	}

	t := reflect.TypeOf(obj)
	r.tmpls[t] = append(r.tmpls[t], tmpl)
	return nil
}

// Render.
func (r *RenderInfo) Run(obj interface{}, w io.Writer) error {
	tmpl := r.currentTmpl(obj)
	if tmpl == nil {
		return fmt.Errorf("Render: don't know how to render %T", obj)
	}

	h := handler.GetHandler(obj)
	if h == nil {
		return fmt.Errorf("Render: GetHandler return nil for %T", obj)
	}

	dot, err := h(r.ctx, obj)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, dot)
}
