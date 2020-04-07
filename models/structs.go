package models

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ReaderAdd struct {
	Name     string  `json:"name"`
	Pronouns Pronoun `json:"pronouns"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
}

type Pronoun struct {
	Subject    string `json:"subject"`
	Object     string `json:"object"`
	Possessive string `json:"possessive"`
}

type LoginReader struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WebBook struct {
	Title       string   `json:"title"`
	Year        string   `json:"year"`
	Authors     []Author `json:"authors"`
	Description string   `json:"description"`
	CoverUrl    string   `json:"cover_url"`
}

type AuthorBook struct {
	BookId string `json:"book_id"`
	Year int `json:"-"`
	Title string `json:"title"`
	CoverUrl string `json:"cover_url"`
}

type AuthorBooks []AuthorBook

func (a *AuthorBooks) Sort() {
	sort.SliceStable(a, func(i, j int) bool {return (*a)[i].Year < (*a)[j].Year})
}

type Books []Book

type WebBooks []WebBook

type AllWebBook struct {
	BookId   string `json:"book_id"`
	Title    string `json:"title"`
	CoverUrl string `json:"cover_url"`
}

type WebAuthor struct {
	Name string `json:"name"`
	Books AuthorBooks `json:"books"`
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
		Year:        year,
		Description: html.EscapeString(v.Get("description")),
		CoverUrl:    url,
		ReaderID:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   nil,
		Authors: []Author{
			{
				AuthorId: "",
				Name:     "",
			},
		},
	}
}

func (b Book) ToWebBook() WebBook {
	return WebBook{
		Title:       b.Title,
		Year:        strconv.Itoa(b.Year),
		Description: b.Description,
		CoverUrl:    b.CoverUrl,
	}
}

func (b Book) ToAuthorBook() AuthorBook {
	return AuthorBook{
		BookId:   b.BookId,
		Title:    b.Title,
		CoverUrl: b.CoverUrl,
	}
}

func (b Book) ToAllWebBook() AllWebBook {
	return AllWebBook{
		BookId:   b.BookId,
		Title:    b.Title,
		CoverUrl: b.CoverUrl,
	}
}

func (bs Books) ToWebBooks() WebBooks{
	var books WebBooks
	for _, b := range bs {
		books = append(books, b.ToWebBook())
	}
	return books

}

func (bs Books) ToAuthorBooks() AuthorBooks{
	var books AuthorBooks
	if &bs != nil {
		for _, b := range bs {
			books = append(books, b.ToAuthorBook())
		}
		if len(books) > 1 {
			books.Sort()
		}
	}
	return books
}

func (a Author) ToWebAuthor(b Books) WebAuthor {
	return WebAuthor{
		Name:  a.Name,
		Books: b.ToAuthorBooks(),
	}
}

func (w WebAuthor) ToJson() []byte {
	j, err := json.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}

