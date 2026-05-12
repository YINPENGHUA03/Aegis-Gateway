package bootstrap

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"aegis-gateway/internal/global"

	_ "github.com/go-sql-driver/mysql"
)

func InitMySQL() {
	dsn := fmt.Sprintf("root:%s@tcp(%s)/appoint_db?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("MYSQL_ROOT_PASSWORD"),
		os.Getenv("MYSQL_ADDR"))

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
