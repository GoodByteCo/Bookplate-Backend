package routes

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/GoodByteCo/Bookplate-Backend/Models"
	"github.com/GoodByteCo/Bookplate-Backend/utils"
	"gopkg.in/kothar/go-backblaze.v0"
)

func AddBook(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(r.Header)
	r.ParseMultipartForm(32 << 20)
	//fmt.Println(r.Form)
	//fmt.Println(reflect.TypeOf(r.Form))
	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	names := strings.Split(header.Filename, ".")
	fmt.Printf("File name %s.%s\n", names[0], names[1])
	name := utils.String(32)
	name = fmt.Sprintf("%s.%s", name, names[1])
	//maybe make random later
	fmt.Println(name)
	bookplateBucket := getBucket()
	metadata := make(map[string]string)
	b2file, err := bookplateBucket.UploadFile(name, metadata, file)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(b2file.UploadTimestamp)
	url, err := bookplateBucket.FileURL(name)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(url)
	b := Models.UrlValueToBook(r.Form, url)
	db := utils.ConnectToBook()
	emptyBook := Models.Book{}
	val := 1
	orginalId := b.BookId
	for !db.Where(Models.Book{BookId: b.BookId}).Find(&emptyBook).RecordNotFound() {
		b.BookId = fmt.Sprintf("%s%d", orginalId, val)
		val += 1
		emptyBook = Models.Book{}
	}
	db.Create(&b)

}

func GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	book, ok := ctx.Value("book").(Models.Book)
	if !ok {
		//errpr
		return
	}
	fmt.Println(book)
	webbook := book.ToWebBook()
	js := webbook.ToJson()
	fmt.Println(string(js))
	w.Header().Set("Content-Type", "application/json")
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
