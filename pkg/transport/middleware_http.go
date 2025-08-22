package transport

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Resolyte-official/service-gps/pkg/config"
	"github.com/Resolyte-official/service-gps/pkg/contextkeys"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	grpcClient "github.com/Resolyte-official/service-gps/pkg/grpc"
	"github.com/Resolyte-official/service-gps/pkg/logging"
	"github.com/Resolyte-official/service-gps/pkg/metrics"
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

func TracingLoggingMetricsMiddleware(logger logging.Logger, envConfig *config.Configuration, metrics *metrics.Metrics) func(http.Handler) http.Handler {
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
				logE.UserID = rw.UserID
				logE.Duration = time.Since(start).String()
				logE.Log(logger)

				// metrics
				labels := []string{
					"transport", logE.Transport,
					"path", logE.Path,
					"rpc", "",
					"status", strconv.Itoa(logE.Status),
				}
				metrics.RequestLatency.With(labels...).Observe(time.Since(start).Seconds())
				metrics.RequestCount.With(labels...).Add(1)
			}()

			// don't trace request to /metrics endpoint
			if r.URL.Path != "/metrics" {

				// requestID is required
				reqID := r.Header.Get("X-Request-ID")
				if reqID == "" {
					encodeError(r.Context(), svcErrors.ErrMissingRequestID, rw)
					return
				}
				if uuid.Validate(reqID) != nil {
					encodeError(r.Context(), svcErrors.ErrInvalidRequestID, rw)
					return
				}
				logE.RequestID = reqID
				ctx := context.WithValue(r.Context(), contextkeys.RequestIDKey, reqID)
				r = r.WithContext(ctx)
			}

			// call next handler
			next.ServeHTTP(rw, r)
		})
	}
}

type AuthToken string

const (
	AccessToken  AuthToken = "access_token"
	RefreshToken AuthToken = "refresh_token"
)

func AuthMiddleware(govClient grpcClient.GovernanceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			switch r.URL.Path {
			case "/healthz", "/start-trip", "/end-trip", "/metrics":
				// unprotected paths
			case "/get-vehicle-trips":
				claims, err := governanceCallback(w, r, govClient, "gps.vehicles.trip_history", "view")
				if err != nil {
					encodeError(r.Context(), err, w)
					return
				}
				ctx := context.WithValue(r.Context(), contextkeys.PbClaimsKey, claims)
				r = r.WithContext(ctx)
			default:
				// 404 paths
			}

			next.ServeHTTP(w, r)
		})
	}
}
