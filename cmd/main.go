package main

// Generating swagger doc spec//
//go:generate swag fmt -g ../pkg/api/api.go
//go:generate swag init --parseDependency -g ../cmd/main.go -o ../pkg/api/docs

import (
	"flag"

	"github.com/bsn-si/IPEHR-stat/pkg/api"
	_ "github.com/bsn-si/IPEHR-stat/pkg/api/docs"
	"github.com/bsn-si/IPEHR-stat/pkg/config"
	"github.com/bsn-si/IPEHR-stat/pkg/infrastructure"
	"github.com/bsn-si/IPEHR-stat/pkg/service/syncer"

	"github.com/gin-contrib/cors"
)

// @title        IPEHR Stat API
// @version      0.1
// @description  IPEHR Stat is an open API service for providing public statistics from the IPEHR system.

// @contact.name   API Support
// @contact.url    https://bsn.si/blockchain
// @contact.email  support@bsn.si

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      stat.ipehr.org
// host      localhost:8080
// @BasePath  /

func main() {
	cfgPath := flag.String("config", "./config.json", "config file path")

	flag.Parse()

	cfg := config.New(*cfgPath)

	infra := infrastructure.New(cfg)
	defer infra.Close()

	syncer.New(
		infra.DB,
		infra.EthClient,
		syncer.Config(cfg.Sync),
	).Start()

	a := api.New(cfg, infra).Build()

	//TODO complete CORS config
	a.Use(cors.Default())

	err := a.Run(cfg.Host)
	if err != nil {
		panic(err)
	}
}
