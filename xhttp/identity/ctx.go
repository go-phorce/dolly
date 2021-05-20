// Package identity extracts the callers contextual identity information from the HTTP/TLS
// requests and exposes them for access via the generalized go context model.
package identity

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "xhttp/context")

type contextKey int

const (
	keyContext contextKey = iota
	keyIdentity
)

// NodeInfoFactory returns NodeInfo
type NodeInfoFactory func() netutil.NodeInfo

var (
	nodeInfoFactory                      = newNodeInfoFactory()
	identityMapper     Mapper            = GuestIdentityMapper
	gRPCIdentityMapper MapperFromContext = nil
)

// RequestContext represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type RequestContext struct {
	identity      Identity
	correlationID string
	clientIP      string
}

// NewRequestContext creates a request context with a specific identity.
func NewRequestContext(id Identity) *RequestContext {
	return &RequestContext{
		identity: id,
	}
}

// Context represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type Context interface {
	Identity() Identity
	CorrelationID() string
	ClientIP() string
}

type defaultNodeInfoFactory struct {
	lock     sync.Mutex
	nodeInfo netutil.NodeInfo
}

func (f *defaultNodeInfoFactory) getNodeInfo() netutil.NodeInfo {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.nodeInfo == nil {
		nodeInfo, err := netutil.NewNodeInfo(nil)
		if err != nil {
			logger.Panicf("api=getNodeInfo, err=[%v]", err.Error())
		}
		f.nodeInfo = nodeInfo
	}

	return f.nodeInfo
}

func newNodeInfoFactory() NodeInfoFactory {
	factory := &defaultNodeInfoFactory{}
	return factory.getNodeInfo
}

// SetGlobalNodeInfo applies NodeInfo for the application
func SetGlobalNodeInfo(n netutil.NodeInfo) {
	if n == nil {
		logger.Panic("NodeInfo must not be nil")
	}
	factory := &defaultNodeInfoFactory{nodeInfo: n}
	nodeInfoFactory = factory.getNodeInfo
}

// SetGlobalIdentityMapper applies global IdentityMapper for the application
func SetGlobalIdentityMapper(e Mapper) {
	if e == nil {
		logger.Panic("IdentityMapper must not be nil")
	}
	identityMapper = e
}

// SetGlobalGRPCIdentityMapper applies global IdentityMapper for the application
func SetGlobalGRPCIdentityMapper(e MapperFromContext) {
	gRPCIdentityMapper = e
}

//FromContext extracts the RequestContext stored inside a go context. Returns null if no such value exists.
func FromContext(ctx context.Context) *RequestContext {
	ret, _ := ctx.Value(keyContext).(*RequestContext)
	return ret
}

//AddToContext returns a new golang context that adds `rq` as the dolly request context.
func AddToContext(ctx context.Context, rq *RequestContext) context.Context {
	return context.WithValue(ctx, keyContext, rq)
}

// ForRequest returns the full context ascocicated with this http request.
func ForRequest(r *http.Request) *RequestContext {
	v := r.Context().Value(keyContext)
	if v == nil {
		clientIP := ClientIPFromRequest(r)
		identity, err := identityMapper(r)
		if err != nil {
			logger.Errorf("api=ForRequest, reason=identityMapper, ip=%q, err=[%v]", clientIP, err.Error())
			identity = NewIdentity(GuestRoleName, clientIP, "")
		}

		return &RequestContext{
			identity:      identity,
			correlationID: extractCorrelationID(r),
			clientIP:      clientIP,
		}
	}
	return v.(*RequestContext)
}

// NewContextHandler returns a handler that will extact the role & contextID from the request
// and stash them away in the request context for later handlers to use.
// Also adds header to indicate which host is currently servicing the request
func NewContextHandler(delegate http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		// Set XHostname on the response
		w.Header().Set(header.XHostname, nodeInfoFactory().HostName())

		var rctx *RequestContext
		v := r.Context().Value(keyContext)
		if v == nil {
			clientIP := ClientIPFromRequest(r)
			identity, err := identityMapper(r)
			if err != nil {
				logger.Errorf("api=ForRequest, reason=identityMapper, ip=%q, err=[%v]", clientIP, err.Error())
				marshal.WriteJSON(w, r, httperror.WithUnauthorized(err.Error()))
				return
			}

			rctx = &RequestContext{
				identity:      identity,
				correlationID: extractCorrelationID(r),
				clientIP:      clientIP,
			}
			r = r.WithContext(context.WithValue(r.Context(), keyContext, rctx))
		} else {
			rctx = v.(*RequestContext)
		}

		w.Header().Set(header.XCorrelationID, rctx.correlationID)

		delegate.ServeHTTP(w, r)
	}
	return http.HandlerFunc(h)
}

var grpcGuestIdentity = NewIdentity(GuestRoleName, "", "")

// NewAuthUnaryInterceptor returns grpc.UnaryServerInterceptor that
// identity to the context
func NewAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var id Identity
		if gRPCIdentityMapper != nil {
			var err error
			id, err = gRPCIdentityMapper(ctx)
			if err != nil {
				return nil, status.Errorf(codes.PermissionDenied, "unable to get identity: %v", err)
			}
		}
		if id == nil {
			id = grpcGuestIdentity
		}
		ctx = AddToContext(ctx, NewRequestContext(id))

		return handler(ctx, req)
	}
}

// Identity returns request's identity
func (c *RequestContext) Identity() Identity {
	return c.identity
}

// CorrelationID returns request's CorrelationID, extracted from X-CorrelationID header.
// If it was not provided by the client, the a random will be generated.
func (c *RequestContext) CorrelationID() string {
	return c.correlationID
}

// ClientIP returns request's IP
func (c *RequestContext) ClientIP() string {
	return c.clientIP
}

// extractCorrelationID will find or create a requestID for this http request.
func extractCorrelationID(req *http.Request) string {
	corID := req.Header.Get(header.XCorrelationID)
	if corID == "" {
		corID = req.Header.Get(header.XDeviceID)
	}
	if corID == "" {
		corID = guid.MustCreate()
	}
	return corID
}
