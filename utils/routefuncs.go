package utils

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/AvraamMavridis/randomcolor"
	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	berror "github.com/GoodByteCo/Bookplate-Backend/errors"
	"github.com/GoodByteCo/Bookplate-Backend/models"
	sq "github.com/Masterminds/squirrel"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pquerna/ffjson/ffjson"
)

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
	log.Println(add.Year)
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
	log.Println(add.Authors)
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
		return 0, nil, berror.UserExistError{Email: add.Email}
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

func GetReaderBook(id uint, book_id string) models.ReqInList {
	db := bdb.Connect()
	defer db.Close()
	var reader models.Reader
	db.Where(&models.Reader{ID: id}).First(&reader)
	inList := models.InternalInList{
		Read:    contains(reader.Read, book_id),
		Liked:   contains(reader.Liked, book_id),
		ToRead:  contains(reader.ToRead, book_id),
		Library: contains(reader.Library, book_id),
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
	pronoun := getPronouns(reader.Pronouns.RawMessage)
	return models.ReqProfile{
		Name:          reader.Name,
		ProfileColour: reader.ProfileColour,
		Pronoun:       pronoun.Possessive,
		FavouriteBook: favBookModel,
		LikedBooks:    booklist,
	}
}

func GetBookList(reader models.Reader, length int, itemGetter func(int) string) models.ReqProfileList {
	db := bdb.Connect()
	defer db.Close()
	var booklist []models.BookForProfile
	for i := length - 1; i >= 0; i-- {
		str := itemGetter(i)
		var book models.Book
		db.Where(models.Book{BookID: str}).Find(&book)
		forProfile := models.BookForProfile{
			BookID:   str,
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
func SearchPage(queryRaw, term string, page uint) []models.ReqSearchResult {
	query := paginatedQuery(queryRaw).addOffset(page)
	db := bdb.Connect()
	defer db.Close()
	results := make([]models.ReqSearchResult, 0, 10)
	fmt.Println(query)
	db.Raw(query, term).Scan(&results)
	log.Println(results)
	return results
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

func GetFriends(friend models.Reader, readerID uint) models.ResGetFriends {
	if readerID == friend.ID {
		return models.ResGetFriends{
			Name: "Same person", // is a hack
		}
	}
	pronoun := getPronouns(friend.Pronouns.RawMessage)
	db := bdb.Connect()
	if !isMutualFriend(readerID, friend.ID, db) {
		return models.ResGetFriends{
			Name:          friend.Name,
			ProfileColour: friend.ProfileColour,
			Pronoun:       pronoun.Possessive,
			Friends:       nil,
		}

	}
	var friends models.Friends
	for _, r := range friend.Friends {
		var fReader models.Reader
		db.Select("id, name, profile_colour").Where(models.Reader{ID: uint(r)}).Find(&fReader)
		friendAdd := models.Friend{
			ID:            fReader.ID,
			Name:          fReader.Name,
			ProfileColour: fReader.ProfileColour,
		}
		friends = append(friends, friendAdd)
	}
	db.Close()
	return models.ResGetFriends{
		Name:          friend.Name,
		ProfileColour: friend.ProfileColour,
		Pronoun:       pronoun.Possessive,
		Friends:       friends,
	}
}

func GetReaderFriends(friend models.Reader, readerID uint) (map[uint]string, error) {
	if friend.ID == readerID {
		return nil, errors.New("same person")
	}
	db := bdb.Connect()
	if !isMutualFriend(readerID, friend.ID, db) {
		return nil, errors.New("not mutual friends")
	}
	db.Close()
	maping := make(map[uint]string)
	for _, f := range friend.Friends {
		status := GetStatus(readerID, uint(f))
		maping[uint(f)] = status.Status
	}
	return maping, nil
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
	if readerID == friendID {
		return models.Status{Status: "You"}
	}
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

func ForgotPasswordRequest(email string) error {
	r, err := CheckIfPresent(email)
	if err != nil {
		return err
	}
	id := r.ID
	ulid := genULID()
	err = addPasswordKey(id, ulid)
	if err != nil {
		return err
	}
	sendForgotPasswordEmail(email, r.Name, ulid)
	return nil
}

func ResetPassword(readerID uint, password string) error {
	hashedPassword, err := HashAndSalt(password)
	if err != nil {
		return err
	}
	db := bdb.Connect()
	defer db.Close()
	db = db.Model(&models.Reader{ID: readerID}).Update(models.Reader{PasswordHash: hashedPassword})
	return db.Error
}
