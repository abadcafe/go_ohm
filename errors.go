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

type ErrorRedisCommandsFailed struct {
	error
}

func NewErrorRedisCommandsFailed(redisErr error) *ErrorRedisCommandsFailed {
	return &ErrorRedisCommandsFailed{
		fmt.Errorf("execute redis commands failed: %w", redisErr),
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
