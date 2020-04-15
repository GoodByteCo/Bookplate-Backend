package middleware

import (
	"context"
	"fmt"
	db2 "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"net/http"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/go-chi/chi"
)

func BookCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookId := chi.URLParam(r, "bookID")
		book := models.Book{}
		var authors []models.Author
		db := db2.Connect()
		db.Where(models.Book{BookId: bookId}).First(&book)
		db.Model(&book).Related(&authors, "Authors")
		ctx := context.WithValue(r.Context(), "book", book)
		ctx = context.WithValue(ctx, "authors", authors)
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

func LoginWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, claims, err := jwtauth.FromContext(r.Context())

		if err != nil {
			next.ServeHTTP(w,r)
			return
		}

		issb := token.Claims.(jwt.MapClaims).VerifyIssuer(utils.Issuer, false)
		if !issb {
			next.ServeHTTP(w,r)
			return
		}

		if token == nil || !token.Valid {
			next.ServeHTTP(w,r)
			return
		}
		fmt.Println(claims["reader_id"])
		ctx := context.WithValue(r.Context(), "reader_id", claims["reader_id"])
		//get claims
		// Token is authenticated, pass it through
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


