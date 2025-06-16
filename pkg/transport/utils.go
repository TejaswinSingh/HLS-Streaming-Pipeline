package transport

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/Resolyte-official/service-gps/pkg/logging"
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
