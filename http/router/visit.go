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

func (p *pathTree) matchPath(path string) (func(http.ResponseWriter, *http.Request, []string), []string) {
	if path == "" || path == "/" {
		return p.Function, nil
	}

	node := p
	var ctx []string
	i := 0
	n := len(path)

	for i < n {
		for i < n && path[i] == '/' {
			i++
		}
		if i >= n {
			return node.Function, ctx
		}

		start := i
		for i < n && path[i] != '/' {
			i++
		}
		seg := path[start:i]

		if next := node.SubPaths[seg]; next != nil {
			node = next
			continue
		}
		if node.SubVariablesPaths[Int] != nil && isInt(seg) {
			ctx = append(ctx, seg)
			node = node.SubVariablesPaths[Int]
			continue
		}
		if node.SubVariablesPaths[String] != nil {
			ctx = append(ctx, seg)
			node = node.SubVariablesPaths[String]
			continue
		}
		return NotFound, nil
	}
	return node.Function, ctx
}

func (p *pathTree) visitPath(path string) (func(http.ResponseWriter, *http.Request, []string), []string) {
	return p.matchPath(path)
}

func (p *pathTree) Visit(w http.ResponseWriter, r *http.Request) {
	f, c := p.matchPath(r.URL.Path)
	f(w, r, c)
}
