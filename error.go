package worker

import "fmt"

// Error worker error
type Error struct {
	worker string
	error
}

func (e *Error) Error() string {
	return fmt.Sprintf("worker: %s", e.worker)
}
