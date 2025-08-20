package repository

import (
	"errors"
)

var (
	ErrTechnicalSpecificationNotFound = errors.New("technical specification not found")
	ErrVersionNotFound                = errors.New("version not found")
	ErrDuplicateVersion               = errors.New("version with this number already exists for this technical specification")
	ErrLLMCacheNotFound               = errors.New("llm cache not found")
	ErrErrorNotFound                  = errors.New("error not found")
)
