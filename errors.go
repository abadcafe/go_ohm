package go_ohm

import (
	"fmt"
)

type ErrorUnsupportedObjectType struct {
	error
}

func newErrorUnsupportedObjectType(nam string) *ErrorUnsupportedObjectType {
	return &ErrorUnsupportedObjectType{
		fmt.Errorf("the type of object '%s' is unsupported", nam),
	}
}

type ErrorRedisCommandFailed struct {
	error
}

func newErrorRedisCommandFailed(nam string, err error) *ErrorRedisCommandFailed {
	return &ErrorRedisCommandFailed{
		fmt.Errorf("execute redis command failed on object '%s': %w", nam, err),
	}
}

type ErrorObjectWithoutHashKey struct {
	error
}

func newErrorObjectWithoutHashKey(nam string) *ErrorObjectWithoutHashKey {
	return &ErrorObjectWithoutHashKey{
		fmt.Errorf("can not determine hash key of object '%s'", nam),
	}
}

type ErrorBugOccurred struct {
	error
}

func newErrorBugOccurred(nam string) *ErrorBugOccurred {
	return &ErrorBugOccurred{
		fmt.Errorf("bug occurred on object '%s'", nam),
	}
}

type ErrorJsonFailed struct {
	error
}

func newErrorJsonFailed(nam string, err error) *ErrorJsonFailed {
	return &ErrorJsonFailed{
		fmt.Errorf("json marshal/unmarshal failed on object '%s': %w", nam, err),
	}
}
