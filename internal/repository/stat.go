package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	TableNamePatients  = "stat_patients"
	TableNameDocuments = "stat_documents"
)

type StatsStorage struct {
	db *sqlx.DB
}

func NetStatsSotrage(db *sqlx.DB) *StatsStorage {
	return &StatsStorage{
		db: db,
	}
}

func (repo *StatsStorage) StatPatientsCountGet(start, end int64) (uint64, error) {
	const query = `SELECT COALESCE(SUM(count), 0)
			  FROM ` + TableNamePatients + `
			  WHERE timestamp_day >= ? AND timestamp_day < ?`

	var count uint64
	if err := repo.db.Get(&count, query, start, end); err != nil {
		return 0, fmt.Errorf("cannot get patients count: %w", err)
	}

	return count, nil
}

func (repo *StatsStorage) StatPatientsCountIncrement(timestamp time.Time) error {
	const query = `INSERT INTO ` + TableNamePatients + ` (timestamp_day, count) VALUES (?, 1)
			  ON CONFLICT (timestamp_day) DO UPDATE SET 
			  count = count + 1`

	timestamp = timestamp.Truncate(time.Hour * 24)

	_, err := repo.db.Exec(query, timestamp.Unix())
	if err != nil {
		return fmt.Errorf("StatPatientsCountIncrement error: %w query: %s timestamp: %d", err, query, timestamp.Unix())
	}

	return nil
}

func (repo *StatsStorage) StatDocumentsCountGet(start, end int64) (uint64, error) {
	const query = `SELECT COALESCE(SUM(count), 0)
			  FROM ` + TableNameDocuments + ` 
			  WHERE timestamp_day >= ? AND timestamp_day < ?`

	var count uint64
	if err := repo.db.Get(&count, query, start, end); err != nil {
		return 0, fmt.Errorf("cannot get documents count: %w", err)
	}

	return count, nil
}

func (repo *StatsStorage) StatDocumentsCountIncrement(timestamp time.Time) error {
	const query = `INSERT INTO ` + TableNameDocuments + ` (timestamp_day, count) VALUES (?, 1)
			  ON CONFLICT (timestamp_day) DO UPDATE SET 
			  count = count + 1`

	timestamp = timestamp.Truncate(time.Hour * 24)
	if _, err := repo.db.Exec(query, timestamp.Unix()); err != nil {
		return fmt.Errorf("StatPatientsCountIncrement error: %w query: %s timestamp: %d", err, query, timestamp.Unix())
	}

	return nil
}

func (repo *StatsStorage) SyncLastBlockGet() (uint64, error) {
	const query = `SELECT value FROM sync WHERE key = 'last_synced_block' LIMIT 1`

	var count uint64
	if err := repo.db.Get(&count, query); err != nil {
		return 0, fmt.Errorf("cannot get last block: %w", err)
	}

	return count, nil
}

func (repo *StatsStorage) SyncLastBlockSet(lastSyncedBlock uint64) error {
	const query = `INSERT INTO sync (key, value) VALUES ('last_synced_block', $1)
			  ON CONFLICT (key) DO UPDATE SET 
			  value = $1`

	_, err := repo.db.Exec(query, lastSyncedBlock)
	if err != nil {
		return fmt.Errorf("cannot add or update 'last_synced_block': %w", err)
	}

	return nil
}
