package service

import "errors"

var ErrPipeNotConfigured = errors.New("pipe is not configured")

// ErrJSONParsing hides json marshalling errors from users
var ErrJSONParsing = errors.New("failed to parse response from service, please contact support")

type SetParamsError struct{ Err error }

func (e SetParamsError) Error() string { return e.Err.Error() }
func (e SetParamsError) Unwrap() error { return e.Err }

type LoadError struct{ Err error }

func (e LoadError) Error() string { return e.Err.Error() }
func (e LoadError) Unwrap() error { return e.Err }

type RefreshError struct{ Err error }

func (e RefreshError) Error() string { return e.Err.Error() }
func (e RefreshError) Unwrap() error { return e.Err }
