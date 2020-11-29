package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

var (
	actionType string
)

func init() {
	flag.StringVar(
		&actionType,
		"type",
		"!parse",
		"Go parser action type",
	)
}

func main() {
	now := time.Now()
	flag.Parse()
	db, err := newDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	parser := NewParser(db)
	defer parser.cancel()

	if actionType != "parse" {
		getBuyNow(parser)
		return
	}
	parseLots(parser)
	fmt.Printf("Время выполнения %g секунд\n", time.Now().Sub(now).Seconds())
}

func newDb() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/parser")
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func parseLots(parser *Parser) {
	parser.getAuctions()
	parser.getAllLots()
	parser.clearOldLots()
	parser.insertLots()
}

func getBuyNow(parser *Parser) {
	parser.getLotsFromDB()
	if len(parser.lots) == 0 {
		fmt.Println("Отсутствуют лоты для проверки")
		return
	}
	parser.getBuyNowLots()
	//parser.clearOldLots()
	parser.insertLots()
}
