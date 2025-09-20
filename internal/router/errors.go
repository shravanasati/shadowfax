package router

import "errors"

var ErrNotFound = errors.New("404 not found")
var ErrMethodNotAllowed = errors.New("405 method not allowed")
