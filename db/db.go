package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
)

func Connect() *gorm.DB {
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=quinnpollock dbname=BookPlateGo password=bookplate sslmode=disable")
	if err != nil {
		panic("DB Down")
		fmt.Println(err)
	}
	return db
}

func ConnectToBook() *gorm.DB {
	db := Connect()
	return db.Table("books")
}

func ConnectToReader() *gorm.DB {
	db := Connect()
	return db.Table("readers")
}

func ConnectToAuthor() *gorm.DB {
	db := Connect()
	return db.Table("authors")

}

