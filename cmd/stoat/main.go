package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/model"
)

func main() {
	dbPath := flag.String("db", "", "path to SQLite database file (e.g. ./mydb.sqlite)")
	dbDSN := flag.String("dsn", "", "PostgreSQL connection string (e.g. postgres://user:password@host:port/database)")
	flag.Parse()

	m := model.New()
	if *dbPath != "" {
		m.SetPendingConfig(database.Config{
			Name:   filepath.Base(*dbPath),
			DBMS:   database.DBMSSQLite,
			Values: map[string]string{"path": *dbPath},
		})
	} else if *dbDSN != "" {
		m.SetPendingConfig(database.Config{
			Name:   "postgres",
			DBMS:   database.DBMSPostgres,
			Values: map[string]string{"dsn": *dbDSN},
		})
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	m.SetConfig(cfg)

	program := tea.NewProgram(m)
	app, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
	if m, ok := app.(model.Model); ok {
		m.Close()
	}
}
