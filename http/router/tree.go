package router

import (
	"net/http"
)

type pathTree struct {
	Path              string
	Function          func(http.ResponseWriter, *http.Request, []string)
	Middleware        []func(http.ResponseWriter, *http.Request, []string) bool
	SubPaths          map[string]*pathTree
	SubVariablesPaths map[Type]*pathTree

	// Subtree marks a node that also receives every path below it.
	Subtree bool

	// MethodNotAllowed marks a node planted by registration to answer 405 for a
	// method that has no handler here. A real handler always beats one of these,
	// so a stub can never shadow a route that genuinely exists.
	MethodNotAllowed bool
}

// notAllowedHandler is the single 405 responder shared by every stub. Sharing one
// value keeps the stubs free to compare against, and means the behaviour cannot
// drift between them.
func notAllowedHandler(w http.ResponseWriter, req *http.Request, _ []string) {
	MethodNotAllowed(w, req, nil)
}

// resolve returns the child node for seg.
//
// Candidates are considered in the usual order — literal before variable, then
// varPriority — but a node holding a real handler is preferred over a 405 stub.
// Without that rule, a stub planted for /user/:int would shadow a real POST
// handler registered at /user/:string, because Int outranks String.
func (p *pathTree) resolve(seg string) (*pathTree, bool) {
	var literal *pathTree
	if n := p.SubPaths[seg]; n != nil {
		literal = n
		if !n.MethodNotAllowed {
			return n, false
		}
	}

	var stub *pathTree
	for _, t := range varPriority {
		n := p.SubVariablesPaths[t]
		if n == nil || !matchType(seg, t) {
			continue
		}
		if !n.MethodNotAllowed {
			return n, true
		}
		if stub == nil {
			stub = n
		}
	}

	if stub != nil {
		return stub, true
	}
	return literal, false
}

func parseVarType(seg string) (Type, bool) {
	switch seg {
	case ":int":
		return Int, true
	case ":float":
		return Float, true
	case ":alpha":
		return Alpha, true
	case ":alphanum":
		return AlphaNum, true
	case ":uuid":
		return UUID, true
	case ":string":
		return String, true
	default:
		return 0, false
	}
}

func (p *pathTree) addRoute(paths []string, handler func(http.ResponseWriter, *http.Request, []string)) {
	p.route(paths, handler, false)
}

// addSubtree registers handler and marks the node as a subtree root. Used by
// Static; deliberately not exposed as a ":any" style path variable.
func (p *pathTree) addSubtree(paths []string, handler func(http.ResponseWriter, *http.Request, []string)) {
	p.route(paths, handler, true)
}

func (p *pathTree) route(paths []string, handler func(http.ResponseWriter, *http.Request, []string), subtree bool) {
	if len(paths) == 0 {
		p.Function = handler
		// Registering here replaces any 405 stub planted for this method.
		p.MethodNotAllowed = false
		if subtree {
			p.Subtree = true
		}
		return
	}
	seg := paths[0]
	if varType, ok := parseVarType(seg); ok {
		if p.SubVariablesPaths[varType] == nil {
			p.SubVariablesPaths[varType] = p.newNode(seg)
		}
		p.SubVariablesPaths[varType].route(paths[1:], handler, subtree)
		return
	}
	if p.SubPaths[seg] == nil {
		p.SubPaths[seg] = p.newNode(seg)
	}
	p.SubPaths[seg].route(paths[1:], handler, subtree)
}

func (p *pathTree) newNode(seg string) *pathTree {
	return &pathTree{
		Path:              seg,
		SubPaths:          make(map[string]*pathTree),
		SubVariablesPaths: make(map[Type]*pathTree),
	}
}

func (p *pathTree) getOrCreatePrefix(paths []string) *pathTree {
	node := p
	for _, seg := range paths {
		if node.SubPaths[seg] == nil {
			node.SubPaths[seg] = &pathTree{
				Path:              seg,
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
		}
		node = node.SubPaths[seg]
	}
	return node
}
