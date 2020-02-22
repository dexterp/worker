package worker

import "fmt"

type Error struct {
	worker string
	error
}

func (e *Error) Error() string {
	return fmt.Sprintf("worker: %s", e.worker)
}
