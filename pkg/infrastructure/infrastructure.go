package infrastructure

import (
	"errors"
	"fmt"
	"log"

	_ "github.com/bsn-si/IPEHR-gateway/src/pkg/aqlquerier" //nolint
	"github.com/bsn-si/IPEHR-stat/internal/repository"
	"github.com/bsn-si/IPEHR-stat/pkg/config"
	"github.com/bsn-si/IPEHR-stat/pkg/service/stat"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/sqlite3"
	_ "github.com/golang-migrate/migrate/source/file" //nolint
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" //nolint
)

type Infra struct {
	DB        *sqlx.DB
	EthClient *ethclient.Client
	AqlDB     *sqlx.DB

	StatsRepo *repository.StatsStorage
	ChunkRepo *repository.IndexStorage
	Service   *stat.Service
}

func New(cfg *config.Config) *Infra {
	ehtClient, err := ethclient.Dial(cfg.Sync.Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("sqlite3", cfg.LocalDB.Path)
	if err != nil {
		log.Fatal("sql.Open error: ", err)
	}

	if err := migrateDB(db, cfg.LocalDB.Migrations); err != nil {
		log.Fatal(err)
	}

	aqlDB, err := sqlx.Open("aql", "")
	if err != nil {
		log.Fatal(err)
	}

	statsRepo := repository.NetStatsSotrage(db)
	svc := stat.NewService(statsRepo)

	return &Infra{
		DB:        db,
		EthClient: ehtClient,
		AqlDB:     aqlDB,
		StatsRepo: statsRepo,
		ChunkRepo: repository.NewIndexStorage(db),
		Service:   svc,
	}
}

func (i *Infra) Close() {
	i.DB.Close()
	i.AqlDB.Close()
}

func migrateDB(db *sqlx.DB, migrations string) error {
	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("sqlite3.WithInstance error: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrations, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("migrate.NewWithDatabaseInstance error: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate.Up() error: %w", err)
	}

	return nil
}
