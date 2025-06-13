package transport

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Resolyte-official/service-gps/pkg/config"
	myerrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"github.com/Resolyte-official/service-gps/pkg/logging"
	"github.com/google/uuid"
)

// CorsOptions defines the structure for CORS configuration options
type CorsOptions struct {
	Origins          []string // Origins is a list of origins that are allowed to access the resource
	Methods          []string // Methods is a list of HTTP methods allowed when accessing the resource
	Headers          []string // Headers is a list of HTTP headers that can be used when making the actual request
	ExposeHeaders    []string // ExposeHeaders indicates which headers are safe to expose to the API of a CORS API specification
	AllowCredentials bool     // AllowCredentials indicates whether or not the response to the request can be exposed when the credentials flag is true
}

func CorsMiddleware(corsConfig *CorsOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			origin := r.Header.Get("Origin")

			if origin == "" || !slices.Contains(corsConfig.Origins, origin) {
				if r.Method == http.MethodOptions { // preflight request
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// only set CORS Headers for allowed origins
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.Methods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.Headers, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprint(corsConfig.AllowCredentials))

			// preflight request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TracingAndLoggingMiddleware(logger logging.Logger, envConfig *config.Configuration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()

			rw := &ResponseWriter{ResponseWriter: w, status: http.StatusOK}

			logE := &LogEntry{
				Transport: string(envConfig.HttpMode),
				Method:    r.Method,
				Path:      r.URL.Path,
			}
			defer func() {
				logE.Status = rw.status
				logE.Error = rw.Err
				logE.Duration = time.Since(start).String()
				logE.Log(logger)
			}()

			// don't trace preflight request
			if r.Method != http.MethodOptions {

				// requestID is required
				reqID := r.Header.Get("X-Request-ID")
				if reqID == "" {
					encodeError(r.Context(), myerrors.ErrMissingRequestID, rw)
					return
				}
				if uuid.Validate(reqID) != nil {
					encodeError(r.Context(), myerrors.ErrInvalidRequestID, rw)
					return
				}
				logE.RequestID = reqID

				// TODO: get userID from request
			}

			// call next handler
			next.ServeHTTP(rw, r)
		})
	}
}
