package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/jpeg"
	png2 "image/png"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/ffjson/ffjson"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/nickalie/go-mozjpegbin"
	"github.com/yusukebe/go-pngquant"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	"github.com/AvraamMavridis/randomcolor"
	sq "github.com/Masterminds/squirrel"
	"github.com/cespare/xxhash"
)

type key string

const (
	ReaderKey key = "reader_id"
	AuthorKey key = "author"
	BookKey   key = "book"
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

func AddToBookList(reader_id uint, listAdd models.ReqBookListAdd) error {
	db := bdb.ConnectToBook()
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	setUpdate := fmt.Sprintf("array_append(%s, '%s')", listAdd.List, listAdd.BookID)
	fmt.Println(setUpdate)
	sql, test, err := psql.Update("readers").Set(listAdd.List, setUpdate).Where("ID = ?", reader_id).ToSql()
	fmt.Println(test)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println(sql)
	sql = strings.Replace(sql, "$1", setUpdate, 1)
	sql = strings.Replace(sql, "$2", "$1", 1)
	db = db.Exec(sql, reader_id)
	return db.Error
}

func AddBook(add models.ReqWebBook, reader_id uint) (string, error) {
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
		BookID:      "",
		Title:       add.Title,
		Year:        year,
		Description: add.Description,
		CoverURL:    add.CoverUrl,
		ReaderID:    uint(reader_id), //do thing where i get reader added
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
		DeletedAt:   nil,
	}
	book.SetStringId()
	return book.BookID, db.Create(&book).Association("authors").Append(authors).Error
}

func AddReader(add models.ReqReader) (uint uint, error, usererror error) {
	emailHash := HashEmail(add.Email)
	_, noUser := GetReaderFromDB(emailHash)
	if !noUser {
		return 0, nil, UserExistError{add.Email}
	}
	passwordHash, err := HashAndSalt(add.Password)
	if err != nil {
		return 0, err, nil
	}

	psPronouns, err := ffjson.Marshal(add.Pronouns)
	if err != nil {
		return 0, err, nil
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
		return 0, dbc.Error, nil
	}

	return reader.ID, nil, nil
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

func GetReaderBook(id uint, book_id string) models.ReqInList {
	db := bdb.Connect()
	var reader models.Reader
	db.Where(&models.Reader{ID: id}).First(&reader)
	sort.Strings(reader.Library)
	sort.Strings(reader.Read)
	sort.Strings(reader.ToRead)
	sort.Strings(reader.Liked)
	inList := models.InternalInList{
		Read:    binarySearch(book_id, reader.Read),
		Liked:   binarySearch(book_id, reader.Liked),
		ToRead:  binarySearch(book_id, reader.ToRead),
		Library: binarySearch(book_id, reader.Library),
	}
	var in models.Friends
	db.Raw("select readers.ID, readers.name, readers.profile_colour from readers inner join (select friends from readers where ID = $1) vtable on readers.id = ANY (vtable.friends) WHERE readers.library @> ARRAY['$2']::VARCHAR[]", id, book_id).Scan(&in)
	fmt.Println(inList)
	fmt.Println(in)

	finalList := models.ReqInList{
		Read:    inList.Read,
		Liked:   inList.Liked,
		ToRead:  inList.ToRead,
		Library: inList.Library,
		Friends: in,
	}
	return finalList
}

func binarySearch(searchWord string, list []string) bool {

	low := 0
	high := len(list) - 1

	for low <= high {
		median := (low + high) / 2

		if list[median] < searchWord {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == len(list) || list[low] != searchWord {
		return false
	}

	return true
}
