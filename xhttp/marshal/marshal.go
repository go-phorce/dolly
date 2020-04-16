package marshal

import (
	"bufio"
	"compress/gzip"
	goErrors "errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
	"github.com/ugorji/go/codec"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "xhttp")

// WriteHTTPResponse is for types to implement this interface to get full control
// over how they are written out as a http response
type WriteHTTPResponse interface {
	WriteHTTPResponse(w http.ResponseWriter, r *http.Request)
}

// WriteJSON will serialize the supplied body parameter as a http response.
// If the body value implements the WriteHTTPResponse interface,
// then that will be called to have it do the response generation
// if body implements error, then that's returned as a server error
// use the Error type to fully specify your error response
// otherwise body is assumed to be a succesful response, and its serialized
// and written as a json response with a 200 status code.
//
// multiple body parameters can be supplied, in which case the first
// non-nil one will be used. This is useful as it allows you to do
//		x, err := doSomething()
//		WriteJSON(logger,w,r,err,x)
//	and if there was an error, that's what'll get returned
//
func WriteJSON(w http.ResponseWriter, r *http.Request, bodies ...interface{}) {
	var body interface{}
	for i := range bodies {
		if bodies[i] != nil {
			body = bodies[i]
			break
		}
	}

	switch bv := body.(type) {
	case WriteHTTPResponse:
		// errors.Error impls WriteHTTPResponse, so will take this path and do its thing
		bv.WriteHTTPResponse(w, r)
		tryLogHttpError(bv, r)
		return

	case error:
		var resp WriteHTTPResponse

		if goErrors.As(bv, resp) {
			resp.WriteHTTPResponse(w, r)
			tryLogHttpError(bv, r)
			return
		}

		// you should really be using Error to get a good error response returned
		logger.Debugf("api=WriteJSON, reason=generic_error, type=%T, err=[%v]", bv, bv)
		WriteJSON(w, r, httperror.WithUnexpected(bv.Error()))
		return

	default:
		w.Header().Set(header.ContentType, header.ApplicationJSON)
		var out io.Writer = w
		if r != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(out)
			out = gz
			defer gz.Close()
		}
		bw := bufio.NewWriter(out)
		if err := NewEncoder(bw, r).Encode(body); err != nil {
			logger.Warningf("api=WriteJSON, reason=encode, type=%T, err=[%v]", body, err.Error())
		}
		bw.Flush()
	}
}

func tryLogHttpError(bv interface{}, r *http.Request) {
	if e, ok := bv.(*httperror.Error); ok {
		if e.HTTPStatus >= 500 {
			logger.Errorf("INTERNAL_ERROR=%s:%d:%s:%s",
				r.URL.Path, e.HTTPStatus, e.Code, e.Message)
		} else {
			logger.Warningf("API_ERROR=%s:%d:%s:%s",
				r.URL.Path, e.HTTPStatus, e.Code, e.Message)
		}
		if e.Cause != nil {
			logger.Errorf(errors.ErrorStack(e))
		}
	}
}

// WritePlainJSON will serialize the supplied body parameter as a http response.
func WritePlainJSON(w http.ResponseWriter, statusCode int, body interface{}, printSetting PrettyPrintSetting) {
	w.Header().Set(header.ContentType, header.ApplicationJSON)
	w.WriteHeader(statusCode)
	if err := codec.NewEncoder(w, encoderHandle(printSetting)).Encode(body); err != nil {
		logger.Warningf("api=WritePlainJSON, reason=encode, type=%T, err=[%v]", body, err.Error())
	}
}
