package errors

import (
	"fmt"
	"net/http"
)

type ServiceError struct {
	InternalErr error  `json:"-"`
	HttpStatus  int    `json:"-"`
	ErrorCode   string `json:"errorCode"`
	Message     string `json:"message"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

func (e *ServiceError) WithInternalErr(err error) *ServiceError {
	return &ServiceError{
		InternalErr: err,
		HttpStatus:  e.HttpStatus,
		ErrorCode:   e.ErrorCode,
		Message:     e.Message,
	}
}

func (e *ServiceError) Is(target error) bool {
	t, ok := target.(*ServiceError)
	if !ok {
		return false
	}
	return e.ErrorCode == t.ErrorCode
}

func (e *ServiceError) Unwrap() error {
	return e.InternalErr
}

func (e *ServiceError) DetailedError() string {
	if e.InternalErr != nil {
		return fmt.Sprintf("%s: %s [Internal Error: %s]", e.ErrorCode, e.Message, e.InternalErr.Error())
	} else {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
	}
}

func NewServiceError(httpStatus int, errorCode, message string) *ServiceError {
	return &ServiceError{HttpStatus: httpStatus, ErrorCode: errorCode, Message: message}
}

var (
	// standard errors
	ErrBadRequest     = NewServiceError(http.StatusBadRequest, "BAD_REQUEST", "The request could not be processed due to malformed syntax or missing required parameters.")
	ErrInternalServer = NewServiceError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrAccessDenied   = NewServiceError(http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to perform this action.")
	ErrUnauthorized   = NewServiceError(http.StatusUnauthorized, "UNAUTHORIZED_REQUEST", "Unauthorized request. Please provide a valid Authorization token/cookie.")

	// others
	ErrMissingRequestID = NewServiceError(http.StatusBadRequest, "MISSING_REQUEST_ID", "Please provide a valid X-Request-ID header.")
	ErrInvalidRequestID = NewServiceError(http.StatusBadRequest, "INVALID_REQUEST_ID", "X-Request-ID must be a valid UUID.")
	ErrTripNotFound     = NewServiceError(http.StatusNotFound, "TRIP_NOT_FOUND", "The requested trip was not found.")
	ErrMongo            = NewServiceError(http.StatusInternalServerError, "MONGO_ERROR", "An unexpected error occurred on the server. Please try again later or contact support if the issue persists.")
	ErrMongoNoDocs      = NewServiceError(http.StatusNotFound, "MONGO_DOC_NOT_FOUND", "The requested document does not exist in mongo.")
)
