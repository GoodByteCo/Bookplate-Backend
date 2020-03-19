package Middleware

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/holopollock/Bookplate/Models"
	"github.com/holopollock/Bookplate/utils"
	"net/http"
)

func ArticleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		bookId := chi.URLParam(r, "bookID")
		book := Models.Book{}
		db := utils.ConnectToBook()
		db.Where(Models.Book{BookId:bookId}).First(&book)
		ctx := context.WithValue(r.Context(), "book", book)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
