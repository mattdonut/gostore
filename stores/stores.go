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
	Name, Address, Phone, Website string
	Lat, Long float64
	Inventory []inventory.Item `datastore:"-" json:",omitempty`
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
	c.Infof("About to get the products")
	storeProducts, err := GetStoreProducts(c, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Infof("Got 'em! %v", storeProducts)
	store.Inventory = storeProducts
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
	storeProducts, err := GetStoreProducts(c, id);
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return		
	}
	if out, err := json.Marshal(storeProducts); err != nil {
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
		c.Infof("Error in Get for item %v : %v", item, err)
		return err
	}
	if err := <-itemLock; err != nil {
		c.Infof("Error in Get for item %v : %v", item, err)
		return err
	}

	item.StoreKey = datastore.NewKey(c, "store", "",item.StoreID, nil)
	item.ProductKey = datastore.NewKey(c, "product", "",item.ProductID, nil)
	_, err := datastore.Put(c, datastore.NewIncompleteKey(c,"inventory", nil), item)
	if err != nil {
		c.Infof("Error in Put: %v", err)
		return err
	}
	return nil
}

func GetStoreProducts(c appengine.Context, id int64) ([]inventory.Item, error) {
	q := datastore.NewQuery("inventory").Filter("StoreKey =", datastore.NewKey(c, "store", "",id, nil))
	c.Infof("Query made")
	count := 0
	items := make(chan inventory.Item)
	itemSlice := make([]inventory.Item, 0)
	for t := q.Run(c); ; {
		var item inventory.Item
		_, err := t.Next(&item)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		c.Infof("Item to fetch product: %v", item)
		go func(c appengine.Context, item inventory.Item) {
			var product products.Product
			if err := datastore.Get(c, item.ProductKey, &product); err != nil {
				items <- item
			}
			item.Product = product
			items <- item
		}(c, item)
		count++
	}
	for i := 0; i < count; i++ {
		itemSlice = append(itemSlice, <-items)
	}

	return itemSlice, nil
}