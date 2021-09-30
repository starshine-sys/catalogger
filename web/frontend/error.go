package main

import (
	"net/http"

	"github.com/google/uuid"
)

type errData struct {
	ErrorCode int
	ErrorName string
	Text      string
	UUID      string
}

func (s *server) error(w http.ResponseWriter, code int, genID bool, str string) uuid.UUID {
	id := uuid.Nil

	data := errData{
		ErrorCode: code,
		ErrorName: http.StatusText(code),
		Text:      str,
	}

	if genID {
		id = uuid.New()
		data.UUID = id.String()
	}

	err := tmpl.ExecuteTemplate(w, "error.html", data)
	if err != nil {
		s.Sugar.Errorf("Error executing template: %v", err)
		return id
	}
	return id
}
