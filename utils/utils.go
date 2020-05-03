package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
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
	defer db.Close()
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	setUpdate := fmt.Sprintf("array_append(%s, '%s')", listAdd.List, listAdd.BookID)
	fmt.Println(setUpdate)
	sql, _, err := psql.Update("readers").Set(listAdd.List, setUpdate).Where("ID = ?", reader_id).ToSql()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	sql = strings.Replace(sql, "$1", setUpdate, 1)
	sql = strings.Replace(sql, "$2", "$1", 1)
	fmt.Println(sql)
	db = db.Exec(sql, reader_id)
	if listAdd.List == "liked" {
		// checks is read if not add to read
		type temp struct {
			ID int
		}
		var tempid temp
		db.Raw("SELECT id from readers WHERE read @> ARRAY[$1]::VARCHAR[] AND ID = $2", listAdd.BookID, reader_id).Scan(&tempid)
		fmt.Println(tempid)
		if tempid.ID == 0 {
			AddToBookList(reader_id, models.ReqBookListAdd{List: "read", BookID: listAdd.BookID})
		}
	} else if listAdd.List == "read" {
		DeleteFromBookList(reader_id, models.ReqBookListAdd{List: "to_read", BookID: listAdd.BookID})
	}
	return db.Error
}

func DeleteFromBookList(reader_id uint, listAdd models.ReqBookListAdd) error {
	db := bdb.ConnectToBook()
	defer db.Close()
	sql, err := genArrayModifySQL(remove, listAdd.List, listAdd.BookID, reader_id)
	if err != nil {
		return err
	}
	fmt.Println(sql)
	db = db.Exec(sql, reader_id)
	return db.Error
}

func AddBook(add models.ReqWebBook, reader_id uint) (string, error) {
	fmt.Println(add.Year)
	db := bdb.Connect()
	defer db.Close()
	authors := add.Authors
	for i, a := range authors {
		if a.AuthorId == "" {
			a.SetStringId()
			authors[i] = a
		}
	}
	year, err := strconv.Atoi(add.Year)
	if err != nil {
		return "", err
	}
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
	defer db.Close()
	reader := models.Reader{
		Name:           add.Name,
		Pronouns:       pronouns,
		Library:        []string{},
		ToRead:         []string{},
		Liked:          []string{},
		Friends:        []int64{},
		Read:           []string{},
		FriendsPending: []int64{},
		FriendsRequest: []int64{},
		ReaderBlocked:  []int64{},
		ProfileColour:  randomcolor.GetRandomColorInHex(),
		PasswordHash:   passwordHash,
		EmailHash:      emailHash,
		Plural:         false,
	}
	if dbc := db.Create(&reader); dbc.Error != nil {
		return 0, dbc.Error, nil
	}

	return reader.ID, nil, nil
}

func GetReaderFromDB(emailHash int64) (models.Reader, bool) {
	db := bdb.ConnectToReader()
	defer db.Close()
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
	defer db.Close()
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
	db.Raw("select readers.ID, readers.name, readers.profile_colour from readers inner join (select friends from readers where ID = $1) vtable on readers.id = ANY (vtable.friends) WHERE readers.library @> ARRAY[$2]::VARCHAR[]", id, book_id).Scan(&in)
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

func GetProfile(reader models.Reader) models.ReqProfile {
	db := bdb.Connect()
	defer db.Close()
	var favBook models.Book
	db.Where(models.Book{BookID: reader.FavouriteBook}).Find(&favBook)
	var booklist []models.BookForProfile
	for i := range reverse(reader.Liked) {
		if i.int >= 5 {
			break
		}
		var book models.Book
		db.Where(models.Book{BookID: i.string}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   i.string,
			Title:    book.Title,
			CoverURL: book.CoverURL,
		}
		booklist = append(booklist, forProfile)
	}
	favBookModel := models.FavouriteBook{
		BookID: favBook.BookID,
		Title:  favBook.Title,
	}
	var pronoun models.Pronoun
	jsonPro := []byte(reader.Pronouns.RawMessage)
	json.Unmarshal(jsonPro, &pronoun)
	return models.ReqProfile{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		Pronoun:       pronoun.Possessive,
		FavouriteBook: favBookModel,
		LikedBooks:    booklist,
	}
}

func GetLiked(reader models.Reader) models.ReqProfileList {
	db := bdb.Connect()
	defer db.Close()
	var booklist []models.BookForProfile
	for i := range reverse(reader.Liked) {
		var book models.Book
		db.Where(models.Book{BookID: i.string}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   i.string,
			Title:    book.Title,
			CoverURL: book.CoverURL,
		}
		booklist = append(booklist, forProfile)
	}
	return models.ReqProfileList{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		BookList:      booklist,
	}

}

func GetRead(reader models.Reader) models.ReqProfileList {
	db := bdb.Connect()
	defer db.Close()
	var booklist []models.BookForProfile
	for i := range reverse(reader.Read) {
		var book models.Book
		db.Where(models.Book{BookID: i.string}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   i.string,
			Title:    book.Title,
			CoverURL: book.CoverURL,
		}
		booklist = append(booklist, forProfile)
	}
	return models.ReqProfileList{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		BookList:      booklist,
	}

}

func GetToRead(reader models.Reader) models.ReqProfileList {
	db := bdb.Connect()
	defer db.Close()
	var booklist []models.BookForProfile
	for i := range reverse(reader.ToRead) {
		var book models.Book
		db.Where(models.Book{BookID: i.string}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   i.string,
			Title:    book.Title,
			CoverURL: book.CoverURL,
		}
		booklist = append(booklist, forProfile)
	}
	return models.ReqProfileList{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		BookList:      booklist,
	}

}

func GetLibrary(reader models.Reader) models.ReqProfileList {
	db := bdb.Connect()
	defer db.Close()
	var booklist []models.BookForProfile
	for i := range reverse(reader.Library) {
		var book models.Book
		db.Where(models.Book{BookID: i.string}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   i.string,
			Title:    book.Title,
			CoverURL: book.CoverURL,
		}
		booklist = append(booklist, forProfile)
	}
	return models.ReqProfileList{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		BookList:      booklist,
	}

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

func RemoveFriends(friendID uint, readerID uint) error {
	friend := strconv.FormatUint(uint64(friendID), 10)
	reader := strconv.FormatUint(uint64(readerID), 10)

	sqlFf, err := genArrayModifySQL(remove, "friends", reader, friendID)
	if err != nil {
		return err
	}
	sqlFr, err := genArrayModifySQL(remove, "friends", friend, readerID)
	if err != nil {
		return err
	}
	sqlRf, err := genArrayModifySQL(remove, "friends_request", reader, friendID)
	if err != nil {
		return err
	}
	sqlRr, err := genArrayModifySQL(remove, "friends_request", friend, readerID)
	if err != nil {
		return err
	}
	sqlPf, err := genArrayModifySQL(remove, "friends_pending", reader, friendID)
	if err != nil {
		return err
	}
	sqlPr, err := genArrayModifySQL(remove, "friends_pending", friend, readerID)
	if err != nil {
		return err
	}

	db := bdb.Connect()
	defer db.Close()

	db = db.Exec(sqlFf, friendID)
	db = db.Exec(sqlRf, friendID)
	db = db.Exec(sqlPf, friendID)
	db = db.Exec(sqlFr, readerID)
	db = db.Exec(sqlRr, readerID)
	db = db.Exec(sqlPr, readerID)
	return db.Error

}

func AddBlocked(friendID uint, readerID uint) error {
	friend := strconv.FormatUint(uint64(friendID), 10)

	sql, err := genArrayModifySQL(add, "reader_blocked", friend, readerID)
	if err != nil {
		return err
	}
	db := bdb.Connect()
	defer db.Close()

	db = db.Exec(sql, readerID)
	return db.Error
}

func RemoveBlocked(friendID uint, readerID uint) error {
	friend := strconv.FormatUint(uint64(friendID), 10)

	sql, err := genArrayModifySQL(remove, "reader_blocked", friend, readerID)
	if err != nil {
		return err
	}
	db := bdb.Connect()
	defer db.Close()

	db = db.Exec(sql, readerID)
	return db.Error
}

func AddFriend(friendID uint, readerID uint) error {
	friend := strconv.FormatUint(uint64(friendID), 10)
	reader := strconv.FormatUint(uint64(readerID), 10)
	// if responsed to request
	type temp struct {
		ID uint
	}
	var tempid temp
	db := bdb.Connect()
	fmt.Println(friendID)
	fmt.Println(readerID)
	db.Raw("SELECT id from readers WHERE friends_request @> ARRAY[$1]::INT[] AND ID = $2", friendID, readerID).Scan(&tempid)
	db.Close()
	fmt.Println(tempid)
	if tempid.ID != 0 {
		//add to friends for both
		sqlR, err := genArrayModifySQL(add, "friends", friend, readerID)
		if err != nil {
			return err
		}
		sqlF, err := genArrayModifySQL(add, "friends", reader, friendID)
		if err != nil {
			return err
		}
		//remove from pending from friend
		sqlRemPend, err := genArrayModifySQL(remove, "friends_pending", reader, friendID)
		//remove from requested from you
		sqlRemReq, err := genArrayModifySQL(remove, "friends_request", friend, readerID)

		db := bdb.Connect()
		defer db.Close()
		db = db.Exec(sqlR, readerID)
		db = db.Exec(sqlF, friendID)
		db = db.Exec(sqlRemPend, friendID)
		db = db.Exec(sqlRemReq, readerID)
		return db.Error
	} else { // if requesting
		sqlPending, err := genArrayModifySQL(add, "friends_pending", friend, readerID)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		sqlRequest, err := genArrayModifySQL(add, "friends_request", reader, friendID)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		db := bdb.Connect()
		defer db.Close()
		db = db.Exec(sqlPending, readerID)
		db = db.Exec(sqlRequest, friendID)
		return db.Error
	}
}

func GetStatus(readerID uint, friendID uint) models.Status {
	db := bdb.Connect()
	defer db.Close()
	if hasBlocked(readerID, friendID, db) {
		return models.Status{Status: "Unblock"}
	} else if blockedBy(readerID, friendID, db) {
		return models.Status{Status: "Add Friend"}
	} else if isMutualFriend(readerID, friendID, db) {
		return models.Status{Status: "Remove Friend"}
	} else if isPending(readerID, friendID, db) {
		return models.Status{Status: "Pending"}
	} else if isRequested(readerID, friendID, db) {
		return models.Status{Status: "Accept Friend"}
	} else {
		return models.Status{Status: "Add Friend"}
	}
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
