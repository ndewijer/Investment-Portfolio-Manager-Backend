package validation

import (
	"fmt"
	"strings"
)

type Error struct {
	Fields map[string]string
}

func (e *Error) Error() string {
	msgs := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(msgs, "; ")
}
