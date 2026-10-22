//go:build ignore

// 删掉已废弃的 station_agents 表。工位智能体改由 server/agents/<通道>/AGENT.md 固定提供。
//
//	cd server && go run scripts/drop-station-agents.go
//	cd server && go run scripts/drop-station-agents.go -config etc/ade-dev.yaml
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"chengzi-agent-code-server/internal/config"

	_ "github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	configFile := flag.String("config", "etc/ade.yaml", "配置文件路径")
	flag.Parse()

	var cfg config.Config
	conf.MustLoad(*configFile, &cfg)
	if err := cfg.Validate(); err != nil {
		exit(err)
	}

	driver := strings.ToLower(cfg.Database.Driver)
	var (
		db  *sql.DB
		err error
	)
	switch driver {
	case "sqlite":
		path := strings.TrimSpace(cfg.Database.SqlitePath)
		if path == "" {
			path = "data/app.db"
		}
		if _, statErr := os.Stat(path); statErr != nil {
			fmt.Printf("数据库文件不存在（%s），无需删表\n", path)
			return
		}
		db, err = sql.Open("sqlite", path)
	case "mysql":
		db, err = sql.Open("mysql", cfg.Database.DSN())
	default:
		exit(fmt.Errorf("不支持的数据库驱动：%s", driver))
	}
	if err != nil {
		exit(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		exit(fmt.Errorf("连接数据库失败: %w", err))
	}

	present, err := tableExists(db, driver)
	if err != nil {
		exit(err)
	}
	if !present {
		fmt.Println("station_agents 不存在，无需删除")
		return
	}
	if _, err = db.Exec("DROP TABLE IF EXISTS station_agents"); err != nil {
		exit(fmt.Errorf("删除 station_agents 失败: %w", err))
	}
	fmt.Println("已删除表 station_agents")
}

func tableExists(db *sql.DB, driver string) (bool, error) {
	var query string
	switch driver {
	case "sqlite":
		query = `SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'station_agents'`
	default:
		query = `SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'station_agents'`
	}
	var one int
	err := db.QueryRow(query).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查 station_agents 失败: %w", err)
	}
	return true, nil
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
