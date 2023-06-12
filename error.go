package worker

import "fmt"

// Error worker error
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return fmt.Sprintf("worker: %s", e.msg)
}
