package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/iamdanhart/te-live/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	label := flag.String("label", "", "identifier for this host user (e.g. dan, venue)")
	pass := flag.String("passcode", "", "plaintext passcode to hash and store")
	flag.Parse()

	if *label == "" || *pass == "" {
		log.Fatal("both -label and -passcode are required")
	}

	cfg := config.Load()
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*pass), 12)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO telive.host_users (label, passcode_hash) VALUES ($1, $2)`,
		*label, string(hash),
	)
	if err != nil {
		log.Fatalf("insert: %v", err)
	}
	fmt.Printf("host user %q created\n", *label)
}