package error

import (
	"errors"
	"fmt"
)

var Nil = errors.New("")

var errorMap = map[string]error{
	Nil.Error(): Nil,
}

func FromString(errString string) error {
	err, found := errorMap[errString]
	if !found {
		err = fmt.Errorf("Undefined: %s", errString)
	}
	return err
}
