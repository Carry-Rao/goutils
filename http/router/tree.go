package router

import (
	"net/http"
)

type pathTree struct {
	Path              string
	Function          func(http.ResponseWriter, *http.Request, []string)
	SubPaths          map[string]*pathTree
	SubVariablesPaths map[Type]*pathTree
}

func (p *pathTree) addRoute(paths []string, handler func(http.ResponseWriter, *http.Request, []string)) {
	if len(paths) == 0 {
		p.Function = handler
		return
	}
	seg := paths[0]
	switch seg {
	case ":int":
		if p.SubVariablesPaths[Int] == nil {
			p.SubVariablesPaths[Int] = &pathTree{
				Path:              seg,
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
		}
		p.SubVariablesPaths[Int].addRoute(paths[1:], handler)
	case ":string":
		if p.SubVariablesPaths[String] == nil {
			p.SubVariablesPaths[String] = &pathTree{
				Path:              seg,
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
		}
		p.SubVariablesPaths[String].addRoute(paths[1:], handler)
	default:
		if p.SubPaths[seg] == nil {
			p.SubPaths[seg] = &pathTree{
				Path:              seg,
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
		}
		p.SubPaths[seg].addRoute(paths[1:], handler)
	}
}
