package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	smtp struct {
		host     string
		port     int
		username string
		password string
	}
	frontEnd string
}

type application struct {
	config   config
	infoLog  *log.Logger
	errorLog *log.Logger
	version  string
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	app.infoLog.Printf("Starting invoicing microservice on port %d\n", app.config.port)
	return srv.ListenAndServe()
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 5000, "Server port to listen on")
	flag.StringVar(&cfg.frontEnd, "frontend", "http://localhost:4000", "URL to front-end app")
	flag.Parse()

	cfg.smtp.host = os.Getenv("SMTP_HOST")
	var err error
	cfg.smtp.port, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		cfg.smtp.port = 1025
	}
	cfg.smtp.username = os.Getenv("SMTP_USER")
	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	app := &application{
		config:   cfg,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
	}
	err = app.CreateDirIfNotExists("./invoices")
	if err != nil {
		log.Fatal(err)
	}
	err = app.serve()
	if err != nil {
		log.Fatal(err)
	}
}
