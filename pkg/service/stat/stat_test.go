package stat

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/bsn-si/IPEHR-stat/pkg/config"
	"github.com/bsn-si/IPEHR-stat/pkg/localDB"
)

var testDBPath = "/tmp/ipehrstat_test.db"

func TestCheckCounting(t *testing.T) {
	db := prepare(t)
	defer tearDown(t, db)

	ts := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 31; i++ {
		db.StatPatientsCountIncrement(ts)
		ts = ts.Add(time.Hour * 24)
	}

	service := NewService(db)

	tests := []struct {
		name     string
		period   string
		expected uint64
	}{
		{
			"1. expected 0 for old period",
			"202001",
			0,
		},
		{
			"2. expected 31 for empty period",
			"",
			31,
		},
		{
			"3. expected 31 for correct period",
			"202201",
			31,
		},
		{
			"4. expected 0 for period in future",
			"202301",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := service.GetPatientsCount(tt.period)
			if err != nil {
				t.Fatal(err)
			}

			if count != tt.expected {
				t.Fatalf("Expected %d, received %d", tt.expected, count)
			}
		})
	}
}

func prepare(t *testing.T) *localDB.DB {
	t.Helper()

	cfgPath := flag.String("config", "./config.json", "config file path")

	flag.Parse()

	cfg := config.New(*cfgPath)

	_, err := os.Create(testDBPath)
	if err != nil {
		t.Fatal(err)
	}

	db := localDB.New(testDBPath)

	err = db.Migrate(cfg.LocalDB.Migrations)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func tearDown(t *testing.T, db *localDB.DB) {
	db.Close()

	err := os.Remove(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
}
