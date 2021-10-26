package handler

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/getsentry/sentry-go"
)

// SentryHandler ...
type SentryHandler struct {
	mu       sync.RWMutex
	handlers map[reflect.Type]reflect.Value
}

// NewSentry ...
func NewSentry() *SentryHandler {
	return &SentryHandler{
		handlers: make(map[reflect.Type]reflect.Value),
	}
}

// AddHandler adds the given event handler.
func (h *SentryHandler) AddHandler(v interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	handler, err := newSentryHandler(v)
	if err != nil {
		panic(err)
	}

	h.handlers[handler.event] = handler.callback
}

// Handle handles the given event + sentry hub
func (h *SentryHandler) Handle(hub *sentry.Hub, ev interface{}) {
	evV := reflect.ValueOf(ev)
	evT := evV.Type()

	h.mu.RLock()
	defer h.mu.RUnlock()

	fn, ok := h.handlers[evT]
	if !ok {
		fmt.Println("no handler for event", evT.String())
		return
	}

	fn.Call([]reflect.Value{
		reflect.ValueOf(hub),
		evV,
	})
}

var sentryHandler = reflect.TypeOf(&sentry.Hub{})

func newSentryHandler(fn interface{}) (handler, error) {
	fnV := reflect.ValueOf(fn)
	fnT := fnV.Type()

	handler := handler{
		callback: fnV,
	}

	if fnT.Kind() != reflect.Func {
		return handler, errors.New("fn is not a function")
	}

	if fnT.NumIn() != 2 {
		return handler, errors.New("number of arguments must be 2")
	}

	if fnT.NumOut() != 0 {
		return handler, errors.New("handler must have no returns")
	}

	handler.event = fnT.In(1)

	if fnT.In(0) != sentryHandler {
		return handler, errors.New("handler must have *sentry.Hub as first argument")
	}

	var kind = handler.event.Kind()

	// Accept either pointer type or interface{} type
	if kind != reflect.Ptr {
		return handler, errors.New("argument 1 must be a pointer")
	}

	return handler, nil
}
