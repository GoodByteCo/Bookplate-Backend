package utils

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	"github.com/jinzhu/gorm"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func connect() *gorm.DB {
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=quinnpollock dbname=BookPlateGo password=bookplate sslmode=disable")
	if err != nil {
		panic("DB Down")
		fmt.Println(err)
	}
	return db
}

func Migrate() {
	db := connect()
	fmt.Println()
	db.AutoMigrate(&models.Reader{}, &models.Book{}, &models.Author{})
	db.Close()
}

func ConnectToBook() *gorm.DB {
	db := connect()
	return db.Table("books")
}

func ConnectToReader() *gorm.DB {
	db := connect()
	return db.Table("readers")
}

func GetReaderFromDB(emailhash int) (models.Reader, bool) {
	db := ConnectToReader()
	emptyReader := models.Reader{}
	found := db.Where(models.Reader{EmailHash: emailhash}).Find(&emptyReader).RecordNotFound()
	defer db.Close()
	return emptyReader, found
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
