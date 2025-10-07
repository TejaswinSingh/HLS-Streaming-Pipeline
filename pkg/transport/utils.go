package transport

import (
	"errors"
	"net/http"
	"reflect"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	grpcClient "github.com/Resolyte-official/service-gps/pkg/client/grpc"
	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"github.com/Resolyte-official/service-gps/pkg/logging"
	"github.com/google/uuid"
)

type ResponseWriter struct {
	http.ResponseWriter
	status int
	Err    string
	UserID string
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

type LogEntry struct {
	RequestID string `key:"request_id"`
	UserID    string `key:"user_id"`
	Transport string `key:"transport"`
	Method    string `key:"method"`
	Path      string `key:"path"`
	RPC       string `key:"rpc"`
	Status    int    `key:"status"`
	Error     string `key:"error"`
	Duration  string `key:"duration"`
}

func (e *LogEntry) Log(logger logging.Logger) {
	if e.RequestID == "" {
		e.RequestID = "Missing"
	}

	v := reflect.ValueOf(*e)
	t := v.Type()

	keyVals := make(map[string]any)
	for i, fieldType := range reflect.VisibleFields(t) {
		fieldVal := v.Field(i)
		tag := fieldType.Tag.Get("key")

		switch fieldVal.Kind() {
		case reflect.String:
			val := fieldVal.String()
			// skip error field, it is logged below explicitly
			if val != "" && tag != "error" {
				keyVals[tag] = val
			}
		case reflect.Int:
			if val := fieldVal.Int(); val != 0 {
				keyVals[tag] = val
			}
		}
	}
	if e.Status <= 300 {
		logger.InfoMap(keyVals)
	} else if e.Status >= 400 && e.Status < 500 { // no stack trace for 400 errors
		logger.ErrorNtMap(errors.New(e.Error), keyVals)
	} else { // 500 errors
		logger.ErrorMap(errors.New(e.Error), keyVals)
	}
}

func governanceCallback(w http.ResponseWriter, r *http.Request, govClient grpcClient.GovernanceClient, obj, act string) (*pb.Claims, error) {

	// custom response writer used for propagating info b/w middlewares
	rw, _ := w.(*ResponseWriter)

	cookie, err := r.Cookie(string(AccessToken))
	if err != nil || cookie.Valid() != nil || cookie.Value == "" {
		return nil, svcErrors.ErrUnauthorized
	}

	// grpc request to governance service for verifying token
	resp, err := govClient.Authorize(
		r.Context(),
		cookie.Value, // token
		obj,
		act,
	)
	if err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}
	claims := resp.GetClaims()
	if claims != nil && uuid.Validate(claims.Sub) == nil {
		rw.UserID = claims.Sub
	}
	if !resp.Eft {
		switch resp.Status {
		case http.StatusUnauthorized:
			return nil, svcErrors.ErrUnauthorized
		case http.StatusForbidden:
			return nil, svcErrors.ErrAccessDenied
		default:
			return nil, svcErrors.ErrInternalServer
		}
	}

	return claims, nil
}
