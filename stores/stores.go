package stores

import (
	"fmt"
	"net/http"
	"pathways"
	"inventory"
	"products"
	"encoding/json"
    "appengine"
    "appengine/datastore"
)

type Store struct {
	Name string
	Lat, Long float64
}

func Pathway() *pathways.Edge {
	return &pathways.Edge{Pattern: "stores",
						Get: new(StoreList),
						Post: new(StorePost),
						Forks: []*pathways.Edge{&pathways.Edge{Pattern: "<id>", 
																Get: new(StoreGet), 
																Forks:[]*pathways.Edge{&pathways.Edge{Pattern:"inventory",
																										Get:new(InventoryList),
																										Post:new(InventoryPost),},
																									},
																},
						},}
}

type StorePost int8
type StoreList int8
type StoreGet int8
type InventoryList int8
type InventoryPost int8

func (sp *StorePost) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	c := appengine.NewContext(r.Request)
	var stores []Store
	if err := json.Unmarshal(r.BodyBytes, &stores); err != nil {
		fmt.Fprintf(w, "json Unmarshal error: %+v", err)
	}
	key, err := datastore.Put(c, datastore.NewIncompleteKey(c,"store", nil), &(stores[0]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"key\":\"%d\"}", key.IntID())
}

func (sl *StoreList) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	c := appengine.NewContext(r.Request)
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	storemap := make(map[string] Store)
	q := datastore.NewQuery("store")
	for t := q.Run(c); ; {
		var store Store
		key, err := t.Next(&store)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		storemap[fmt.Sprintf("%d",key.IntID())] = store
	}
	if out, err := json.Marshal(storemap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w,string(out))
	}
}

func (sg *StoreGet) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	c := appengine.NewContext(r.Request)
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	var store Store
	var id int64
	fmt.Sscan( r.Request.Form.Get("id"), &id)
	if err := datastore.Get(c, datastore.NewKey(c, "store", "",id, nil), &store); err != nil {
		http.Error(w, "No store with that id found", http.StatusNotFound)
		return
	}
	if out, err := json.Marshal(store); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w,string(out))
	}
}

func (il *InventoryList) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	c := appengine.NewContext(r.Request)
	var id int64
	fmt.Sscan( r.Request.Form.Get("id"), &id)
	q := datastore.NewQuery("inventory").Filter("Store =", datastore.NewKey(c, "store", "",id, nil))
	keys := make([]*datastore.Key,1000)
	i := 0
	for t := q.Run(c); ; {
		var item inventory.Item
		_, err := t.Next(&item)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		keys[i] = item.Product
		i++
	}
	keys = keys[:i]
	products := make([]products.Product,i)
	if err := datastore.GetMulti(c, keys, products); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if out, err := json.Marshal(products); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		fmt.Fprint(w,string(out))
	}

}
func (il *InventoryPost) EdgeHandler(w http.ResponseWriter, r *pathways.PathRequest) {
	headers := w.Header()
	headers.Set("Content-Type","application/json")
	c := appengine.NewContext(r.Request)
	var id int64
	fmt.Sscan( r.Request.Form.Get("id"), &id)
	var items []inventory.Item
	if err := json.Unmarshal(r.BodyBytes, &items); err != nil {
		fmt.Fprintf(w, "json Unmarshal error: %+v", err)
	}
	putLock := make(chan error, len(items))
	for _, item := range items {
		item.StoreID = id
		go func() {
			putLock <- PutInventory(c, &item)
		}()
	}
	for i:=0;i<len(items);i++ {
		if err := <-putLock; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	fmt.Fprint(w,"{\"success\":true}")
}
func PutInventory(c appengine.Context, item *inventory.Item) error {
	itemLock := make(chan error, 2)
	var store Store
	var product products.Product
	go func() {
		itemLock <- datastore.Get(c, datastore.NewKey(c, "store", "",item.StoreID, nil), &store)
	}()
	go func() {
		itemLock <- datastore.Get(c, datastore.NewKey(c, "product", "",item.ProductID, nil), &product)
	}()
	if err := <-itemLock; err != nil {
		c.Infof("Error in Get: %v", err)
		return err
	}
	if err := <-itemLock; err != nil {
		c.Infof("Error in Get: %v", err)
		return err
	}

	item.Store = datastore.NewKey(c, "store", "",item.StoreID, nil)
	item.Product = datastore.NewKey(c, "product", "",item.ProductID, nil)
	_, err := datastore.Put(c, datastore.NewIncompleteKey(c,"inventory", nil), item)
	if err != nil {
		c.Infof("Error in Put: %v", err)
		return err
	}
	return nil
}