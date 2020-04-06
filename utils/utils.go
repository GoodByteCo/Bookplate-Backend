package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/jpeg"
	png2 "image/png"
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nickalie/go-mozjpegbin"
	pngquant "github.com/yusukebe/go-pngquant"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	"github.com/AvraamMavridis/randomcolor"
	"github.com/cespare/xxhash"
)

var TokenAuth *jwtauth.JWTAuth

var Issuer string
var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	Issuer = os.Getenv("ISSUER")
	TokenAuth = jwtauth.New("HS256", []byte(os.Getenv("TOKENSECRET")), nil)
}

type UserExistError struct {
	email string
}

func (e UserExistError) Error() string {
	return fmt.Sprintf("User with %s email exists", e.email)
}

type NoUserError struct {
	email string
}

func (e NoUserError) Error() string {
	return fmt.Sprintf("No User with %s email exists", e.email)

}

func CompressPng(png io.Reader) io.Reader {
	img, err := png2.Decode(png)
	if err != nil {
		panic(err)
	}
	out := new(bytes.Buffer)
	cimg, err := pngquant.Compress(img, "1")
	if err != nil {
		panic(err)
	}
	err = png2.Encode(out, cimg)
	return out
}

func CompressJpg(jpg io.Reader) io.Reader {
	img, err := jpeg.Decode(jpg)
	if err != nil {
		panic(err)
	}
	out := new(bytes.Buffer)
	err = mozjpegbin.Encode(out, img, &mozjpegbin.Options{
		Quality:  80,
		Optimize: true,
	})
	if err != nil {
		panic(err)
	}
	return out
}

func HashAndSalt(str string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(str), 7)
	return string(hash), err
}

func HashEmail(str string) int64 {
	h := xxhash.New()
	h.Write([]byte(str))
	bs := h.Sum(nil)
	r := binary.BigEndian.Uint64(bs)
	fmt.Println(int64(r))
	return int64(r)
}

func GetClaim(ctx context.Context) jwt.MapClaims {
	_, claims, _ := jwtauth.FromContext(ctx)
	return claims
}

func CompareEmail(hashEmail string, email string) bool {
	hashEmailBytes := []byte(hashEmail)
	emailBytes := []byte(email)
	err := bcrypt.CompareHashAndPassword(hashEmailBytes, emailBytes)
	if err != nil {
		return false
	}
	return true
}

func ConfirmPassword(hashPassword string, password string) bool {
	hashPassBytes := []byte(hashPassword)
	passBytes := []byte(password)
	err := bcrypt.CompareHashAndPassword(hashPassBytes, passBytes)
	if err != nil {
		return false
	}
	return true
}

func CheckIfPresent(email string) (models.Reader, error) {
	emailHash := HashEmail(email)
	fmt.Println(emailHash)
	reader, noUser := GetReaderFromDB(emailHash)
	if noUser {
		return models.Reader{}, NoUserError{email}
	}
	return reader, nil
}

//func AddAuthor(add models.Author) error{
//}

func AddBook(add models.WebBook) error {
	fmt.Println(add.Year)
	db := bdb.Connect()
	authors := add.Authors
	for i, a := range authors {
		if a.AuthorId == "" {
			a.SetStringId()
			authors[i] = a
		}
	}
	year, _ := strconv.Atoi(add.Year)
	fmt.Println(add.Authors)
	book := models.Book{
		BookId:        "",
		Title:         add.Title,
		Year:          year,
		Description:   add.Description,
		CoverUrl:      add.CoverUrl,
		ReaderID: 0, //do thing where i get reader added
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
		DeletedAt:     nil,
	}
	book.SetStringId()
	fmt.Println("+++++")
	fmt.Println(book)
	return db.Create(&book).Association("authors").Append(authors).Error
}

func AddReader(add models.ReaderAdd) (error, usererror error) {
	emailHash := HashEmail(add.Email)
	_, noUser := GetReaderFromDB(emailHash)
	if !noUser {
		return nil, UserExistError{add.Email}
	}
	passwordHash, err := HashAndSalt(add.Password)
	if err != nil {
		return err, nil
	}

	psPronouns, err := json.Marshal(add.Pronouns)
	if err != nil {
		return err, nil
	}
	pronouns := postgres.Jsonb{RawMessage: psPronouns}
	db := bdb.ConnectToReader()
	reader := models.Reader{
		Name:          add.Name,
		Pronouns:      pronouns,
		Library:       []string{},
		ToRead:        []string{},
		Liked:         []string{},
		Friends:       []int64{},
		ProfileColour: randomcolor.GetRandomColorInHex(),
		PasswordHash:  passwordHash,
		EmailHash:     emailHash,
		Plural:        false,
	}
	if dbc := db.Create(&reader); dbc.Error != nil {
		return dbc.Error, nil
	}
	return nil, nil
}

func GetReaderFromDB(emailHash int64) (models.Reader, bool) {
	db := bdb.ConnectToReader()
	emptyReader := models.Reader{}
	found := db.Where(models.Reader{EmailHash: emailHash}).Find(&emptyReader).RecordNotFound()
	defer db.Close()
	return emptyReader, found
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
