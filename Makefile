include .env
all:
	go build -v

parse:
	./parser --type=parse

buyNow:
	./parser

.PHONY: migrate
migrate:
	migrate -path ./db/migrations/ -database "mysql://${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST})/${DB_NAME}" up
