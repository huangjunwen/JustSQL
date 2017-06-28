package tmpl

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/embed_db"
	"io"
	"reflect"
	"text/template"
)

// RenderInfo contains information to render variable of specified
// type: SelectStmt/TableInfo ...
type RenderInfo struct {
	// Type of variable, should be unique
	typ reflect.Type

	// Name of the type, should be unique
	name string

	// Templates
	defaultTmpl *template.Template
	tmpl        *template.Template

	// Function to create template context (dot)
	contextFn func(*embed_db.EmbedDB, interface{}) (interface{}, error)
}

var (
	type2Name map[reflect.Type]string = make(map[reflect.Type]string)
	name2Info map[string]*RenderInfo  = make(map[string]*RenderInfo)
)

func RegistRenderInfo(obj interface{}, name string, default_tmpl_text string,
	context_fn func(*embed_db.EmbedDB, interface{}) (interface{}, error)) {

	// Create RenderInfo
	default_tmpl, err := template.New("default").Parse(default_tmpl_text)
	if err != nil {
		panic(err)
	}

	info := &RenderInfo{
		typ:         reflect.TypeOf(obj),
		name:        name,
		contextFn:   context_fn,
		defaultTmpl: default_tmpl,
	}

	// Check type and name
	if info.typ == nil {
		panic(fmt.Errorf("Missing RenderInfo.typ"))
	} else {
		if _, ok := type2Name[info.typ]; ok {
			panic(fmt.Errorf("Duplicated RenderInfo.typ"))
		}
	}

	if info.name == "" {
		panic(fmt.Errorf("Missing RenderInfo.name"))
	} else {
		if _, ok := name2Info[info.name]; ok {
			panic(fmt.Errorf("Duplicated RenderInfo.name: %q", info.name))
		}
	}

	// Store
	type2Name[info.typ] = info.name
	name2Info[info.name] = info
}

func Get(obj interface{}) *RenderInfo {
	if name, ok := type2Name[reflect.TypeOf(obj)]; ok {
		return GetByName(name)
	}
	return nil
}

func GetByName(name string) *RenderInfo {
	if info, ok := name2Info[name]; ok {
		return info
	}
	return nil
}

func (info *RenderInfo) AddTemplate(template_name, template_text string) error {
	if tmpl, err := info.defaultTmpl.New(template_name).Parse(template_text); err != nil {
		return err
	} else {
		if template_name == "" {
			info.tmpl = tmpl
		}
		return nil
	}
}

func Render(db *embed_db.EmbedDB, obj interface{}, w io.Writer) error {
	info := Get(obj)
	if info == nil {
		return fmt.Errorf("Don't know how to render %T", obj)
	}

	tmpl := info.tmpl
	if tmpl == nil {
		tmpl = info.defaultTmpl
	}

	ctx, err := info.contextFn(db, obj)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, ctx)
}
