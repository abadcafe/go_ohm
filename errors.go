package go_ohm

import (
	"fmt"
)

type ErrorUnsupportedObjectType struct {
	error
}

func NewErrorUnsupportedObjectType(nam string) *ErrorUnsupportedObjectType {
	return &ErrorUnsupportedObjectType{
		fmt.Errorf("the type of object '%s' is unsupported", nam),
	}
}

type ErrorRedisCommandFailed struct {
	error
}

func NewErrorRedisCommandFailed(nam string, err error) *ErrorRedisCommandFailed {
	return &ErrorRedisCommandFailed{
		fmt.Errorf("execute redis command failed on object '%s': %w", nam, err),
	}
}

type ErrorObjectWithoutHashKey struct {
	error
}

func NewErrorObjectWithoutHashKey(nam string) *ErrorObjectWithoutHashKey {
	return &ErrorObjectWithoutHashKey{
		fmt.Errorf("can not determine hash key of object '%s'", nam),
	}
}

type ErrorBugOccurred struct {
	error
}

func NewErrorBugOccurred(nam string) *ErrorBugOccurred {
	return &ErrorBugOccurred{
		fmt.Errorf("bug occurred on object '%s'", nam),
	}
}

type ErrorJsonFailed struct {
	error
}

func NewErrorJsonFailed(nam string, err error) *ErrorJsonFailed {
	return &ErrorJsonFailed{
		fmt.Errorf("json marshal/unmarshal failed on object '%s': %w", nam, err),
	}
}
