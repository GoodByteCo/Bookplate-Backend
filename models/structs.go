package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

type ResWebBook struct {
	Title       string   `json:"title"`
	Year        string   `json:"year"`
	Authors     BookAuthors `json:"authors"`
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

type BookAuthor struct {
	AuthorId string `json:"author_id"`
	Name string `json:"name"`
}

type BookAuthors []BookAuthor

type Books []Book

type WebBooks []WebBook

type Authors []Author

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

func (a Author) ToBookAuthor() BookAuthor {
	return BookAuthor{
		AuthorId: a.AuthorId,
		Name:     a.Name,
	}

}

func (as Authors) ToBookAuthors() BookAuthors {
	var authors BookAuthors
	for _, a := range as {
		authors = append(authors, a.ToBookAuthor())
	}
	return authors

}

func (b Book) ToWebBook() WebBook {
	return WebBook{
		Title:       b.Title,
		Year:        strconv.Itoa(b.Year),
		Description: b.Description,
		CoverUrl:    b.CoverUrl,
	}
}

func (b Book) ToResWebBook(author Authors) ResWebBook {
	return ResWebBook{
		Title:       b.Title,
		Year:        strconv.Itoa(b.Year),
		Authors:     author.ToBookAuthors(),
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

func (w ResWebBook) ToJson() []byte {
	j, err := json.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}
