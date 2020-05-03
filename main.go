package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"github.com/go-chi/jwtauth"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	"github.com/GoodByteCo/Bookplate-Backend/middleware"
	"github.com/GoodByteCo/Bookplate-Backend/routes"
	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-chi/jwtauth"
)

func init() {
	models.Migrate()
}

func main() {
	r := chi.NewRouter()
	c := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link", "Set-Cookie", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(c.Handler)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(jwtauth.Verifier(utils.TokenAuth))
	r.Use(middleware.LoginWare)
	r.Route("/", func(r chi.Router) {
		r.Get("/books", routes.GetAllBooks)
		r.Post("/logout", routes.Logout)
		r.Get("/ping", routes.Ping)
		r.Group(func(r chi.Router) {
			r.Use(chimiddleware.AllowContentType("application/json"))
			r.Post("/reader/add", routes.AddReader)
			r.Post("/login", routes.Login)
			r.Route("/list", func(r chi.Router) {
				r.Post("/add", routes.AddToList)
				r.Post("/remove", routes.DeleteFromList)
			})
		})

		r.Route("/profile", func(r chi.Router) {
			r.Route("/{readerID}", func(r chi.Router) {
				r.Use(middleware.ReaderWare)
				r.Get("/", routes.GetProfile)
				r.Get("/liked", routes.GetLiked)
				r.Get("/library", routes.GetLibrary)
				r.Get("/read", routes.GetRead)
				r.Get("/to-read", routes.GetToRead)
			})
		})

		r.Route("/friend", func(r chi.Router) {
			r.Use(middleware.AuthWare)
			r.Post("/add/{readerID}", routes.AddFriend)
			// r.Post("/remove/{readerID}")
			// r.Post("/blocked/{readerID}")
		})

		r.Route("/author", func(r chi.Router) {
			r.Route("/{authorID}", func(r chi.Router) {
				r.Use(middleware.AuthorCtx)
				r.Get("/", routes.GetAuthor)
			})
		})

		r.Route("/reader", func(r chi.Router) {
			r.Use(middleware.LoginWare)
			r.Use(middleware.AuthWare)
			r.Route("/profile", func(r chi.Router) {
				r.Get("/{readerID}", routes.GetReaderProfile)
			})
			r.Route("/book", func(r chi.Router) {
				r.Use(middleware.CheckBook)
				r.Get("/{bookID}", routes.GetReaderBook)
			})
		})

		r.Route("/book", func(r chi.Router) {
			r.Post("/add", routes.AddBook)
			r.Route("/{bookID}", func(r chi.Router) {
				r.Use(middleware.BookCtx)
				r.Get("/", routes.GetBook)
			})
		})
		r.Post("/upload", routes.UploadBook)
	})

	fmt.Println("serving on port 8081")
	_ = http.ListenAndServe(":8081", r)
}
