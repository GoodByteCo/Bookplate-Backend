package models

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/pquerna/ffjson/ffjson"
)

//Reader gotten from Request to add reader
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

//Reader gotten from login request
type LoginReader struct {
	Email    string `json:"email" scheme:"email"`
	Password string `json:"password" scheme:"email"`
}

//Book Info gotten from request to add book
type ReqWebBook struct {
	Title       string   `json:"title"`
	Year        string   `json:"year"`
	Authors     []Author `json:"authors"`
	Description string   `json:"description"`
	CoverUrl    string   `json:"cover_url"`
}

type ReqBookListAdd struct {
	List   string `json:"list"`
	BookID string `json:"book_id"`
}

type InternalInList struct {
	Read    bool `json:"read"`
	Liked   bool `json:"liked"`
	ToRead  bool `json:"to_read"`
	Library bool `json:"library"`
}

type ReqInList struct {
	Read    bool    `json:"read"`
	Liked   bool    `json:"liked"`
	ToRead  bool    `json:"to_read"`
	Library bool    `json:"library"`
	Friends Friends `json:"friends"`
}

type Friend struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ProfileColor string `json:"profile_color"`
}

type Friends []Friend

//Book info sent to site
type ResWebBook struct {
	Title       string            `json:"title"`
	Year        string            `json:"year"`
	Authors     ResAuthorsForBook `json:"authors"`
	Description string            `json:"description"`
	CoverUrl    string            `json:"cover_url"`
	BookColor   string            `json:"book_color"`
}

//Book data for author request
type ResBookForAuthor struct {
	BookId    string `json:"book_id"`
	Year      int    `json:"-"`
	Title     string `json:"title"`
	CoverUrl  string `json:"cover_url"`
	BookColor string `json:"book_color"`
}

//Author data to respond to request for author
type ResWebAuthor struct {
	Name  string            `json:"name"`
	Books ResBooksForAuthor `json:"books"`
}

//Author data for book request
type ResAuthorForBook struct {
	AuthorId string `json:"author_id"`
	Name     string `json:"name"`
}

//List aliases

type ResAuthorsForBook []ResAuthorForBook

type Books []Book

type ReqWebBooks []ReqWebBook

type Authors []Author

type ResBooksForAuthor []ResBookForAuthor

//Info when asking for all books
type AllWebBook struct {
	BookId    string `json:"book_id"`
	Title     string `json:"title"`
	CoverUrl  string `json:"cover_url"`
	BookColor string `json:"book_color"`
}

func (a *ResBooksForAuthor) Sort() {
	sort.SliceStable(a, func(i, j int) bool { return (*a)[i].Year < (*a)[j].Year })
}

func (w ReqWebBook) ToJson() []byte {
	j, err := ffjson.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}

func (a Author) ToBookAuthor() ResAuthorForBook {
	return ResAuthorForBook{
		AuthorId: a.AuthorId,
		Name:     a.Name,
	}

}

func (as Authors) ToBookAuthors() ResAuthorsForBook {
	var authors ResAuthorsForBook
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
		CoverUrl:    b.CoverURL,
	}
}

func (b Book) ToResWebBook(author Authors) ResWebBook {
	return ResWebBook{
		Title:       b.Title,
		Year:        strconv.Itoa(b.Year),
		Authors:     author.ToBookAuthors(),
		Description: b.Description,
		CoverUrl:    b.CoverURL,
		BookColor:   b.BookColor,
	}
}

func (b Book) ToBookForAuthor() ResBookForAuthor {
	return ResBookForAuthor{
		BookId:    b.BookID,
		Title:     b.Title,
		CoverUrl:  b.CoverURL,
		BookColor: b.BookColor,
	}
}

func (b Book) ToAllWebBook() AllWebBook {
	return AllWebBook{
		BookId:    b.BookID,
		Title:     b.Title,
		CoverUrl:  b.CoverURL,
		BookColor: b.BookColor,
	}
}

func (bs Books) ToAuthorBooks() ResBooksForAuthor {
	var books ResBooksForAuthor
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
	j, err := ffjson.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}

func (w ResWebBook) ToJson() []byte {
	j, err := ffjson.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return j
}
