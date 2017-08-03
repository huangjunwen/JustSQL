package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"io"
	"reflect"
	"text/template"
)

const (
	RootTemplateName    = ""
	DefaultTemplateName = "default"
)

// Handler takes context and an object (TableMeta/SelectStmt...) as
// parameters, returns an object (the 'dot' object) for template renderring.
type Handler func(*context.Context, interface{}) (interface{}, error)

// Builtin mapping.
var (
	// Map type name -> type
	typeMap map[string]reflect.Type = make(map[string]reflect.Type)
	// Map type -> handler
	handlerMap map[reflect.Type]Handler = make(map[reflect.Type]Handler)
	// Map type -> (template name -> template content)
	templateMap map[reflect.Type]map[string]string = make(map[reflect.Type]map[string]string)
	// True if RenderContext has been created.
	initialized = false
)

// RegistType regist a type for renderring.
func RegistType(typeName string, obj interface{}, handler Handler) {

	if initialized {
		panic(fmt.Errorf("RegistType: called after initialized"))
	}

	if _, ok := typeMap[typeName]; ok {
		panic(fmt.Errorf("RegistType: type name %+q has already registered", typeName))
	}

	t := reflect.TypeOf(obj)
	if _, ok := handlerMap[t]; ok {
		panic(fmt.Errorf("RegistType: type %T has already registered", obj))
	}

	typeMap[typeName] = t
	handlerMap[t] = handler
	templateMap[t] = map[string]string{}

}

// RegistBuiltinTemplate regist a builtin template for a type with the given name.
func RegistBuiltinTemplate(typeName string, templateName string, templateContent string) {

	if initialized {
		panic(fmt.Errorf("RegistBuiltinTemplate: called after initialized"))
	}

	t, ok := typeMap[typeName]
	if !ok {
		panic(fmt.Errorf("RegistBuiltinTemplate: type %+q has not registered yet", typeName))
	}

	templates := templateMap[t]
	if _, ok := templates[templateName]; ok {
		panic(fmt.Errorf("RegistBuiltinTemplate: template %+q of type %+q has already registered",
			templateName, typeName))
	}

	templates[templateName] = templateContent

}

// RenderContext contain information to render objects.
type RenderContext struct {
	// Global context.
	*context.Context

	// Root template is empty template used to create
	// associated templates, see http://golang.org/pkg/text/template/#hdr-Associated_templates
	RootTemplates map[reflect.Type]*template.Template

	// Templates used to render object.
	Templates map[reflect.Type]*template.Template

	// Extra functions used in templates.
	ExtraFuncs template.FuncMap
}

// NewRenderContext create RenderContext from Context.
func NewRenderContext(ctx *context.Context) (*RenderContext, error) {

	// Mark initialized.
	initialized = true

	ret := &RenderContext{
		Context:       ctx,
		RootTemplates: make(map[reflect.Type]*template.Template),
		Templates:     make(map[reflect.Type]*template.Template),
		ExtraFuncs:    BuildExtraFuncs(ctx),
	}

	for _, t := range typeMap {
		// Create empty root template for each registered type.
		root := template.Must(template.New(RootTemplateName).Parse(""))
		ret.RootTemplates[t] = root
		ret.Templates[t] = root

		// Parse all builtin templates.
		for templateName, templateContent := range templateMap[t] {
			tmpl := root.New(templateName).Funcs(ret.ExtraFuncs)
			if _, err := tmpl.Parse(templateContent); err != nil {
				return nil, err
			}
			if templateName == DefaultTemplateName {
				ret.Templates[t] = tmpl
			}
		}
	}

	return ret, nil
}

// AddTemplate add a template for a type with the given name.
// This latest added template is used as the main template to render the given type.
func (r *RenderContext) AddTemplate(typeName string, templateName string, templateContent string) error {

	// Check type.
	t, ok := typeMap[typeName]
	if !ok {
		return fmt.Errorf("AddTemplate: type %+q has not registered yet", typeName)
	}

	// Parse the template.
	tmpl := r.RootTemplates[t].New(templateName).Funcs(r.ExtraFuncs)
	if _, err := tmpl.Parse(templateContent); err != nil {
		return err
	}

	// Latest added template is used as the main template.
	r.Templates[t] = tmpl

	return nil
}

// Render an object.
func (r *RenderContext) Render(obj interface{}, w io.Writer) error {

	t := reflect.TypeOf(obj)

	// Choose template.
	tmpl, ok := r.Templates[t]
	if !ok {
		return fmt.Errorf("Render: don't know how to render %T", obj)
	}

	// Generate 'dot' object.
	dot, err := handlerMap[t](r.Context, obj)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, dot)

}
