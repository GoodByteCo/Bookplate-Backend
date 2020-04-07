package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

type ReqReader struct {
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

type ReqWebBook struct {
	Title       string   `json:"title"`
	Year        string   `json:"year"`
	Authors     []Author `json:"authors"`
	Description string   `json:"description"`
	CoverUrl    string   `json:"cover_url"`
}

type ResWebBook struct {
	Title       string         `json:"title"`
	Year        string         `json:"year"`
	Authors     AuthorsForBook `json:"authors"`
	Description string         `json:"description"`
	CoverUrl    string         `json:"cover_url"`

}

type BookForAuthor struct {
	BookId string `json:"book_id"`
	Year int `json:"-"`
	Title string `json:"title"`
	CoverUrl string `json:"cover_url"`
}

type ResWebAuthor struct {
	Name  string         `json:"name"`
	Books BooksForAuthor `json:"books"`
}

type AuthorForBook struct {
	AuthorId string `json:"author_id"`
	Name string `json:"name"`
}

type AuthorsForBook []AuthorForBook

type Books []Book

type ReqWebBooks []ReqWebBook

type Authors []Author

type BooksForAuthor []BookForAuthor

type AllWebBook struct {
	BookId   string `json:"book_id"`
	Title    string `json:"title"`
	CoverUrl string `json:"cover_url"`
}

func (a *BooksForAuthor) Sort() {
	sort.SliceStable(a, func(i, j int) bool {return (*a)[i].Year < (*a)[j].Year})
}

func (w ReqWebBook) ToJson() []byte {
	j, err := json.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}

func (a Author) ToBookAuthor() AuthorForBook {
	return AuthorForBook{
		AuthorId: a.AuthorId,
		Name:     a.Name,
	}

}

func (as Authors) ToBookAuthors() AuthorsForBook {
	var authors AuthorsForBook
	for _, a := range as {
		authors = append(authors, a.ToBookAuthor())
	}
	return authors

}

func (b Book) ToWebBook() ReqWebBook {
	return ReqWebBook{
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

func (b Book) ToBookForAuthor() BookForAuthor {
	return BookForAuthor{
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

func (bs Books) ToAuthorBooks() BooksForAuthor {
	var books BooksForAuthor
	if &bs != nil {
		for _, b := range bs {
			books = append(books, b.ToBookForAuthor())
		}
		if len(books) > 1 {
			books.Sort()
		}
	}
	return books
}

func (a Author) ToWebAuthor(b Books) ResWebAuthor {
	return ResWebAuthor{
		Name:  a.Name,
		Books: b.ToAuthorBooks(),
	}
}

func (w ResWebAuthor) ToJson() []byte {
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
