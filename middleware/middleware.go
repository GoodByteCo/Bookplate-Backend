package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	db2 "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/go-chi/chi"
)

func BookCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookId := chi.URLParam(r, "bookID")
		book := models.Book{}
		var authors []models.Author
		db := db2.Connect()
		defer db.Close()
		db.Where(models.Book{BookID: bookId}).First(&book)
		db.Model(&book).Related(&authors, "Authors")
		ctx := context.WithValue(r.Context(), utils.BookKey, book)
		ctx = context.WithValue(ctx, utils.AuthorKey, authors)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthorCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorId := chi.URLParam(r, "authorID")
		author := models.Author{}
		var books []models.Book
		db := db2.Connect()
		defer db.Close()
		db.Where(models.Author{AuthorId: authorId}).First(&author)
		db.Model(&author).Related(&books, "Books")
		ctx := context.WithValue(r.Context(), utils.AuthorKey, author)
		ctx = context.WithValue(ctx, utils.BookKey, books)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReaderWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readerID := chi.URLParam(r, "readerID")
		intReaderID, err := strconv.ParseUint(readerID, 10, 64)
		if err != nil {
			http.Error(w, "not a user", 404)
			return
		}
		if intReaderID == 0 {
			http.Error(w, "not a user", 404)
			return
		}
		var reader models.Reader
		db := db2.Connect()
		defer db.Close()
		notFound := db.Where(models.Reader{ID: uint(intReaderID)}).Find(&reader).RecordNotFound()
		if notFound {
			http.Error(w, "user not found", 404)
			return
		}
		ctx := context.WithValue(r.Context(), utils.ReaderUserKey, reader)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LoginWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, claims, err := jwtauth.FromContext(r.Context())

		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		err = claims.Valid()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		switch exp := claims["exp"].(type) {
		case float64:
			fmt.Println(exp)
		case json.Number:
			fmt.Print("json: expiry")
			fmt.Println(exp)
		}

		issb := token.Claims.(jwt.MapClaims).VerifyIssuer(utils.Issuer, false)
		if !issb {
			next.ServeHTTP(w, r)
			return
		}

		if token == nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		fmt.Println(claims["reader_id"])
		tID := claims["reader_id"]
		tempReaderID := tID.(float64)
		readerID := uint(tempReaderID)
		ctx := context.WithValue(r.Context(), utils.ReaderKey, readerID)
		//get claims
		// Token is authenticated, pass it through
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, ok := ctx.Value(utils.ReaderKey).(uint)
		if !ok {
			http.Error(w, "not logged in", 401)
			return
		}
		if id == 0 {
			http.Error(w, "not logged in", 401)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CheckBook(next http.Handler) http.Handler {
	{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bookId := chi.URLParam(r, "bookID")
			book := models.Book{}
			db := db2.Connect()
			defer db.Close()
			not := db.Where(models.Book{BookID: bookId}).First(&book).RecordNotFound()
			fmt.Println(not)
			if not == true {
				http.Error(w, "book doesn't exist", 404)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CachingWare(duration time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
