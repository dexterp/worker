package worker

import (
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	e := &Error{worker: "test"}
	if !strings.HasPrefix(e.Error(), "worker: ") {
		t.Fail()
	}
}
