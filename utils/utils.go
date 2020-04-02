package utils

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"os"
	"time"

	"github.com/GoodByteCo/Bookplate-Backend/models"

	"github.com/AvraamMavridis/randomcolor"
	"github.com/cespare/xxhash"
	"github.com/jinzhu/gorm"
)

var TokenAuth *jwtauth.JWTAuth

var Issuer string
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	Issuer = os.Getenv("ISSUER")
	TokenAuth = jwtauth.New("HS256", []byte(os.Getenv("TOKENSECRET")), nil)

}

type UserExistError struct{
	email string
}

func (e UserExistError) Error() string{
	return fmt.Sprintf("User with %s email exists", e.email)
}

type NoUserError struct {
	email string
}


func (e NoUserError) Error() string {
	return fmt.Sprintf("No User with %s email exists", e.email)

}


func connect() *gorm.DB {
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=quinnpollock dbname=BookPlateGo password=bookplate sslmode=disable")
	if err != nil {
		panic("DB Down")
		fmt.Println(err)
	}
	return db
}

func HashAndSalt(str string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(str), 7)
	return string(hash), err
}

func HashEmail(str string) int64{
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
	db := ConnectToReader()
	reader := models.Reader{
		Name:          add.Name,
		Pronouns:      pronouns,
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

func Migrate() {
	db := connect()
	fmt.Println()
	db.AutoMigrate(&models.Reader{}, &models.Book{}, &models.Author{})
	db.Close()
}

func ConnectToBook() *gorm.DB {
	db := connect()
	return db.Table("books")
}

func ConnectToReader() *gorm.DB {
	db := connect()
	return db.Table("readers")
}

func GetReaderFromDB(emailHash int64) (models.Reader, bool) {
	db := ConnectToReader()
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


