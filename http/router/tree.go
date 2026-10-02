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
