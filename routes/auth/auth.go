package auth

import (
	"crypto/sha1"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"log"
	"net/http"
	"os"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)


var h = sha1.New()

func init() {
	//gothic.Store :=
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_CLIENT"), os.Getenv("GOOGLE_SECRET"), "http://localhost:8081/api/auth/callback?provider=google"),
		github.New(os.Getenv("GITHUB_CLIENT"), os.Getenv("GITHUB_SECRET"), "http://localhost:8081/api/auth/callback?provider=github"),
	)
}

func AuthCallback(res http.ResponseWriter, req *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(res, req)
	if err != nil {
		//do something
		log.Println(res, err)
		panic("yikes")
	}

	log.Printf ("%+v",gothUser)
	emailHash, err := h.Write([]byte(gothUser.Email))
	user, found := utils.GetReaderFromDB(emailHash)
	if found {
		gothic.StoreInSession("user_id", string(user.ID), req, res)
		//http.Redirect(things)
	} else {
		//no users
		res.Write([]byte("lol"))
	}
}

func Auth(res http.ResponseWriter, req *http.Request) {
	if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
		log.Println(gothUser)
	} else {
		gothic.BeginAuthHandler(res, req)
	}

}
