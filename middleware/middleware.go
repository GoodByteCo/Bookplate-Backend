package middleware

import (
	"context"
	db2 "github.com/GoodByteCo/Bookplate-Backend/db"
	"net/http"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/go-chi/chi"
)

func BookCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookId := chi.URLParam(r, "bookID")
		book := models.Book{}
		db := db2.Connect()
		db.Where(models.Book{BookId: bookId}).First(&book)
		ctx := context.WithValue(r.Context(), "book", book)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthorCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorId := chi.URLParam(r, "authorID")
		author := models.Author{}
		var books []models.Book
		db := db2.Connect()
		db.Where(models.Author{AuthorId: authorId}).First(&author)
		db.Model(&author).Related(&books, "Books")
		ctx := context.WithValue(r.Context(), "author", author)
		ctx = context.WithValue(ctx, "books", books)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}