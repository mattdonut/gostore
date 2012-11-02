package hello

import (
    "fmt"
    "strings"
    "net/http"
    "appengine"
    "stores"
    "products"
    "pathways"
)

type Store struct {
	Name string
	Lat, Long float64
}

func init() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/api/", startpath)
}

func startpath(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	basepath := pathways.EdgePattern("api").SetForks([]*pathways.Edge{stores.Pathway(), products.Pathway()})
	steps := strings.Split(r.URL.Path, "/")[2:]
	endpoint := pathways.EdgePattern("/")
	endpoint = basepath
	r.ParseForm()
	pathRequest := (&pathways.PathRequest{Request:r}).ParseBodyBytes()
	// go through each path segment and see if any forks match
	for _, segment := range steps {
		c.Infof("Step processing: %v", segment)
		matched := false
		for _, fork := range endpoint.Forks {
			if fork.MatchPattern(segment, pathRequest) {
				endpoint = fork
				matched = true
				break
			}
		}
		if matched == false {
			break
		}
	}

	var destination pathways.Destination
	switch r.Method {
		case "GET":
			destination = endpoint.Get
		case "POST":
			destination = endpoint.Post
	}
	if destination != nil {
		destination.EdgeHandler(w,pathRequest)
	} else {
		new(pathways.DeadEnd).EdgeHandler(w,pathRequest)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	//c := appengine.NewContext(r)
	text := r.FormValue("json")
    fmt.Fprint(w, "Hello, world! You gave me %+v", text)
}