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

func cleanPaths(paths []string) []string {
	clean := paths[:0]
	for _, s := range paths {
		if s != "" {
			clean = append(clean, s)
		}
	}
	return clean
}

func (p *pathTree) matchPath(path string, w http.ResponseWriter, r *http.Request) (func(http.ResponseWriter, *http.Request, []string), []string, bool) {
	if path == "" || path == "/" {
		for _, mw := range p.Middleware {
			if !mw(w, r, nil) {
				return nil, nil, false
			}
		}
		return p.Function, nil, true
	}

	node := p
	var ctx []string
	i := 0
	n := len(path)

	// 执行根节点中间件
	for _, mw := range node.Middleware {
		if !mw(w, r, ctx) {
			return nil, nil, false
		}
	}

	for i < n {
		for i < n && path[i] == '/' {
			i++
		}
		if i >= n {
			return node.Function, ctx, true
		}

		start := i
		for i < n && path[i] != '/' {
			i++
		}
		seg := path[start:i]

		if next := node.SubPaths[seg]; next != nil {
			node = next
			for _, mw := range node.Middleware {
				if !mw(w, r, ctx) {
					return nil, nil, false
				}
			}
			continue
		}
		if node.SubVariablesPaths[Int] != nil && isInt(seg) {
			ctx = append(ctx, seg)
			node = node.SubVariablesPaths[Int]
			for _, mw := range node.Middleware {
				if !mw(w, r, ctx) {
					return nil, nil, false
				}
			}
			continue
		}
		if node.SubVariablesPaths[String] != nil {
			ctx = append(ctx, seg)
			node = node.SubVariablesPaths[String]
			for _, mw := range node.Middleware {
				if !mw(w, r, ctx) {
					return nil, nil, false
				}
			}
			continue
		}
		return NotFound, nil, true
	}
	return node.Function, ctx, true
}

func (p *pathTree) visitPath(path string, w http.ResponseWriter, r *http.Request) (func(http.ResponseWriter, *http.Request, []string), []string, bool) {
	return p.matchPath(path, w, r)
}

func (p *pathTree) Visit(w http.ResponseWriter, r *http.Request) {
	f, c, ok := p.matchPath(r.URL.Path, w, r)
	if f != nil && ok {
		f(w, r, c)
	}
}
