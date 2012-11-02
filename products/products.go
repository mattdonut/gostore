package products

import (
	"fmt"
	"net/http"
	"pathways"
	"encoding/json"
    "appengine"
    "appengine/datastore"
)

type Product struct {
	Name string
	Type string
}

func Pathway() *pathways.Edge {
	return &pathways.Edge{Pattern: "products",
						Get: new(ProductList),
						Post: new(ProductPost),
						Forks: nil,
						}
}

type ProductPost int8
type ProductList int8

func (sp *ProductPost) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	c := appengine.NewContext(r.Request)
	var products []Product
	if err := json.Unmarshal(r.BodyBytes, &products); err != nil {
		fmt.Fprintf(w, "json Unmarshal error: %+v", err)
	}
	key, err := datastore.Put(c, datastore.NewIncompleteKey(c,"product", nil), &(products[0]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"key\":\"%d\"}", key.IntID())
}

func (sl *ProductList) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	c := appengine.NewContext(r.Request)
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	productmap := make(map[string] Product)
	q := datastore.NewQuery("product")
	for t := q.Run(c); ; {
		var product Product
		key, err := t.Next(&product)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		productmap[fmt.Sprintf("%d",key.IntID())] = product
	}
	if out, err := json.Marshal(productmap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w,string(out))
	}
}