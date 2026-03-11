package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/config"
	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/database/provider"
	"github.com/jxdones/stoat/internal/ui/datasource"
	"github.com/jxdones/stoat/internal/ui/model"
)

func main() {
	dbPath := flag.String("db", "", "path to SQLite database file (e.g. ./mydb.sqlite)")
	flag.Parse()

	m := model.New()

	if *dbPath != "" {
		path := *dbPath
		dbCfg := database.Config{
			Name:   filepath.Base(path),
			DBMS:   database.DBMSSQLite,
			Values: map[string]string{"path": path},
		}
		conn, err := provider.FromConfig(dbCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open database: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close()
		m.SetDataSource(datasource.FromConnection(conn))
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	m.SetConfig(cfg)

	program := tea.NewProgram(m)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(1)
	}
}
