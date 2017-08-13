package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"io"
	"reflect"
	"text/template"
)

const (
	RootTemplateSetName    = ""
	DefaultTemplateSetName = "default"
)

// Handler takes a renderer and an object (TableMeta/SelectStmt...) as
// parameters, returns an object (the 'dot' object) for template renderring.
type Handler func(*Renderer, interface{}) (interface{}, error)

// Builtin mapping.
var (
	// Map type name -> type
	typeMap map[string]reflect.Type = make(map[string]reflect.Type)
	// Map type -> handler
	handlerMap map[reflect.Type]Handler = make(map[reflect.Type]Handler)
	// Map type -> (template set name -> template content)
	templateMap map[reflect.Type]map[string]string = make(map[reflect.Type]map[string]string)
	// True if Renderer has been created.
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
func RegistBuiltinTemplate(typeName string, templateSetName string, templateContent string) {

	if initialized {
		panic(fmt.Errorf("RegistBuiltinTemplate: called after initialized"))
	}

	t, ok := typeMap[typeName]
	if !ok {
		panic(fmt.Errorf("RegistBuiltinTemplate: type %+q has not registered yet", typeName))
	}

	templates := templateMap[t]
	if _, ok := templates[templateSetName]; ok {
		panic(fmt.Errorf("RegistBuiltinTemplate: template set %+q of type %+q has already registered",
			templateSetName, typeName))
	}

	templates[templateSetName] = templateContent

}

// Renderer contain information to render objects.
type Renderer struct {
	// Global context.
	*context.Context

	// Which file is renderring.
	*Scopes

	// Go type adapter.
	*TypeAdapter

	// Extra functions used in templates.
	ExtraFuncs template.FuncMap

	// Map type -> (template set name -> template).
	Templates map[reflect.Type]map[string]*template.Template

	// Use which set of templates for renderring.
	// Find the first of TemplateSetName/DefaultTemplateSetName/RootTemplateSetName
	// to render.
	TemplateSetName string
}

// NewRenderer create Renderer from Context.
func NewRenderer(ctx *context.Context) (*Renderer, error) {

	// Mark initialized.
	initialized = true

	ret := &Renderer{
		Context:         ctx,
		Scopes:          NewScopes(),
		Templates:       make(map[reflect.Type]map[string]*template.Template),
		TemplateSetName: DefaultTemplateSetName,
	}

	ret.TypeAdapter = NewTypeAdapter(ret.Scopes)
	ret.ExtraFuncs = BuildExtraFuncs(ret)

	for _, t := range typeMap {
		// Create empty root template for each registered type.
		root := template.Must(template.New(RootTemplateSetName).Parse(""))
		ret.Templates[t] = map[string]*template.Template{
			RootTemplateSetName: root,
		}

		// Parse all builtin templates.
		for templateSetName, templateContent := range templateMap[t] {
			tmpl := root.New(templateSetName).Funcs(ret.ExtraFuncs)
			if _, err := tmpl.Parse(templateContent); err != nil {
				return nil, err
			}
			ret.Templates[t][templateSetName] = tmpl
		}
	}

	return ret, nil
}

// AddTemplate add a template (in a template set ) for a type.
func (r *Renderer) AddTemplate(typeName string, templateSetName string, templateContent string) error {

	if templateSetName == RootTemplateSetName {
		return fmt.Errorf("AddTemplate: empty template name")
	}

	// Check type.
	t, ok := typeMap[typeName]
	if !ok {
		return fmt.Errorf("AddTemplate: type name %+q has not registered yet", typeName)
	}

	// Parse the template.
	tmpl := r.Templates[t][RootTemplateSetName].New(templateSetName).Funcs(r.ExtraFuncs)
	if _, err := tmpl.Parse(templateContent); err != nil {
		return err
	}

	// Store.
	r.Templates[t][templateSetName] = tmpl

	return nil
}

// Use set template set name for renderring.
func (r *Renderer) Use(templateSetName string) {
	r.TemplateSetName = templateSetName
}

// Render an object.
func (r *Renderer) Render(obj interface{}, w io.Writer) error {

	t := reflect.TypeOf(obj)

	// Choose template.
	tmpls, ok := r.Templates[t]
	if !ok {
		return fmt.Errorf("Render: don't know how to render %T", obj)
	}

	var tmpl *template.Template
	for _, templateSetName := range [3]string{r.TemplateSetName, DefaultTemplateSetName, RootTemplateSetName} {
		tmpl, ok = tmpls[templateSetName]
		if ok {
			break
		}
	}
	if tmpl == nil {
		panic(fmt.Errorf("Failed to find a template for renderring"))
	}

	// Generate 'dot' object.
	dot, err := handlerMap[t](r, obj)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, dot)

}
