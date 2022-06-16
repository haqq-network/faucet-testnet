package database

import (
	"errors"
	"time"

	"github.com/go-pg/pg"
	"github.com/haqq-network/faucet-testnet/models"
)

// RequestStore implements database operations for request management.
type RequestStore struct {
	db *pg.DB
}

// NewRequestStore returns a RequestStore implementation.
func NewRequestStore(db *pg.DB) *RequestStore {
	return &RequestStore{
		db: db,
	}
}

// Get gets a request by github ID.
func (s *RequestStore) Get(github string) (*models.Request, error) {
	p := models.Request{Github: github}
	_, err := s.db.Model(&p).
		Where("github = ?", github).
		SelectAndCount()

	return &p, err
}

// Insert a request with github id
func (s *RequestStore) Insert(github string) (*models.Request, error) {
	p := models.Request{Github: github}
	count, _ := s.db.Model(&p).
		Where("github = ?", github).
		SelectAndCount()
	if count == 0 {
		p = models.Request{Github: github, RequestDate: time.Now().Unix()}
		_, err := s.db.Model(&p).Insert()
		return &p, err
	}

	if time.Now().Unix() >= (p.RequestDate + 86400) {
		p.RequestDate = time.Now().Unix()
		err := s.Update(&p)
		return &p, err
	}
	return nil, errors.New("account already requested tokens")
}

// Update updates profile.
func (s *RequestStore) Update(p *models.Request) error {
	err := s.db.Update(p)
	return err
}
