package repository

import "github.com/jmoiron/sqlx"

type IndexStorage struct {
	db *sqlx.DB
}

func NewIndexStorage(db *sqlx.DB) *IndexStorage {
	return &IndexStorage{
		db: db,
	}
}
