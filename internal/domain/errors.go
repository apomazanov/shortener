package domain

import "errors"

var ErrAliasDuplicate = errors.New("alias already exists")
var ErrOriginalURLDuplicate = errors.New("original URL already exists")
var ErrNotFound = errors.New("entity not found")
var ErrSaveRetryLimitExceeded = errors.New("generating unique alias: retry limit exceeded")
var ErrFoundDeleted = errors.New("entity existed but was deleted")
