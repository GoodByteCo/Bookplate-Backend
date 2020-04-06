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
	Title       string `json:"title"`
	Year        string  `json:"year"`
	Authors     []Author `json:"authors"`
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

func UrlValueToBook(v url.Values, url string) Book{
	bookid := v.Get("bookname")
	bookid = strings.ToLower(bookid)
	bookid = strings.ReplaceAll(bookid, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	bookid = reg.ReplaceAllString(bookid, "")
	year, _ := strconv.Atoi(v.Get("year"))
	return Book{
		BookId:      bookid,
		Title:       v.Get("bookname"),
		Year:        int(year),
		Description: html.EscapeString(v.Get("description")),
		CoverUrl:    url,
		ReaderID:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   nil,
		Authors: []Author{
			{
				AuthorId:  "",
				Name:      "",
			},

		},
	}
}

func (b Book) ToWebBook() WebBook {
	return WebBook{
		Title:       b.Title,
		Year:        string(b.Year),
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


