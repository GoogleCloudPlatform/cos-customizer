package partutil

import (
	"errors"
	"testing"
)

func TestCheck(t *testing.T) {
	var testData = []struct {
		testName string
		err      error
		errStr   string
		want     bool
	}{
		{
			"error with msg",
			errors.New("testing error"),
			"msg:testing error",
			true,
		}, {
			"error with no msg",
			errors.New(""),
			"",
			true,
		}, {
			"nil error",
			nil,
			"",
			false,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if Check(input.err, input.errStr) != input.want {
				t.Errorf("wrongly detect error %v", input.err)
			}
		})
	}
}
