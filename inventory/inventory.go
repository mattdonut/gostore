package inventory

import (
	"appengine/datastore"
)

type Item struct {
	Store *datastore.Key
	StoreID int64 `datastore:"-"`
	Product *datastore.Key
	ProductID int64 `datastore:"-"`
	Price float64
}