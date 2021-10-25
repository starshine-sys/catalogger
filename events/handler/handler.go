package handler

import (
	"errors"
	"reflect"
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

// Handler ...
type Handler struct {
	mu       sync.RWMutex
	handlers []handler

	HandleResponse func(reflect.Value, *Response)
	HandleError    func(reflect.Value, error)
}

// New creates a new Handler.
func New() *Handler {
	return &Handler{}
}

// Call calls the handler with the given event.
// This should be passed as a handler to the main state handler.
// Yes, this means that we reflect all events twice...
// luckily, we don't really care about the few milliseconds that might take at worst.
func (h *Handler) Call(ev interface{}) {
	evV := reflect.ValueOf(ev)
	evT := evV.Type()

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.handlers {
		if entry.not(evT) {
			continue
		}

		go h.call(entry, evV)
	}
}

func (h *Handler) call(hn handler, ev reflect.Value) {
	resps := hn.call(ev)

	err := resps[1].Interface().(error)
	if err != nil {
		h.HandleError(ev, err)
	}

	resp := resps[0].Interface().(*Response)
	if resp != nil {
		h.HandleResponse(ev, resp)
	}
}

// AddHandler adds the given function handler.
func (h *Handler) AddHandler(fn interface{}) {
	handler, err := newHandler(fn)
	if err != nil {
		panic(err)
	}

	h.mu.Lock()
	h.handlers = append(h.handlers, handler)
	h.mu.Unlock()
}

// Response must be returned by handler functions.
type Response struct {
	// Channel ID to log to
	ChannelID discord.ChannelID

	Embeds []discord.Embed
	Files  []sendpart.File
}

type handler struct {
	event    reflect.Type
	callback reflect.Value
}

var returnType0 = reflect.TypeOf(&Response{})
var returnType1 = reflect.TypeOf(error(nil))

func newHandler(fn interface{}) (handler, error) {
	fnV := reflect.ValueOf(fn)
	fnT := fnV.Type()

	handler := handler{
		callback: fnV,
	}

	if fnT.Kind() != reflect.Func {
		return handler, errors.New("fn is not a function")
	}

	if fnT.NumIn() != 1 {
		return handler, errors.New("number of arguments must be 1")
	}

	if fnT.NumOut() != 2 {
		return handler, errors.New("number of returns must be 2")
	}

	handler.event = fnT.In(0)

	if fnT.Out(0) != returnType0 {
		return handler, errors.New("return 0 must be a *Response")
	}

	if fnT.Out(1) != returnType1 {
		return handler, errors.New("return 1 must be an error")
	}

	var kind = handler.event.Kind()

	// Accept either pointer type or interface{} type
	if kind != reflect.Ptr {
		return handler, errors.New("argument must be a pointer")
	}

	return handler, nil
}

func (h handler) not(event reflect.Type) bool {
	return h.event != event
}

func (h handler) call(event reflect.Value) []reflect.Value {
	return h.callback.Call([]reflect.Value{event})
}
