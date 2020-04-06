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
		db := db2.ConnectToBook()
		db.Where(models.Book{BookId: bookId}).First(&book)
		ctx := context.WithValue(r.Context(), "book", book)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

