package middleware

import (
	"context"
	"net/http"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"github.com/go-chi/chi"
)

func ArticleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookId := chi.URLParam(r, "bookID")
		book := models.Book{}
		db := utils.ConnectToBook()
		db.Where(models.Book{BookId: bookId}).First(&book)
		ctx := context.WithValue(r.Context(), "book", book)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

