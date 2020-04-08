package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"gopkg.in/gormigrate.v1"
)

//Book type for database
type Book struct {
	ID          uint   `gorm:"primary_key" json:"-"`
	BookId      string `gorm:"unique"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Description string `gorm:"type:text"`
	CoverUrl    string
	ReaderID    uint
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `sql:"index" json:"-"`
	Authors     []Author   `gorm:"many2many:book_authors;"`
}

//Remove non url safe characters from book title and set it as Id
func (b *Book) ToUrlSafe() {
	bookId := b.Title
	bookId = strings.ToLower(bookId)
	bookId = strings.ReplaceAll(bookId, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	bookId = reg.ReplaceAllString(bookId, "")
	b.BookId = bookId
}

//Find if Book Id exist and append number if so
func (b *Book) SetStringId() {
	db := bdb.ConnectToBook()
	b.ToUrlSafe()
	emptyBook := Book{}
	val := 1
	orginalId := b.BookId
	for !db.Where(Book{BookId: b.BookId}).Find(&emptyBook).RecordNotFound() {
		b.BookId = fmt.Sprintf("%s%d", orginalId, val)
		val += 1
		emptyBook = Book{}
	}
}

//Reader type for database
type Reader struct {
	ID            uint       `gorm:"primary_key" json:"-"`
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
	DeletedAt     *time.Time `sql:"index" json:"-"`
	Name          string
	Pronouns      postgres.Jsonb
	ProfileColour string
	Library       pq.StringArray `gorm:"type:varchar(64)[]"`
	ToRead        pq.StringArray `gorm:"type:varchar(64)[]"`
	Liked         pq.StringArray `gorm:"type:varchar(64)[]"`
	Friends       pq.Int64Array  `gorm:"type:integer[]"`
	PasswordHash  string
	EmailHash     int64
	Plural        bool
	Books         []Book `gorm:"foreignkey:ReaderAddedId"` //Book added by reader
}

//Author type for database
type Author struct {
	ID        uint       `gorm:"primary_key" json:"-"`
	AuthorId  string     `gorm:"unique" json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
	Books     []Book     `gorm:"many2many:book_authors;"`
}

//Remove non url safe characters from author name and set it as Id
func (a *Author) ToUrlSafe() {
	authorId := a.Name
	authorId = strings.ToLower(authorId)
	authorId = strings.ReplaceAll(authorId, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	authorId = reg.ReplaceAllString(authorId, "")
	a.AuthorId = authorId
}

//Find if BookId exist and append number if so
func (a *Author) SetStringId() {
	db := bdb.ConnectToAuthor()
	a.ToUrlSafe()
	emptyAuthor := Author{}
	val := 1
	originalId := a.AuthorId
	for !db.Where(Author{AuthorId: a.AuthorId}).Find(&emptyAuthor).RecordNotFound() {
		a.AuthorId = fmt.Sprintf("%s%d", originalId, val)
		val += 1
		emptyAuthor = Author{}
	}
}

//Migration Function Update as database structs change
func Start(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "initial",
			Migrate: func(tx *gorm.DB) error {
				type Book struct {
					ID          uint   `gorm:"primary_key" json:"-"`
					BookId      string `gorm:"unique"`
					Title       string
					Year        int32  `json:"year"`
					Description string `gorm:"type:text"`
					CoverUrl    string
					ReaderID    uint
					CreatedAt   time.Time  `json:"-"`
					UpdatedAt   time.Time  `json:"-"`
					DeletedAt   *time.Time `sql:"index" json:"-"`
					Authors     []Author   `gorm:"many2many:book_authors;"`
				}

				type Reader struct {
					ID            uint       `gorm:"primary_key" json:"-"`
					CreatedAt     time.Time  `json:"-"`
					UpdatedAt     time.Time  `json:"-"`
					DeletedAt     *time.Time `sql:"index" json:"-"`
					Name          string
					Pronouns      postgres.Jsonb
					ProfileColour string
					Library       pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead        pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked         pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends       pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash  string
					EmailHash     int64
					Plural        bool
					Books         []Book `gorm:"foreignkey:ReaderAddedId"` //Book added by reader
				}

				type Author struct {
					ID        uint       `gorm:"primary_key" json:"-"`
					AuthorId  string     `gorm:"unique" json:"id"`
					Name      string     `json:"name"`
					CreatedAt time.Time  `json:"-"`
					UpdatedAt time.Time  `json:"-"`
					DeletedAt *time.Time `sql:"index" json:"-"`
					Books     []Book     `gorm:"many2many:book_authors;"`
				}

				return tx.CreateTable(&Reader{}, Author{}, &Book{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable(&Reader{}, Author{}, &Book{}).Error
			},
		},
	})
	return m.Migrate()
}

func Migrate() {
	db := bdb.Connect()
	fmt.Println()
	Start(db)
	db.Close()
}
