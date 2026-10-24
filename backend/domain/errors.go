package domain

import "errors"

var ErrInvalidInput = errors.New("entrada inválida")
var ErrDatabase = errors.New("database error")
var ErrFileSystem = errors.New("filesystem error")
var ErrNotFound = errors.New("not found")
