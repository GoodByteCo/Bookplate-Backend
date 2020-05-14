package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/go-chi/valve"
)

func init() {
	models.Migrate()
}

func main() {
	valv := valve.New()
	baseCtx := valv.Context()
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
	// r.Use(chimiddleware.Timeout(60 * time.Second))
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

		r.Route("/search", func(r chi.Router) {
			r.Get("/books/{term}", routes.SearchBooks)
			r.Get("/authors/{term}", routes.SearchAuthors)
		})

		r.Route("/forgot-password", func(r chi.Router) {
			r.Post("/", routes.ForgotPasswordRequest)
			r.Route("/{passwordKey}", func(r chi.Router) {
				r.Use(middleware.ConfirmPassKey)
				r.Post("/", routes.ForgotPasswordReset)
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
				r.Route("/friends", func(r chi.Router) {
					r.Get("/", routes.GetFriends)
				})
			})
		})

		r.Route("/friend", func(r chi.Router) {
			r.Use(middleware.AuthWare)
			r.Post("/add/{readerID}", routes.AddFriend)
			r.Post("/remove/{readerID}", routes.RemoveFriend)
			r.Post("/block/{readerID}", routes.BlockReader)
			r.Post("/unblock/{readerID}", routes.UnblockReader)
		})

		r.Route("/author", func(r chi.Router) {
			r.Route("/{authorID}", func(r chi.Router) {
				r.Use(middleware.AuthorCtx)
				r.Get("/", routes.GetAuthor)
			})
		})

		r.Route("/reader", func(r chi.Router) {
			r.Use(middleware.LoginWare)
			r.Route("/profile", func(r chi.Router) {
				r.Route("/{readerID}", func(r chi.Router) {
					r.Use(middleware.ReaderWare)
					r.Get("/", routes.GetReaderProfile)
					r.Route("/friends", func(r chi.Router) {
						r.Use(middleware.AuthWare)
						r.Get("/", routes.GetReaderFriends)
					})
				})
			})
			r.Route("/book", func(r chi.Router) {
				r.Route("/{bookID}", func(r chi.Router) {
					r.Use(middleware.CheckBook)
					r.Get("/", routes.GetReaderBook)
				})
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

	fmt.Println("serving on port 8080")
	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      chi.ServerBaseContext(baseCtx, r),
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			// sig is a ^C, handle it
			fmt.Println("shutting down..")

			// first valv
			valv.Shutdown(20 * time.Second)

			// create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			// start http shutdown
			srv.Shutdown(ctx)

			// verify, in worst case call cancel via defer
			select {
			case <-time.After(21 * time.Second):
				fmt.Println("not all connections done")
			case <-ctx.Done():

			}
		}
	}()
	srv.ListenAndServe()
}
