package router

import (
	"net/http"
)

func isInt(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case c == '-':
			if i != 0 {
				return false
			}
			if len(s) == 1 {
				return false
			}
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

func isFloat(s string) bool {
	if s == "" {
		return false
	}
	hasDot := false
	for i, c := range s {
		switch {
		case c == '-':
			if i != 0 {
				return false
			}
			if len(s) == 1 {
				return false
			}
		case c == '.':
			if hasDot {
				return false
			}
			hasDot = true
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return hasDot
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

func isAlphaNum(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch {
		case c == '-':
			if i != 8 && i != 13 && i != 18 && i != 23 {
				return false
			}
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'):
		default:
			return false
		}
	}
	return true
}

func cleanPaths(paths []string) []string {
	clean := paths[:0]
	for _, s := range paths {
		if s != "" {
			clean = append(clean, s)
		}
	}
	return clean
}

// matchType reports whether seg satisfies the variable type t.
func matchType(seg string, t Type) bool {
	switch t {
	case Int:
		return isInt(seg)
	case Float:
		return isFloat(seg)
	case Alpha:
		return isAlpha(seg)
	case AlphaNum:
		return isAlphaNum(seg)
	case UUID:
		return isUUID(seg)
	case String:
		return true
	default:
		return false
	}
}

// varPriority is the order in which variable types are attempted at a node.
// More specific types must be tried before the catch-all String.
var varPriority = [...]Type{Int, Float, UUID, Alpha, AlphaNum, String}

// routeMatch is a resolved path. nodes runs root to leaf so middleware
// executes in registration order.
type routeMatch struct {
	nodes   []*pathTree
	handler func(http.ResponseWriter, *http.Request, []string)
	vars    []string

	// notAllowed reports that the resolved node is a 405 stub rather than a real
	// route, so the router can attach an Allow header before replying.
	notAllowed bool
}

// exec runs middleware then the handler, reporting whether it got that far.
func (m *routeMatch) exec(w http.ResponseWriter, r *http.Request) bool {
	for _, node := range m.nodes {
		for _, mw := range node.Middleware {
			if !mw(w, r, m.vars) {
				return false
			}
		}
	}
	if m.handler == nil {
		return false
	}
	m.handler(w, r, m.vars)
	return true
}

// match resolves path without executing anything, which is how 405 detection
// probes the other method trees.
func (p *pathTree) match(path string) (*routeMatch, bool) {
	m := &routeMatch{nodes: make([]*pathTree, 0, 8)}

	if path == "" || path == "/" {
		m.nodes = append(m.nodes, p)
		m.handler = p.Function
		m.notAllowed = p.MethodNotAllowed
		return m, p.Function != nil
	}

	node := p
	m.nodes = append(m.nodes, node)
	i, n := 0, len(path)

	for i < n {
		for i < n && path[i] == '/' {
			i++
		}
		if i >= n {
			break
		}

		start := i
		for i < n && path[i] != '/' {
			i++
		}
		seg := path[start:i]

		// A literal segment outranks a variable, and a real handler outranks a
		// 405 stub; resolve applies both rules.
		if next, isVar := node.resolve(seg); next != nil {
			if isVar {
				m.vars = append(m.vars, seg)
			}
			node = next
			m.nodes = append(m.nodes, node)
			continue
		}

		// A subtree root takes whatever remains. Checked after segment
		// matching so an explicit sibling route still wins.
		if node.Subtree && node.Function != nil {
			m.handler = node.Function
			m.notAllowed = node.MethodNotAllowed
			return m, true
		}

		return nil, false
	}

	m.handler = node.Function
	m.notAllowed = node.MethodNotAllowed
	return m, node.Function != nil
}

// pathExists reports whether path resolves to a real route. A 405 stub is not a
// route, so it does not count.
func (p *pathTree) pathExists(path string) bool {
	m, ok := p.match(path)
	return ok && m != nil && !m.notAllowed
}
