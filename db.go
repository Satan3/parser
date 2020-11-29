package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func newDb(config *Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func (p *Parser) insertLots() {
	queryTemplate := "INSERT INTO lots (lot, year, vin, buyNow) VALUES "
	valuesStr := "(?, ?, ?, ?)"

	values := make([]interface{}, 0, len(p.lots)*4)
	valuesStrSlice := make([]string, 0, len(p.lots))

	for _, lot := range p.lots {
		valuesStrSlice = append(valuesStrSlice, valuesStr)

		year, err := strconv.Atoi(lot.Year)
		if err != nil {
			year = 0
		}

		var buyNow bool
		var intBuyNow int
		if lot.BuyNow == "" {
			buyNow = false
		} else {
			if buyNow, err = strconv.ParseBool(lot.BuyNow); err != nil {
				buyNow = false
			}
		}

		if buyNow {
			intBuyNow = 1
		}

		values = append(values, lot.Lot, year, lot.Vin, intBuyNow)
	}

	sqlStr := queryTemplate + strings.Join(valuesStrSlice, ", ")
	stmt, _ := p.db.Prepare(sqlStr)
	_, err := stmt.Exec(values...)
	if err != nil {
		log.Fatal(err)
	}
}

func (p *Parser) getLotsFromDB() {
	var id int
	var lotLink string
	var year int
	var vin string
	var buyNow int
	var createdAt string

	rawLots, err := p.db.Query("SELECT * FROM lots")
	if err != nil {
		log.Fatal(err)
	}
	defer rawLots.Close()

	for rawLots.Next() {
		if err := rawLots.Scan(&id, &lotLink, &year, &vin, &buyNow, &createdAt); err != nil {
			fmt.Println("Ошибка извлечения лота из базы данных")
		}
		lot := Lot{
			Lot:    lotLink,
			Year:   strconv.Itoa(year),
			Vin:    vin,
			BuyNow: strconv.Itoa(buyNow),
		}
		p.lots = append(p.lots, lot)
	}
}

func (p *Parser) clearOldLots() {
	if _, err := p.db.Exec("DELETE FROM lots"); err != nil {
		log.Fatal(err)
	}
}
