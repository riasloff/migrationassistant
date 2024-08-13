package dbworker

import (
	"database/sql"
	"fmt"
	"log"
)

type Migrations struct {
	Migrations []Migration
}

type Migration struct {
	ID   int64
	Name string
	Time string
}

func (m *Migration) GetMigrationStatus(db *sql.DB, id int) error {

	query := fmt.Sprintf("SELECT * FROM schema_migrations WHERE id = %d", id)
	row := db.QueryRow(query)

	if err := row.Scan(&m.ID, &m.Name, &m.Time); err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return err
	}
	return nil
}

func Ping(db *sql.DB) error {
	// ping
	err := db.Ping()
	if err != nil {
		log.Printf("error pinging postgres db: %s\n", err)
		return err
	}
	return nil
}

func ExecByte(db *sql.DB, query []byte) error {

	// exec
	_, err := db.Exec(string(query))
	if err != nil {
		log.Printf("error executing query: %s\n", err)
		return err
	}
	return nil
}

func ExecString(db *sql.DB, query string) error {

	// exec
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("error executing query: %s\n", err)
		return err
	}
	return nil
}
