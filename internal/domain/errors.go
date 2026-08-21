package domain

import "errors"

// ErrAliasDuplicate is an error for duplicated alias in storage.
var ErrAliasDuplicate = errors.New("alias already exists")

// ErrOriginalURLDuplicate is an error for duplicated original URL in storage.
var ErrOriginalURLDuplicate = errors.New("original URL already exists")

// ErrNotFound is an error for non-existing alias.
var ErrNotFound = errors.New("entity not found")

// ErrSaveRetryLimitExceeded is an error for exceed limit of alias generating retries.
var ErrSaveRetryLimitExceeded = errors.New("generating unique alias: retry limit exceeded")

// ErrFoundDeleted is an error for alias that has already been deleted from storage.
var ErrFoundDeleted = errors.New("entity existed but was deleted")
