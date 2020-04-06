package models

import (
	"fmt"
	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"gopkg.in/gormigrate.v1"
	"regexp"
	"strings"
	"time"
)

type Book struct {
	ID            uint   `gorm:"primary_key" json:"-"`
	BookId        string `gorm:"unique"`
	Title         string `json:"title"`
	Year          int  `json:"year"`
	Description   string `gorm:"type:text"`
	CoverUrl      string
	ReaderID uint
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
	DeletedAt     *time.Time `sql:"index" json:"-"`
	Authors       []Author   `gorm:"many2many:book_authors;"`
}

func (b *Book) ToUrlSafe() {
	bookid := b.Title
	bookid = strings.ToLower(bookid)
	bookid = strings.ReplaceAll(bookid, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	bookid = reg.ReplaceAllString(bookid, "")
	b.BookId = bookid
	fmt.Println(b.BookId)
}

func (b *Book) SetStringId() {
	db := bdb.ConnectToBook()
	fmt.Println(b.Title)
	b.ToUrlSafe()
	fmt.Println(b.BookId)
	emptyBook := Book{}
	val := 1
	orginalId := b.BookId
	for !db.Where(Book{BookId: b.BookId}).Find(&emptyBook).RecordNotFound() {
		b.BookId = fmt.Sprintf("%s%d", orginalId, val)
		val += 1
		emptyBook = Book{}
	}
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

func (a *Author) ToUrlSafe() {
	authorid := a.Name
	authorid = strings.ToLower(authorid)
	authorid = strings.ReplaceAll(authorid, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	authorid = reg.ReplaceAllString(authorid, "")
	a.AuthorId = authorid
}

func (a *Author) SetStringId() {
	db := bdb.ConnectToAuthor()
	a.ToUrlSafe()
	emptyAuthor := Author{}
	val := 1
	orginalId := a.AuthorId
	for !db.Where(Author{AuthorId: a.AuthorId}).Find(&emptyAuthor).RecordNotFound() {
		a.AuthorId = fmt.Sprintf("%s%d", orginalId, val)
		val += 1
		emptyAuthor = Author{}
	}
}

func Start(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "initial",
			Migrate: func(tx *gorm.DB) error {
				type Book struct {
					ID            uint   `gorm:"primary_key" json:"-"`
					BookId        string `gorm:"unique"`
					Title         string
					Year          int32  `json:"year"`
					Description   string `gorm:"type:text"`
					CoverUrl      string
					ReaderID uint
					CreatedAt     time.Time  `json:"-"`
					UpdatedAt     time.Time  `json:"-"`
					DeletedAt     *time.Time `sql:"index" json:"-"`
					Authors       []Author   `gorm:"many2many:book_authors;"`
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
