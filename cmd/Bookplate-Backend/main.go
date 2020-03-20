package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/GoodByteCo/Bookplate-Backend/Middleware"
	"github.com/GoodByteCo/Bookplate-Backend/routes"
	"github.com/GoodByteCo/Bookplate-Backend/routes/auth"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-chi/jwtauth"
)

func init() {
	utils.Migrate()
}

func main() {
	r := chi.NewRouter()
	c := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(c.Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(r chi.Router) {
		r.Get("/ping", routes.Ping)

		r.Get("/auth", auth.Auth)
		r.Get("/auth/callback", auth.AuthCallback)
		r.Route("/book", func(r chi.Router) {
			r.Post("/add", routes.AddBook)
			r.Route("/{bookID}", func(r chi.Router) {
				r.Use(Middleware.ArticleCtx)
				r.Get("/", routes.GetBook)
			})
		})
	})

	fmt.Println("serving on port 8081")
	_ = http.ListenAndServe(":8081", r)
}
