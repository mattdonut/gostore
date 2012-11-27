package inventory

import (
	"products"
	"appengine/datastore"
)

type Item struct {
	StoreKey *datastore.Key `json:"-"`
	StoreID int64 `datastore:"-" json:"-"`
	ProductKey *datastore.Key `json:"-"`
	Product products.Product `datastore:"-"`
	ProductID int64 `datastore:"-" json:",omitempty"`
	Price float64
	Quantity int64
}