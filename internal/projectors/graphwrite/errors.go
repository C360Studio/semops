package graphwrite

import (
	"errors"

	"github.com/c360studio/semstreams/pkg/errs"
)

type ClassifiedFailure struct {
	Code   string
	Detail map[string]any
}

func Classified(err error) (ClassifiedFailure, bool) {
	var classified *errs.ClassifiedError
	if !errors.As(err, &classified) {
		return ClassifiedFailure{}, false
	}
	if classified.Code == "" {
		return ClassifiedFailure{}, false
	}
	return ClassifiedFailure{
		Code:   classified.Code,
		Detail: classified.Detail,
	}, true
}

func EntityID(detail map[string]any, fallback string) string {
	if detail == nil {
		return fallback
	}
	if entity, ok := detail["entity"].(string); ok && entity != "" {
		return entity
	}
	return fallback
}
