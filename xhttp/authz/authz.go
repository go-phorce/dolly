// Package authz provides an implemention of http authorization where specific
// URI (or URI's and their children) are allowed access by a set of roles
//
// the caller can supply a way to map from a request to a role name.
//
// the access control points are on entire URI segments only, e.g.
// Allow("/foo/bar", "bob")
// gives access to /foo/bar /foo/bar/baz, but not /foo/barry
//
// Access is based on the deepest matching path, not the accumulated paths, so,
// Allow("/foo", "bob")
// Allow("/foo/bar", "barry")
// will allow barry access to /foo/bar but not access to /foo
//
// AllowAny("/foo") will allow any authenticated request access to the /foo resource
// AllowAnyRole("/bar") will allow any authenticated request with a non-empty role access to the /bar resource
//
// AllowAny, allowAnyRole always overrides any matching Allow regardless of the order of calls
// multiple calls to Allow for the same resource are cumulative, e.g.
// Allow("/foo", "bob")
// Allow("/foo", "barry")
// is equivilent to
// Allow("/foo", "bob", "barry")
//
// Once you've built your Provider you can call NewHandler to get a http.Handler
// that implements those rules.
//
package authz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/go-phorce/dolly/algorithms/math"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "authz")

var (
	// ErrNoRoleMapperSpecified can't call NewHandler before you've set the RoleMapper function
	ErrNoRoleMapperSpecified = errors.New("Provider has no RoleMapper func specified, you must have a RoleMapper set to be able to create a http.Handler")
	// ErrNoPathsConfigured is returned by NewHandler if you call NewHandler, but haven't configured any paths to be accessible
	ErrNoPathsConfigured = errors.New("Provider has no paths authorizated, you must authorization at least one path before being able to create a http.Handler")
)

// RoleMapper abstracts how a role is extracted from an HTTP request
// Your role mapper can be called concurrently by multiple go-routines so should
// be careful if it manages any state.
type RoleMapper func(r *http.Request) string

// Provider represents an Authorization provider,
// You can call Allow or AllowAny to specify which roles are allowed
// access to which path segments.
// once configured you can create a http.Handler that enforces that
// configuration for you by calling NewHandler
type Provider struct {
	roleMapper RoleMapper
	pathRoot   *pathNode
}

type allowTypes int8

const (
	allowAny allowTypes = 1 << iota
	allowAnyRole
)

// the auth info is stored in a tree based on the path segments
// the deepest node that matches the request is used to validate the request
// e.g. if /v1/foo is allowed access by sales and
//			   /v1/bar is allowed access by baristas
// the tree is
// ""
//	- "v1"
//		- "foo"	allow sales
//		- "bar" allow baristas
//
type pathNode struct {
	value        string
	children     map[string]*pathNode
	allowedRoles map[string]bool
	allow        allowTypes
}

// New returns new Authz provider
func New(allow, allowAny, allowAnyRole []string) (*Provider, error) {
	az := new(Provider)

	if len(allowAny) > 0 {
		for _, s := range allowAny {
			az.AllowAny(s)
			logger.Infof("api=authz.New, AllowAny=%s", s)
		}
	}

	if len(allowAnyRole) > 0 {
		for _, s := range allowAnyRole {
			az.AllowAnyRole(s)
			logger.Infof("api=authz.New, AllowAnyRole=%s", s)
		}
	}

	if len(allow) > 0 {
		for _, s := range allow {
			parts := strings.Split(s, ":")
			if len(parts) != 2 {
				return nil, errors.NotValidf("Authz allow configuration '%s'", s)
			}
			logger.Infof("api=authz.New, Allow=%s:%s", parts[0], parts[1])
			roles := strings.Split(parts[1], ",")
			if len(roles) < 1 {
				return nil, errors.NotValidf("Authz allow configuration '%s'", s)
			}
			az.Allow(parts[0], roles...)
		}
	}

	return az, nil
}

// treeAtText will return a string of the current configured tree in
// human readable text format.
func (c *Provider) treeAsText() string {
	o := bytes.NewBuffer(make([]byte, 0, 256))
	io.WriteString(o, "\n")
	roles := func(o io.Writer, n *pathNode) {
		if n.allowAny() {
			io.WriteString(o, "[Any]")
			return
		}
		if (n.allow & allowAnyRole) != 0 {
			io.WriteString(o, "[Any Role]")
			return
		}
		if len(n.allowedRoles) == 0 {
			return
		}
		fmt.Fprintf(o, "[%s]", strings.Join(n.allowedRoleKeys(), ","))
	}
	var visitNode func(int, *pathNode)
	visitNode = func(depth int, n *pathNode) {
		pad := strings.Repeat(" ", depth*2)
		slash := ""
		if len(n.children) > 0 {
			slash = "/"
		}
		rolePad := strings.Repeat(" ", math.Max(1, 32-len(pad)-len(slash)-len(n.value)))
		fmt.Fprintf(o, "%s  %s%s %s", pad, n.value, slash, rolePad)
		roles(o, n)
		fmt.Fprintln(o)
		for _, ck := range n.childKeys() {
			visitNode(depth+1, n.children[ck])
		}
	}
	visitNode(0, c.pathRoot)
	return o.String()
}

// newPathNode returns a newly created pathNode initialized with the supplied path segment
func newPathNode(pathItem string) *pathNode {
	return &pathNode{
		value:        pathItem,
		children:     make(map[string]*pathNode),
		allowedRoles: make(map[string]bool),
	}
}

// childKeys returns a slice containing the child key names sorted alpabetically
func (n *pathNode) childKeys() []string {
	r := make([]string, 0, len(n.children))
	for k := range n.children {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

// allowedRoleKeys return a slice containing the allowed role name sorted alphabetically
func (n *pathNode) allowedRoleKeys() []string {
	r := make([]string, 0, len(n.allowedRoles))
	for k := range n.allowedRoles {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

// clone returns a deep copy of this pathNode
func (n *pathNode) clone() *pathNode {
	if n == nil {
		return nil
	}
	c := newPathNode(n.value)
	c.allow = n.allow
	for k, v := range n.children {
		c.children[k] = v.clone()
	}
	for k := range n.allowedRoles {
		c.allowedRoles[k] = true
	}
	return c
}

func (n *pathNode) allowAny() bool {
	return (n.allow & allowAny) != 0
}

func (n *pathNode) allowRole(r string) bool {
	if r == "" {
		return false
	}
	return ((n.allow & allowAnyRole) != 0) || n.allowedRoles[r]
}

// Clone returns a deep copy of this Provider
func (c *Provider) Clone() *Provider {
	return &Provider{
		roleMapper: c.roleMapper,
		pathRoot:   c.pathRoot.clone(),
	}
}

// SetRoleMapper configures the function that provides the mapping from an HTTP request to a role name
func (c *Provider) SetRoleMapper(m RoleMapper) {
	c.roleMapper = m
}

// AllowAny will allow any authenticated request access to this path and its children
// [unless a specific Allow/AllowAny is called for a child path]
func (c *Provider) AllowAny(path string) {
	c.walkPath(path, true).allow = allowAny
}

// AllowAnyRole will allow any authenticated request that include a non empty role
// access to this path and its children
// [unless a specific Allow/AllowAny is called for a child path]
func (c *Provider) AllowAnyRole(path string) {
	c.walkPath(path, true).allow |= allowAnyRole
}

// Allow will allow the specified roles access to this path and its children
// [unless a specific Allow/AllowAny is called for a child path]
// multiple calls to Allow for the same path are cumulative
func (c *Provider) Allow(path string, roles ...string) {
	node := c.walkPath(path, true)
	for _, role := range roles {
		node.allowedRoles[role] = true
	}
}

// walkPath does the work of converting a URI path into a tree of pathNodes
// if create is true, all nodes required to create a tree equaling the supplied
// path will be created if needed.
// if create is false, the deepest node matching the supplied path is returned.
//
// walkPath is safe for concurrent use only if create is false, and it has previously
// been called with create=true
func (c *Provider) walkPath(path string, create bool) *pathNode {
	if len(path) == 0 || path[0] != '/' {
		panic(fmt.Sprintf("Invalid path supplied to walkPath %v", path))
	}
	if c.pathRoot == nil {
		c.pathRoot = newPathNode("")
	}
	pathLen := len(path)
	pathPos := 1
	currentNode := c.pathRoot
	for pathPos < pathLen {
		segEnd := pathPos
		for segEnd < pathLen && path[segEnd] != '/' {
			segEnd++
		}
		pathSegment := path[pathPos:segEnd]
		childNode := currentNode.children[pathSegment]
		if childNode == nil && !create {
			return currentNode
		}
		if childNode == nil {
			childNode = newPathNode(pathSegment)
			currentNode.children[pathSegment] = childNode
		}
		currentNode = childNode
		pathPos = segEnd + 1
	}
	return currentNode
}

// isAllowed returns true if access to 'path' is allowed for the specified role.
func (c *Provider) isAllowed(path, role string) bool {
	node := c.walkPath(path, false)
	allowAny := node.allowAny()
	allowRole := false
	if !allowAny {
		allowRole = node.allowRole(role)
	}
	res := allowAny || allowRole
	if res {
		var reason string
		if allowAny {
			reason = "AllowAny"
		} else {
			reason = "Role"
		}
		logger.Infof("api=Authz, status=allowed, role='%s', path=%s, reason='%s', node=%s", role, path, reason, node.value)
	} else {
		logger.Infof("api=Authz, status=disallowed, role='%s', path=%s, allowed_roles='%v', node=%s", role, path, strings.Join(node.allowedRoleKeys(), ","), node.value)
	}
	return res
}

// isRequestAllowed returns true if access to the supplied http.request is allowed
func (c *Provider) isRequestAllowed(r *http.Request) bool {
	return c.isAllowed(r.URL.Path, c.roleMapper(r))
}

// NewHandler returns a http.Handler that enforces the current authorization configuration
// The handler has its own copy of the configuration changes to the Provider after calling
// NewHandler won't affect previously created Handlers.
// The returned handler will extract the role and verify that the role has access to the
// URI being request, and either return an error, or pass the request on to the supplied
// delegate handler
func (c *Provider) NewHandler(delegate http.Handler) (http.Handler, error) {
	if c.roleMapper == nil {
		return nil, errors.Trace(ErrNoRoleMapperSpecified)
	}
	if c.pathRoot == nil {
		return nil, errors.Trace(ErrNoPathsConfigured)
	}
	errBody := map[string]string{"code": "Forbidden", "message": "You are not authorized to access this URI"}
	errBodyBytes, err := json.Marshal(errBody)
	if err != nil {
		return nil, errors.Trace(err)
	}
	h := &authHandler{
		delegate:  delegate,
		config:    c.Clone(),
		errorBody: errBodyBytes,
	}
	logger.Infof("api=authz.NewHandler, config=[%s]", h.config.treeAsText())
	return h, nil
}

type authHandler struct {
	delegate  http.Handler
	config    *Provider
	errorBody []byte
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.config.isRequestAllowed(r) {
		a.delegate.ServeHTTP(w, r)
	} else {
		w.Header().Add(header.ContentType, header.ApplicationJSON)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(a.errorBody)
	}
}
