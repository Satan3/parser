package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

var (
	actionType string
	configPath string
)

func init() {
	flag.StringVar(
		&actionType,
		"type",
		"!parse",
		"Go parser action type",
	)
	flag.StringVar(
		&configPath,
		"path",
		"config.toml",
		"Path to config file",
	)
}

func main() {
	now := time.Now()
	flag.Parse()

	config := NewConfig()
	_, err := toml.DecodeFile(configPath, config)

	db, err := newDb(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	parser := NewParser(db)
	defer parser.cancel()

	if actionType != "parse" {
		parser.actualizeBuyNow()
		return
	}
	parser.parse()
	fmt.Printf("Время выполнения %g секунд\n", time.Now().Sub(now).Seconds())
}
