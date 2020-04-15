package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"gopkg.in/kothar/go-backblaze.v0"
	"github.com/gorilla/schema"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	body := "Pong"
	w.Header().Set("Content-Type", http.DetectContentType([]byte(body)))
	w.Header().Add("Accept-Charset", "utf-8")
	w.Write([]byte(body))
}

func UploadBook(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	names := strings.Split(header.Filename, ".")
	var photo io.Reader
	if names[1] == "png" {
		photo = utils.CompressPng(file)
	} else if names[1] == "jpeg" || names[1] == "jpg" {
		photo = utils.CompressJpg(file)
	}
	fmt.Printf("File name %s.%s\n", names[0], names[1])
	name := utils.String(32)
	name = fmt.Sprintf("%s.%s", name, names[1])
	//maybe make random later
	fmt.Println(name)
	bookplateBucket := getBucket()
	metadata := make(map[string]string)
	b2file, err := bookplateBucket.UploadFile(name, metadata, photo)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(b2file.UploadTimestamp)
	url, err := bookplateBucket.FileURL(name)
	if err != nil {
		fmt.Println(err)
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte(url)))
	w.Header().Add("Accept-Charset", "utf-8")
	w.Write([]byte(url))

}
func AddBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	decoder := json.NewDecoder(r.Body)
	var book models.ReqWebBook
	_ = decoder.Decode(&book)
	id, err := utils.AddBook(book)
	if err != nil {
		fmt.Println(err)
		return
	}
	js, _ := json.Marshal(id)
	w.Header().Set("Content-Type", http.DetectContentType(js))
	w.Write(js)
}

//func AddAuthor(w http.ResponseWriter, r *http.Request){
//	decoder := json.NewDecoder(r.body)
//
//}

func AddReader(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	decoder := json.NewDecoder(r.Body)
	var reader models.ReqReader
	_ = decoder.Decode(&reader)
	id, err, userExist := utils.AddReader(reader)
	if err != nil {
		//do something
		http.Error(w, http.StatusText(500)+": Server Error", 500)
	}
	if userExist != nil {
		w.Write([]byte("user exists"))
		return

		//do something
	} else {
		expiry := time.Now().Add(time.Hour * 12)
		mc := jwt.MapClaims{"reader_id": id, "iss": utils.Issuer}
		jwtauth.SetIssuedNow(mc)
		jwtauth.SetExpiry(mc, expiry)
		_, tokenString, tokenErr := utils.TokenAuth.Encode(mc)
		if tokenErr != nil {
			fmt.Println("token Generated Error")
			http.Error(w, http.StatusText(500)+": "+tokenErr.Error(), 500)
			return
		}
		fmt.Println(tokenString)
		http.SetCookie(w, &http.Cookie{
			Name:"user_id",
			Value: strconv.Itoa(int(id)),
			Expires: expiry,
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: expiry,
			HttpOnly: true,
			Domain: "bookplate.co", //add when correct
		})
		w.Write([]byte("user added"))
	}

}

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	decoder := schema.NewDecoder()
	var loginReader models.LoginReader
	err = decoder.Decode(&loginReader, r.PostForm)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(loginReader.Email)
	reader, err := utils.CheckIfPresent(loginReader.Email)
	if err != nil {
		fmt.Println("uh oh")
		fmt.Println(err.Error())
		return
		//no user redirect to create account
	}
	if utils.ConfirmPassword(reader.PasswordHash, loginReader.Password) {
		expiry := time.Now().Add(time.Hour * 12)
		mc := jwt.MapClaims{"reader_id": reader.ID, "iss": utils.Issuer}
		jwtauth.SetIssuedNow(mc)
		jwtauth.SetExpiry(mc, expiry)
		_, tokenString, tokenErr := utils.TokenAuth.Encode(mc)
		if tokenErr != nil {
			fmt.Println("token Generated Error")
			http.Error(w, http.StatusText(500)+": "+tokenErr.Error(), 500)
			return
		}
		fmt.Println(tokenString)
		http.SetCookie(w, &http.Cookie{
			Name:"user_id",
			Value: strconv.Itoa(int(reader.ID)),
			Expires: expiry,
		})
		http.SetCookie(w, &http.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: expiry,
			HttpOnly: true,
			Domain: "bookplate.co", //add when correct
		})
		w.Write([]byte("we did it"))
	} else {
		w.Write([]byte("wrong password"))
	}

}

func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	http.SetCookie(w, &http.Cookie{
		Name:   "jwt",
		Value:  "",
		MaxAge: -1,
	})

	w.Write([]byte("Logged Out"))
}

func GetBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	ctx := r.Context()
	book, ok := ctx.Value("book").(models.Book)
	if !ok {
		//errpr
		return
	}
	authors, ok := ctx.Value("authors").([]models.Author)
	webbook := book.ToResWebBook(authors)
	js := webbook.ToJson()
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func GetAuthor(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	author, ok := ctx.Value("author").(models.Author)
	if !ok {
		//errpr
		return
	}
	books, ok := ctx.Value("books").([]models.Book)
	if !ok {
		//errpr
		return
	}
	webAuthor := author.ToWebAuthor(books)
	js := webAuthor.ToJson()
	w.Write(js)
}

func GetAllBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	var records []models.Book
	var webBooks []models.AllWebBook
	db := bdb.ConnectToBook()
	if err := db.Find(&records).Error; err != nil {
		fmt.Println(err)
	}
	for _, book := range records {
		web := book.ToAllWebBook()
		webBooks = append(webBooks, web)
	}
	js, err := json.Marshal(webBooks)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}
func getBucket() *backblaze.Bucket {
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      os.Getenv("B2_ACCOUNT_ID"),
		ApplicationKey: os.Getenv("B2_APP_KEY"),
	})
	if err != nil {
		fmt.Println(err)
		panic("yikes")
	}
	bookplateBucket, err := b2.Bucket("Bookplate")
	if err != nil {
		fmt.Println(err)
		panic("yikes")
	}
	return bookplateBucket
}
