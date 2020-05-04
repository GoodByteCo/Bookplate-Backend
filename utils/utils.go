package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image/jpeg"
	png2 "image/png"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/nickalie/go-mozjpegbin"
	"github.com/yusukebe/go-pngquant"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/cespare/xxhash"
	"github.com/jinzhu/gorm"
)

type key string

type arrayMod int

const (
	add arrayMod = iota
	remove
)

func (a arrayMod) String() string {
	return [...]string{"add", "remove"}[a]
}

func genArrayModifySQL(a arrayMod, changing string, toChange string, reader uint) (string, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	switch a {
	case add:
		set := fmt.Sprintf("array_append(%s, '%s')", changing, toChange)
		fmt.Println(set)
		sql, _, err := psql.Update("readers").Set(changing, set).Where("ID = ?", reader).ToSql()
		if err != nil {
			fmt.Println(err.Error())
			return "", err
		}
		sql = strings.Replace(sql, "$1", set, 1)
		sql = strings.Replace(sql, "$2", "$1", 1)
		return sql, nil
	case remove:
		set := fmt.Sprintf("array_remove(%s, '%s')", changing, toChange)
		fmt.Println(set)
		sql, _, err := psql.Update("readers").Set(changing, set).Where("ID = ?", reader).ToSql()
		if err != nil {
			fmt.Println(err.Error())
			return "", err
		}
		fmt.Println(sql)
		sql = strings.Replace(sql, "$1", set, 1)
		sql = strings.Replace(sql, "$2", "$1", 1)
		return sql, nil
	}
	return "", errors.New("error")
}

const (
	ReaderKey     key = "reader_id"
	AuthorKey     key = "author"
	BookKey       key = "book"
	ReaderUserKey key = "reader"
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

func MutualFriends(id uint) {
	db := bdb.Connect()
	defer db.Close()
	db.Raw("select readers.ID, readers.name, readers.profile_colour from readers inner join (select ID,friends from readers where ID = $1) as vtable on ARRAY[readers.id] @> (vtable.friends) WHERE ARRAY[vtable.id] @> (readers.friends)", id)
}

func isMutualFriend(readerID uint, friendID uint, db *gorm.DB) bool { // 3
	type temp struct {
		ID uint
	}
	var tempid temp
	db.Raw("select readers.ID from readers inner join (select ID,friends from readers where ID = $1) as vtable on ARRAY[readers.id] <@ (vtable.friends) WHERE ARRAY[vtable.id] <@ (readers.friends) AND readers.ID = $2", readerID, friendID).Scan(&tempid)
	if tempid.ID != 0 {
		return true
	}
	return false
}

func hasBlocked(readerID uint, friendID uint, db *gorm.DB) bool { // 1
	type temp struct {
		ID uint
	}
	var tempid temp
	db.Raw("select ID from readers where ARRAY[$2]::INT[] <@ reader_blocked and ID = $1", readerID, friendID).Scan(&tempid)
	if tempid.ID != 0 {
		return true
	}
	return false
}

func blockedBy(readerID uint, friendID uint, db *gorm.DB) bool { // 2
	type temp struct {
		ID uint
	}
	var tempid temp
	db.Raw("select ID from readers where ARRAY[$1]::INT[] <@ reader_blocked and ID = $2", readerID, friendID).Scan(&tempid)
	if tempid.ID != 0 {
		return true
	}
	return false

}

func isPending(readerID uint, friendID uint, db *gorm.DB) bool { // 4
	type temp struct {
		ID uint
	}
	var tempid temp
	db.Raw("select readers.ID from readers inner join (select ID,friends_pending from readers where ID = $1) as vtable on ARRAY[readers.id] <@ (vtable.friends_pending) WHERE ARRAY[vtable.id] <@ (readers.friends_request) AND readers.ID = $2", readerID, friendID).Scan(&tempid)
	if tempid.ID != 0 {
		return true
	}
	return false

}

func isRequested(readerID uint, friendID uint, db *gorm.DB) bool { //5
	type temp struct {
		ID uint
	}
	var tempid temp
	db.Raw("select readers.ID from readers inner join (select ID,friends_request from readers where ID = $1) as vtable on ARRAY[readers.id] <@ (vtable.friends_request) WHERE ARRAY[vtable.id] <@ (readers.friends_pending) AND readers.ID = $2", readerID, friendID).Scan(&tempid)
	if tempid.ID != 0 {
		return true
	}
	return false
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

func reverse(lst []string) chan struct {
	int
	string
} {
	ret := make(chan struct {
		int
		string
	})
	go func() {
		for i := range lst {
			ret <- struct {
				int
				string
			}{i, lst[len(lst)-1-i]}

		}
		close(ret)
	}()
	return ret
}

func sendForgotPasswordEmail() {
	from := mail.NewEmail("Example User", "test@example.com")
	subject := "Sending with SendGrid is Fun"
	to := mail.NewEmail("Example User", "test@example.com")
	plainTextContent := "and easy to do anywhere, even with Go"
	htmlContent := "<strong>and easy to do anywhere, even with Go</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
