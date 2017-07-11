package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"reflect"
)

// Handler takes context and an object (TableInfo/SelectStmt...) as
// parameters, returns an object (the 'dot' object) for template renderring.
type Handler func(*context.Context, interface{}) (interface{}, error)

// Map type -> Handler
var (
	handlers map[reflect.Type]Handler = make(map[reflect.Type]Handler)
)

// Regist a handler for a given type.
func RegistHandler(obj interface{}, handler Handler) {
	t := reflect.TypeOf(obj)
	if _, ok := handlers[t]; ok {
		panic(fmt.Errorf("RegistHandler: %T has already registed", obj))
	}
	handlers[t] = handler
}

// Get handler for a given object(type).
func GetHandler(obj interface{}) Handler {
	handler, ok := handlers[reflect.TypeOf(obj)]
	if ok {
		return handler
	}
	return nil
}
