//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

type Shipping struct {
	ID        int32 `sql:"primary_key"`
	ProductID int32
	Country   string
	Service   string
	Price     string
}
