// Package models contains application specific entities.
package models

import (
	"github.com/go-ozzo/ozzo-validation"
)

// Request holds specific application settings linked to an Account.
type Request struct {
	ID          int    `json:"-"`
	Github      string `json:"github"`
	RequestDate int64  `json:"request_date,omitempty"`
}

//// BeforeInsert hook executed before database insert operation.
//func (p *Request) BeforeInsert(db orm.DB) error {
//	p.Requested = time.Now().Unix()
//	return nil
//}

// BeforeUpdate hook executed before database update operation.
//func (p *Request) BeforeUpdate(db orm.DB) error {
//	p.Requested = time.Now().Unix()
//	return p.Validate()
//}

// Validate validates Request struct and returns validation errors.
func (p *Request) Validate() error {

	return validation.ValidateStruct(p,
		validation.Field(&p.RequestDate, validation.Required, validation.Min(p.RequestDate+86400)),
	)
}
