package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/davecheney/m/m"
	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	var (
		addr string
		dsn  string
	)
	flag.StringVar(&addr, "a", "127.0.0.1:8080", "address to listen")
	flag.StringVar(&dsn, "d", "", "data source name")
	flag.Parse()

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	svr := &http.Server{
		Addr:         addr,
		Handler:      m.New(db),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(svr.ListenAndServe())
}
