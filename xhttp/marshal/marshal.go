package marshal

import (
	"bufio"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/go-phorce/pkg/xhttp/context"
	"github.com/go-phorce/pkg/xhttp/header"
	"github.com/go-phorce/pkg/xhttp/httperror"
	"github.com/go-phorce/xlog"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/pkg", "xhttp")

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
		if e, ok := bv.(httperror.Error); ok {
			logger.Errorf("APIERROR:%s:%s:%d:%v:%s",
				r.URL.Path, context.CorrelationIDFromRequest(r), e.HTTPStatus, e.Code, e.Message)
		}
		return

	case error:
		// you should really be using Error to get a good error response returned
		logger.Debugf("generic error being returned of type %T/%v, please use Error instead", bv, bv)
		WriteJSON(w, r, httperror.New(http.StatusInternalServerError, "Unexpected error", bv.Error()))
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
			logger.Warningf("Failed to encode %T to json %v", body, err)
		}
		bw.Flush()
	}
}
