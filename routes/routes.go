package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	berrors "github.com/GoodByteCo/Bookplate-Backend/errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/GoodByteCo/Bookplate-Backend/models"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"gopkg.in/kothar/go-backblaze.v0"
)

const replace = "https://photos.bookplate.co"
const start = "https://f001.backblazeb2.com"

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
	url = strings.ReplaceAll(url, start, replace)
	w.Write([]byte(url))
}

func AddBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	decoder := json.NewDecoder(r.Body)
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	var book models.ReqWebBook
	_ = decoder.Decode(&book)
	bID, err := utils.AddBook(book, id)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "internal server Error", http.StatusInternalServerError)
		return
	}
	js, _ := ffjson.Marshal(bID)
	w.Header().Set("Content-Type", http.DetectContentType(js))
	w.Write(js)
}

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
	}
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
		Name:     "jwt",
		Value:    tokenString,
		Expires:  expiry,
		HttpOnly: true,
		Domain:   "bookplate.co", //add when correct
	})
	w.Write([]byte("user added"))
}

func AddToList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	if id == 0 {
		http.Error(w, "not logged in", 401)
	}
	decoder := json.NewDecoder(r.Body)
	var listAdd models.ReqBookListAdd
	err := decoder.Decode(&listAdd)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if listAdd.List != "read" && listAdd.List != "to_read" && listAdd.List != "library" && listAdd.List != "liked" {
		http.Error(w, "cant add to list", 300)
		return

	}
	err = utils.AddToBookList(id, listAdd)
	if err != nil {
		fmt.Println(err.Error())
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("success")))
	w.Write([]byte("success"))
}

func DeleteFromList(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	if id == 0 {
		http.Error(w, "not logged in", 401)
	}
	decoder := json.NewDecoder(r.Body)
	var listAdd models.ReqBookListAdd
	err := decoder.Decode(&listAdd)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if listAdd.List != "read" && listAdd.List != "to_read" && listAdd.List != "library" && listAdd.List != "liked" {
		http.Error(w, "cant remove to list", 300)
		return
	}
	err = utils.DeleteFromBookList(id, listAdd)
	if err != nil {
		fmt.Println(err.Error())
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("success")))
	w.Write([]byte("success"))
}

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	decoder := json.NewDecoder(r.Body)
	var loginReader models.LoginReader
	err := decoder.Decode(&loginReader)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	reader, err := utils.CheckIfPresent(loginReader.Email)
	if err != nil {
		fmt.Println("uh oh")
		fmt.Println(err.Error())
		http.Error(w, "no user with that email", 401)
		return
		//no user redirect to create account
	}
	if utils.ConfirmPassword(reader.PasswordHash, loginReader.Password) {
		expiry := time.Now().Add(time.Hour * 200)
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
			Name:     "jwt",
			Value:    tokenString,
			Expires:  expiry,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Domain:   "bookplate.co", //add when correct
			Secure:   true,
		})
		js := fmt.Sprintf("%v", reader.ID)
		w.Header().Set("Content-Type", http.DetectContentType([]byte(js)))
		w.Write([]byte(js))
	} else {
		http.Error(w, "wrong password", 401)
		w.Write([]byte("No User"))
	}

}

func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   "bookplate.co",
	})
	w.Write([]byte("Logged Out"))
}

func GetBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	ctx := r.Context()
	book, ok := ctx.Value(utils.BookKey).(models.Book)
	if !ok {
		http.Error(w, "book not found", 404)
		return
	}
	if book.Title == "" {
		http.Error(w, "book not found", 404)
		return
	}
	authors, ok := ctx.Value(utils.AuthorKey).([]models.Author)
	webbook := book.ToResWebBook(authors)
	js := webbook.ToJson()
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func GetReaderBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	bookID := chi.URLParam(r, "bookID")
	ctx := r.Context()
	id, _ := ctx.Value(utils.ReaderKey).(uint)
	list := utils.GetReaderBook(id, bookID)
	js, err := ffjson.Marshal(list)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)

}

func GetReaderProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	readerID := chi.URLParam(r, "readerID")
	ctx := r.Context()
	id, _ := ctx.Value(utils.ReaderKey).(uint)
	reader, err := strconv.ParseUint(readerID, 10, 64)
	fmt.Println(err)
	if err != nil {
		http.Error(w, "readers are not strings", 404)
	}
	status := utils.GetStatus(id, uint(reader))
	js, err := ffjson.Marshal(status)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)

}

func GetAuthor(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	author, ok := ctx.Value(utils.AuthorKey).(models.Author)
	if !ok {
		//errpr
		return
	}
	books, ok := ctx.Value(utils.BookKey).([]models.Book)
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
	defer db.Close()
	if err := db.Find(&records).Error; err != nil {
		fmt.Println(err)
	}
	for _, book := range records {
		web := book.ToAllWebBook()
		webBooks = append(webBooks, web)
	}
	js, err := ffjson.Marshal(webBooks)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	reader, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		//errpr
		return
	}
	profile := utils.GetProfile(reader)
	js, err := ffjson.Marshal(profile)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}

func GetLiked(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	reader, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		//errpr
		return
	}
	profile := utils.GetBookList(reader, len(reader.Liked), func(i int) string { return reader.Liked[i] })
	js, err := ffjson.Marshal(profile)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)

}

func GetRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	reader, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		//errpr
		return
	}
	profile := utils.GetBookList(reader, len(reader.Read), func(i int) string { return reader.Read[i] })
	js, err := ffjson.Marshal(profile)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)

}

func GetToRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	reader, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		//errpr
		return
	}
	profile := utils.GetBookList(reader, len(reader.ToRead), func(i int) string { return reader.ToRead[i] })
	js, err := ffjson.Marshal(profile)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)

}

func GetLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	reader, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		//errpr
		return
	}
	profile := utils.GetBookList(reader, len(reader.Library), func(i int) string { return reader.Library[i] })
	js, err := ffjson.Marshal(profile)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}

func GetFriends(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	friend, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		http.Error(w, "not a person", 404)
		return
	}
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		var pronoun models.Pronoun
		jsonPro := []byte(friend.Pronouns.RawMessage)
		json.Unmarshal(jsonPro, &pronoun)
		w.WriteHeader(401)
		res := models.ResGetFriends{
			Name:          friend.Name,
			ProfileColour: friend.ProfileColour,
			Pronoun:       pronoun.Possessive,
			Friends:       nil,
		}
		js, err := ffjson.Marshal(res)
		if err != nil {
			fmt.Println(err)
		}
		w.Write(js)
		return

	}
	friends := utils.GetFriends(friend, id)
	if friends.Name == "Same person" {
		http.Error(w, "deal with later", 404)
	}
	if friends.Friends == nil {
		w.WriteHeader(403)
		js, err := ffjson.Marshal(friends)
		if err != nil {
			fmt.Println(err)
		}
		w.Write(js)
		return
	}
	js, err := ffjson.Marshal(friends)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}

func GetReaderFriends(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	friend, ok := ctx.Value(utils.ReaderUserKey).(models.Reader)
	if !ok {
		http.Error(w, "not a person", 404)
		return
	}
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		http.Error(w, "not logged in", 401)
		return
	}
	maping, err := utils.GetReaderFriends(friend, id)
	if err != nil {
		if err.Error() == "same person" {
			http.Error(w, "not a person", 404)
			return
		} else if err.Error() == "not mutual friends" {
			http.Error(w, "not mutual friends", 403)
			return
		}
	}
	js, err := ffjson.Marshal(maping)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(js)
}

func AddFriend(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	readerID := chi.URLParam(r, "readerID")
	intReaderID, _ := strconv.ParseUint(readerID, 10, 64)
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	err := utils.AddFriend(uint(intReaderID), id)
	if err != nil {
		http.Error(w, "somthing went wrong", 500)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("good")))

	w.Write([]byte("friend thing did"))
}

func RemoveFriend(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	readerID := chi.URLParam(r, "readerID")
	intReaderID, _ := strconv.ParseUint(readerID, 10, 64)
	db := bdb.Connect()
	reader := models.Reader{}
	not := db.Where(models.Reader{ID: uint(intReaderID)}).First(&reader).RecordNotFound()
	db.Close()
	if not == true {
		http.Error(w, "reader doesn't exist", 404)
		return
	}
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	err := utils.RemoveFriends(uint(intReaderID), id)
	if err != nil {
		http.Error(w, "somthing went wrong", 500)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("good")))

	w.Write([]byte("friend removed"))

}

func BlockReader(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	readerID := chi.URLParam(r, "readerID")
	intReaderID, _ := strconv.ParseUint(readerID, 10, 64)
	db := bdb.Connect()
	reader := models.Reader{}
	not := db.Where(models.Reader{ID: uint(intReaderID)}).First(&reader).RecordNotFound()
	db.Close()
	if not == true {
		http.Error(w, "reader doesn't exist", 404)
		return
	}
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	err := utils.AddBlocked(uint(intReaderID), id)
	if err != nil {
		http.Error(w, "somthing went wrong", 500)
		return
	}
	err = utils.RemoveFriends(uint(intReaderID), id)
	if err != nil {
		http.Error(w, "somthing went wrong", 500)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("good")))

	w.Write([]byte("reader Blocked"))

}
func UnblockReader(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	readerID := chi.URLParam(r, "readerID")
	intReaderID, _ := strconv.ParseUint(readerID, 10, 64)
	db := bdb.Connect()
	reader := models.Reader{}
	not := db.Where(models.Reader{ID: uint(intReaderID)}).First(&reader).RecordNotFound()
	db.Close()
	if not == true {
		http.Error(w, "reader doesn't exist", 404)
		return
	}
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderKey).(uint)
	if !ok {
		return
	}
	err := utils.RemoveBlocked(uint(intReaderID), id)
	if err != nil {
		http.Error(w, "somthing went wrong", 500)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType([]byte("good")))

	w.Write([]byte("reader Unblocked"))

}

func ForgotPasswordRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	type email struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	var e email
	err := decoder.Decode(&e)
	if err != nil {
		log.Println(err.Error())
		return
	}
	err = utils.ForgotPasswordRequest(e.Email)
	if err != nil {
		if errors.As(err, &berrors.NoUserError{}) {
			http.Error(w, err.Error(), 404)
			return
		} else if errors.As(err, &berrors.PasskeyExists{}) {
			http.Error(w, err.Error(), 401)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Email sent"))
}

func ForgotPasswordReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, ok := ctx.Value(utils.ReaderPasswordKey).(uint)
	if !ok {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	}
	type tempPassword struct {
		Password string `json:"password"`
	}
	var temp tempPassword
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&temp)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = utils.ResetPassword(id, temp.Password)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Password Changed"))

}

func SearchBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	pageL := r.URL.Query()["page"]
	var page int
	if len(pageL) == 0 {
		page = 1
	} else {
		page, _ = strconv.Atoi(pageL[0])
	}
	term := chi.URLParam(r, "term")

	results := utils.SearchPage("SELECT title, .word_similarity(books.title, '$1') AS trgm_rank FROM books WHERE title % '$1' ORDER BY trgm_rank DESC", term, uint(page))
	js, err := ffjson.Marshal(results)
	if err != nil {
		log.Println("somthing went wrong")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(js)
}

func SearchAuthors(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	pageL := r.URL.Query()["page"]
	var page int
	if len(pageL) == 0 {
		page = 1
	} else {
		page, _ = strconv.Atoi(pageL[0])
	}
	term := chi.URLParam(r, "term")

	results := utils.SearchPage("SELECT title, word_similarity(books.title, '$1') AS trgm_rank FROM books WHERE title % '$1' ORDER BY trgm_rank DESC", term, uint(page))
	js, err := ffjson.Marshal(results)
	if err != nil {
		log.Println("somthing went wrong")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(js)
}

func SearchAuthor(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Charset", "utf-8")
	w.Header().Set("Content-Type", "application/json")
	pageL := r.URL.Query()["page"]
	var page int
	if len(pageL) == 0 {
		page = 1
	} else {
		page, _ = strconv.Atoi(pageL[0])
	}
	term := chi.URLParam(r, "term")

	results := utils.SearchPage("SELECT name,word_similarity(authors.name, $1)AS trgm_rank FROM authors WHERE name % $1 ORDER BY trgm_rank DESC ", term, uint(page))
	js, err := ffjson.Marshal(results)
	if err != nil {
		log.Println("somthing went wrong")
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
