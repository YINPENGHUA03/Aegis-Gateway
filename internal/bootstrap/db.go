package bootstrap

import (
	"database/sql"
	"log"
	"time"

	"aegis-gateway/internal/global"

	_ "github.com/go-sql-driver/mysql"
)

func InitMySQL() {
	dsn := "root:0410@tcp(127.0.0.1:3309)/appointment?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf(" MySQL DSN Parsing failed: %v", err)
	}

	db.SetMaxOpenConns(100) //Set an upper limit on the concurrent traffic entering MySQL
	db.SetMaxIdleConns(20)  //Avoid frequently creating and closing connections
	db.SetConnMaxLifetime(time.Hour * 1)

	if err = db.Ping(); err != nil {
		log.Fatalf(" MySQL connection failed: %v", err)
	}
	global.DB = db
	log.Println("MYSQL connection pool initialized successfully!")
}
