package domain

import "errors"

var ErrDuplicate = errors.New("entity already exists")
var ErrNotFound = errors.New("entity not found")
var ErrSaveRetryLimitExceeded = errors.New("generating unique alias: retry limit exceeded")
