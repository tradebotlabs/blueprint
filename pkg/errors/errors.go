// By Emran A. Hamdan, Lead Architect 
package errors

import (
	"errors"
)

const (
	
	ErrBadRequest       = "Bad request"
	ErrAlreadyExists    = "Already exists"
	ErrNotFound         = "Not Found"
	ErrUnauthorized     = "Unauthorized"
	ErrForbidden        = "Forbidden"
	ErrBadQueryParams   = "Invalid query params"
	ErrRequestTimeout   = "Request Timeout"
	
)

var (
	BadRequest            = errors.New("Bad request")	
)