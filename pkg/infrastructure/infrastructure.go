package infrastructure

import (
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"

	"github.com/bsn-si/IPEHR-stat/pkg/config"

	_ "github.com/bsn-si/IPEHR-gateway/src/pkg/aqlquerier" //nolint
	"github.com/bsn-si/IPEHR-stat/pkg/localDB"
)

type Infra struct {
	DB        *localDB.DB
	EthClient *ethclient.Client
	AqlDB     *sqlx.DB
}

func New(cfg *config.Config) *Infra {
	ehtClient, err := ethclient.Dial(cfg.Sync.Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	db := localDB.New(cfg.LocalDB.Path)
	if err := db.Migrate(cfg.LocalDB.Migrations); err != nil {
		log.Fatal(err)
	}

	aqlDB, err := sqlx.Open("aql", "")
	if err != nil {
		log.Fatal(err)
	}

	return &Infra{
		DB:        db,
		EthClient: ehtClient,
		AqlDB:     aqlDB,
	}
}

func (i *Infra) Close() {
	i.DB.Close()
	i.AqlDB.Close()
}
