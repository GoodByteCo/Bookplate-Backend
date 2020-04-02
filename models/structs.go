package models

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
)

type ReaderAdd struct {
	Name     string `json:"name"`
	Pronouns Pronoun `json:"pronouns"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Pronoun struct {
	Subject    string `json:"subject"`
	Object     string `json:"object"`
	Possessive string `json:"possessive"`
}

type LoginReader struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type Reader struct {
	gorm.Model
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
	Books         []Book
}

type Book struct {
	BookId      string `gorm:"PRIMARY_KEY;unique"`
	Title       string
	Year        int32
	Author      string
	Description string `gorm:"type:text"`
	CoverUrl    string
	ReaderID    uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `sql:"index"`
}

type WebBook struct {
	Title       string `json:"title"`
	Year        int32  `json:"year"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CoverUrl    string `json:"cover_url"`
}

type AllWebBook struct {
	BookId   string `json:"book_id"`
	Title    string `json:"title"`
	CoverUrl string `json:"cover_url"`
}

func (w WebBook) ToJson() []byte {
	j, err := json.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}

func UrlValueToBook(v url.Values, url string) Book {
	bookid := v.Get("bookname")
	bookid = strings.ToLower(bookid)
	bookid = strings.ReplaceAll(bookid, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	bookid = reg.ReplaceAllString(bookid, "")
	year, _ := strconv.Atoi(v.Get("year"))
	return Book{
		BookId:      bookid,
		Title:       v.Get("bookname"),
		Year:        int32(year),
		Author:      v.Get("author"),
		Description: html.EscapeString(v.Get("description")),
		CoverUrl:    url,
		ReaderID:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   nil,
	}
}

func (b Book) ToWebBook() WebBook {
	return WebBook{
		Title:       b.Title,
		Year:        b.Year,
		Author:      b.Author,
		Description: b.Description,
		CoverUrl:    b.CoverUrl,
	}
}

func (b Book) ToAllWebBook() AllWebBook {
	return AllWebBook{
		BookId:   b.BookId,
		Title:    b.Title,
		CoverUrl: b.CoverUrl,
	}
}

type Author struct {
	AuthorId  string `gorm:"PRIMARY_KEY;unique"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}
