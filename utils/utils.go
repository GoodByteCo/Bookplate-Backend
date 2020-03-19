package utils

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

func connect() *gorm.DB {
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=quinnpollock dbname=BookPlateGo password=bookplate sslmode=disable")
	if err != nil {
		fmt.Println(err)
	}
	return db
}

func Migrate() {
	db := connect()
	db.AutoMigrate()
	db.Close()
}

func ConnectToBook() *gorm.DB {
	db := connect()
	return db.Table("books")
}
