package main

import (
	"database/sql"
	"fmt"
	"log"
	"migrationassistant/internal/configreader"
	"migrationassistant/internal/dbworker"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Available struct {
	ID     int
	Name   string
	Time   time.Time
	Status string
}

var db *sql.DB
var cfg configreader.Config
var migration dbworker.Migration
var err error

func main() {
	cfg, err = configreader.ConfigReader("./config.json")
	if err != nil {
		log.Fatalf("error with reading config.json, %s\n", err)
	}

	db, err = sql.Open(cfg.Database[0].Driver, cfg.Database[0].Dsn)
	if err != nil {
		log.Printf("error opening connection with db: %s\n", err)
	}

	defer db.Close()

	if err = dbworker.Ping(db); err != nil {
		log.Fatalf("error with connecting and pinging db, %s\n", err)
	}

	log.Printf("start using %s driver\n", cfg.Database[0].Driver)
	log.Printf("db connect and ping: ok\n")

	router := gin.Default()
	router.LoadHTMLFiles("templates/index.tmpl")
	router.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	router.GET("/", getMigrationsHandler)
	router.GET("/migrations", getMigrationsHandler)
	router.POST("/migrations/apply/:id", applyMigrationsHandler)
	router.POST("/migrations/rollback/:id", rollbackMigrationsHandler)

	router.Run(fmt.Sprintf(":%s", cfg.Server.Port))
}

func getMigrationsHandler(c *gin.Context) {
	items, err := getMigrations()
	if err != nil {
		log.Fatal(err)
	}
	c.HTML(200, "index.tmpl", items)
}

func applyMigrationsHandler(c *gin.Context) {
	// c.String(200, "apply started\n")
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	msgCh := make(chan string)
	go applyMigrations(id, msgCh)
	fullname := <-msgCh
	go applyMigrationsState(id, fullname)
	c.Redirect(http.StatusFound, "/migrations")
}

func rollbackMigrationsHandler(c *gin.Context) {
	// c.String(200, "rollback started\n")
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	go rollbackMigrations(id)
	go rollbackMigrationsState(id)
	c.Redirect(http.StatusFound, "/migrations")
}

func getMigrations() ([]Available, error) {
	items := []Available{}

	files, err := os.ReadDir("./migrations")
	if err != nil {
		log.Fatal(err)
	}

	ids := []int{}
	for _, file := range files {

		id := strings.Split(file.Name(), "_")[0]
		i, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}
		var status string
		if !slices.Contains(ids, i) {
			ids = append(ids, i)
			fullname := strings.SplitN(file.Name(), "_", 2)[1]
			fullname = strings.Split(fullname, ".")[0]
			if err = migration.GetMigrationStatus(db, i); err != nil {
				status = "not yet"
			} else {
				status = "in realese"
			}
			items = append(items, Available{ID: i, Name: fullname, Time: time.Now().Local(), Status: status})
		}
	}
	return items, nil
}

func applyMigrations(id int, nameCh chan string) {

	var fullname string
	files, err := os.ReadDir("./migrations")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), fmt.Sprintf("%d_", id)) {
			fullname = strings.Split(file.Name(), ".")[0]
			break
		}
	}

	file, err := os.ReadFile(fmt.Sprintf("./migrations/%s.up.sql", fullname))
	if err != nil {
		log.Fatal(err)
	}

	err = dbworker.ExecByte(db, file)
	if err != nil {
		log.Printf("error with getting migration info, %s\n", err)
	}
	nameCh <- fullname
	log.Printf("applykMigrations: ok")
}

func rollbackMigrations(id int) {

	var fullname string
	files, err := os.ReadDir("./migrations")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), fmt.Sprintf("%d_", id)) {
			fullname = strings.Split(file.Name(), ".")[0]
			break
		}
	}

	file, err := os.ReadFile(fmt.Sprintf("./migrations/%s.down.sql", fullname))
	if err != nil {
		log.Fatal(err)
	}

	err = dbworker.ExecByte(db, file)
	if err != nil {
		log.Printf("error with getting migration info, %s\n", err)
	}
	log.Printf("rollbackMigrations: ok")
}

func applyMigrationsState(id int, fullname string) {

	name := strings.SplitN(fullname, "_", 2)[1]
	query := fmt.Sprintf("INSERT INTO schema_migrations VALUES (%v, '%s', '%s');", id, name, time.Now().Local())
	if err = dbworker.ExecString(db, query); err != nil {
		log.Printf("error with writing migration changes info, %s\n", err)
	}
	log.Printf("applykMigrationsState: ok")
}

func rollbackMigrationsState(id int) {

	query := fmt.Sprintf("DELETE FROM schema_migrations WHERE id=%v;", id)
	if err = dbworker.ExecString(db, query); err != nil {
		log.Printf("error with writing migration changes info, %s\n", err)
	}
	log.Printf("rollbackMigrationsState: ok")
}
