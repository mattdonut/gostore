package pathways

import (
	"net/http"
)

type PathRequest struct {
	Request *http.Request
	BodyBytes []byte
}

func (pr *PathRequest) ParseBodyBytes() *PathRequest {
	myBytes := make([]byte, 1000, 1000)
	num, _ := pr.Request.Body.Read(myBytes)
	pr.BodyBytes = myBytes[0:num]
	return pr
}

type Destination interface {
	EdgeHandler(w http.ResponseWriter, r *PathRequest)
}

type Edge struct {
	Pattern string
	Get Destination
	Put Destination
	Post Destination
	Delete Destination
	Forks []*Edge
}

type DeadEnd int

func EdgePattern(p string) *Edge {
	return &Edge{Pattern: p,
				Forks: nil,
	}
}

func (e *Edge) MatchPattern(s string, r *PathRequest) bool {
	if e.Pattern[0] == '<' {
		r.Request.Form.Add(e.Pattern[1:len(e.Pattern) -1], s)
		return true
	} else {
		if e.Pattern == s {
			return true
		}
	}
	return false
}

func (e *Edge) SetForks(edges []*Edge) *Edge {
	e.Forks = edges
	return e
}

func (de *DeadEnd) EdgeHandler(w http.ResponseWriter, r *PathRequest) {
	http.Error(w, "No handler for that pathway", http.StatusNotFound)
}