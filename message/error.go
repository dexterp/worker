package message

import "fmt"

// Error worker error
type Error struct {
	Msg string
	error
}

func (e *Error) Error() string {
	return fmt.Sprintf("message: %s", e.Msg)
}
