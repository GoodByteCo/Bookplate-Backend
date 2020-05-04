package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	bdb "github.com/GoodByteCo/Bookplate-Backend/db"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"gopkg.in/gormigrate.v1"
)

//Book type for database
type Book struct {
	ID          uint   `gorm:"primary_key" json:"-"`
	BookID      string `gorm:"unique"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Description string `gorm:"type:text"`
	CoverURL    string
	BookColor   string
	ReaderID    uint
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `sql:"index" json:"-"`
	Authors     []Author   `gorm:"many2many:book_authors;"`
	BooknameCol string     `type:"tsvector"`
}

//ToUrlSafe Remove non url safe characters from book title and set it as Id
func (b *Book) ToUrlSafe() {
	bookID := b.Title
	bookID = strings.ToLower(bookID)
	bookID = strings.ReplaceAll(bookID, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	bookID = reg.ReplaceAllString(bookID, "")
	b.BookID = bookID
}

//SetStringId Find if Book Id exist and append number if so
func (b *Book) SetStringId() {
	db := bdb.ConnectToBook()
	b.ToUrlSafe()
	emptyBook := Book{}
	val := 1
	orginalId := b.BookID
	for !db.Where(Book{BookID: b.BookID}).Find(&emptyBook).RecordNotFound() {
		b.BookID = fmt.Sprintf("%s%d", orginalId, val)
		val++
		emptyBook = Book{}
	}
}

//Reader type for database
type Reader struct {
	ID             uint       `gorm:"primary_key" json:"-"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`
	DeletedAt      *time.Time `sql:"index" json:"-"`
	Name           string
	Pronouns       postgres.Jsonb
	ProfileColour  string
	Library        pq.StringArray `gorm:"type:varchar(64)[]"`
	ToRead         pq.StringArray `gorm:"type:varchar(64)[]"`
	Liked          pq.StringArray `gorm:"type:varchar(64)[]"`
	Read           pq.StringArray `gorm:"type:varchar(64)[]"`
	Friends        pq.Int64Array  `gorm:"type:integer[]"`
	FriendsPending pq.Int64Array  `gorm:"type:integer[]"`
	FriendsRequest pq.Int64Array  `gorm:"type:integer[]"`
	ReaderBlocked  pq.Int64Array  `gorm:"type:integer[]"`
	PasswordHash   string
	EmailHash      int64
	Plural         bool
	FavouriteBook  string
	Books          []Book //Book added by reader
	ForgotPassword ForgotPassword
}

//Author type for database
type Author struct {
	ID        uint       `gorm:"primary_key" json:"-"`
	AuthorId  string     `gorm:"unique" json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
	Books     []Book     `gorm:"many2many:book_authors;"`
}

//Remove non url safe characters from author name and set it as Id
func (a *Author) ToUrlSafe() {
	authorId := a.Name
	authorId = strings.ToLower(authorId)
	authorId = strings.ReplaceAll(authorId, " ", "-")
	reg, _ := regexp.Compile("[^a-zA-Z0-9\\-]+")
	authorId = reg.ReplaceAllString(authorId, "")
	a.AuthorId = authorId
}

//Find if BookId exist and append number if so
func (a *Author) SetStringId() {
	db := bdb.ConnectToAuthor()
	a.ToUrlSafe()
	emptyAuthor := Author{}
	val := 1
	originalId := a.AuthorId
	for !db.Where(Author{AuthorId: a.AuthorId}).Find(&emptyAuthor).RecordNotFound() {
		a.AuthorId = fmt.Sprintf("%s%d", originalId, val)
		val += 1
		emptyAuthor = Author{}
	}
}

type ForgotPassword struct {
	gorm.Model
	ReaderID  uint
	RandomKey string
}

//Migration Function Update as database structs change
func Start(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "initial",
			Migrate: func(tx *gorm.DB) error {
				type Book struct {
					ID          uint   `gorm:"primary_key" json:"-"`
					BookId      string `gorm:"unique"`
					Title       string
					Year        int32  `json:"year"`
					Description string `gorm:"type:text"`
					CoverUrl    string
					ReaderID    uint
					CreatedAt   time.Time  `json:"-"`
					UpdatedAt   time.Time  `json:"-"`
					DeletedAt   *time.Time `sql:"index" json:"-"`
					Authors     []Author   `gorm:"many2many:book_authors;"`
				}

				type Reader struct {
					ID            uint       `gorm:"primary_key" json:"-"`
					CreatedAt     time.Time  `json:"-"`
					UpdatedAt     time.Time  `json:"-"`
					DeletedAt     *time.Time `sql:"index" json:"-"`
					Name          string
					Pronouns      postgres.Jsonb
					ProfileColour string
					Library       pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead        pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked         pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends       pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash  string
					EmailHash     int64
					Plural        bool
					Books         []Book `gorm:"foreignkey:ReaderAddedId"` //Book added by reader
				}

				type Author struct {
					ID        uint       `gorm:"primary_key" json:"-"`
					AuthorId  string     `gorm:"unique" json:"id"`
					Name      string     `json:"name"`
					CreatedAt time.Time  `json:"-"`
					UpdatedAt time.Time  `json:"-"`
					DeletedAt *time.Time `sql:"index" json:"-"`
					Books     []Book     `gorm:"many2many:book_authors;"`
				}

				return tx.CreateTable(&Reader{}, Author{}, &Book{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable(&Reader{}, Author{}, &Book{}).Error
			},
		},
		{
			ID: "Add Color",
			Migrate: func(tx *gorm.DB) error {
				type Book struct {
					ID          uint   `gorm:"primary_key" json:"-"`
					BookId      string `gorm:"unique"`
					Title       string `json:"title"`
					Year        int    `json:"year"`
					Description string `gorm:"type:text"`
					CoverUrl    string
					BookColor   string
					ReaderID    uint
					CreatedAt   time.Time  `json:"-"`
					UpdatedAt   time.Time  `json:"-"`
					DeletedAt   *time.Time `sql:"index" json:"-"`
					Authors     []Author   `gorm:"many2many:book_authors;"`
				}
				return tx.AutoMigrate(&Book{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropColumn("book_color").Error
			},
		},
		{
			ID: "Add Read",
			Migrate: func(tx *gorm.DB) error {
				type Reader struct {
					ID            uint       `gorm:"primary_key" json:"-"`
					CreatedAt     time.Time  `json:"-"`
					UpdatedAt     time.Time  `json:"-"`
					DeletedAt     *time.Time `sql:"index" json:"-"`
					Name          string
					Pronouns      postgres.Jsonb
					ProfileColour string
					Library       pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead        pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked         pq.StringArray `gorm:"type:varchar(64)[]"`
					Read          pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends       pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash  string
					EmailHash     int64
					Plural        bool
					Books         []Book `gorm:"foreignkey:ReaderAddedId"` //Book added by reader
				}
				return tx.AutoMigrate(&Reader{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&Reader{}).DropColumn("read").Error
			},
		},
		{
			ID: "Add book search",
			Migrate: func(tx *gorm.DB) error {
				type Book struct {
					ID          uint   `gorm:"primary_key" json:"-"`
					BookID      string `gorm:"unique"`
					Title       string `json:"title"`
					Year        int    `json:"year"`
					Description string `gorm:"type:text"`
					CoverURL    string
					BookColor   string
					ReaderID    uint
					CreatedAt   time.Time  `json:"-"`
					UpdatedAt   time.Time  `json:"-"`
					DeletedAt   *time.Time `sql:"index" json:"-"`
					Authors     []Author   `gorm:"many2many:book_authors;"`
					BooknameCol string     `type:"tsvector"`
				}
				tx = tx.Exec("ALTER TABLE books ADD COLUMN bookname_col tsvector GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, ''))) STORED;")
				tx = tx.Model(&Book{}).AddIndex("idx_bookname", "bookname_col ")
				return tx.Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx = tx.Model(&Book{}).RemoveIndex("idx_bookname")
				return tx.DropColumn("bookname_col").Error
			},
		},
		{
			ID: "Add Favourite Book",
			Migrate: func(tx *gorm.DB) error {
				type Reader struct {
					ID            uint       `gorm:"primary_key" json:"-"`
					CreatedAt     time.Time  `json:"-"`
					UpdatedAt     time.Time  `json:"-"`
					DeletedAt     *time.Time `sql:"index" json:"-"`
					Name          string
					Pronouns      postgres.Jsonb
					ProfileColour string
					Library       pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead        pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked         pq.StringArray `gorm:"type:varchar(64)[]"`
					Read          pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends       pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash  string
					EmailHash     int64
					Plural        bool
					FavouriteBook string
					Books         []Book `gorm:"foreignkey:ReaderAddedId"` //Book added by reader
				}
				return tx.AutoMigrate(&Reader{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Model(&Reader{}).DropColumn("favourite_book").Error
			},
		},
		{
			ID: "Add array indexs",
			Migrate: func(tx *gorm.DB) error {
				tx = tx.Exec("CREATE INDEX friends_idx ON readers USING GIN (friends)")
				tx = tx.Exec("CREATE INDEX read_idx ON readers USING GIN (read)")
				tx = tx.Exec("CREATE INDEX to_read_idx ON readers USING GIN (to_read)")
				tx = tx.Exec("CREATE INDEX liked_idx ON readers USING GIN (liked)")
				return tx.Exec("CREATE INDEX library_idx ON readers USING GIN (library)").Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx = tx.Exec("DROP INDEX friends_idx")
				tx = tx.Exec("DROP INDEX read_idx")
				tx = tx.Exec("DROP INDEX to_read_idx")
				tx = tx.Exec("DROP INDEX liked_idx")
				return tx.Exec("DROP INDEX library_idx").Error
			},
		},
		{
			ID: "Add Friend Info",
			Migrate: func(tx *gorm.DB) error {
				type Reader struct {
					ID             uint       `gorm:"primary_key" json:"-"`
					CreatedAt      time.Time  `json:"-"`
					UpdatedAt      time.Time  `json:"-"`
					DeletedAt      *time.Time `sql:"index" json:"-"`
					Name           string
					Pronouns       postgres.Jsonb
					ProfileColour  string
					Library        pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead         pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked          pq.StringArray `gorm:"type:varchar(64)[]"`
					Read           pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends        pq.Int64Array  `gorm:"type:integer[]"`
					FriendsPending pq.Int64Array  `gorm:"type:integer[]"`
					FriendsRequest pq.Int64Array  `gorm:"type:integer[]"`
					ReaderBlocked  pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash   string
					EmailHash      int64
					Plural         bool
					FavouriteBook  string
					Books          []Book //Book added by reader
				}
				tx = tx.AutoMigrate(&Reader{})
				tx = tx.Exec("CREATE INDEX friends_pending_idx ON readers USING GIN (friends_pending)")
				tx = tx.Exec("CREATE INDEX friends_request_idx ON readers USING GIN (friends_request)")
				tx = tx.Exec("CREATE INDEX reader_blocked_idx ON readers USING GIN (reader_blocked)")
				return tx.Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx = tx.Model(&Reader{}).DropColumn("friends_pending")
				tx = tx.Model(&Reader{}).DropColumn("friends_request")
				tx = tx.Model(&Reader{}).DropColumn("reader_block")
				tx = tx.Exec("DROP INDEX friends_pending_idx")
				tx = tx.Exec("DROP INDEX friends_request_idx")
				tx = tx.Exec("DROP INDEX reader_blocked_idx")
				return tx.Error
			},
		},
		{
			ID: "Add forgot password field",
			Migrate: func(tx *gorm.DB) error {
				type Reader struct {
					ID             uint       `gorm:"primary_key" json:"-"`
					CreatedAt      time.Time  `json:"-"`
					UpdatedAt      time.Time  `json:"-"`
					DeletedAt      *time.Time `sql:"index" json:"-"`
					Name           string
					Pronouns       postgres.Jsonb
					ProfileColour  string
					Library        pq.StringArray `gorm:"type:varchar(64)[]"`
					ToRead         pq.StringArray `gorm:"type:varchar(64)[]"`
					Liked          pq.StringArray `gorm:"type:varchar(64)[]"`
					Read           pq.StringArray `gorm:"type:varchar(64)[]"`
					Friends        pq.Int64Array  `gorm:"type:integer[]"`
					FriendsPending pq.Int64Array  `gorm:"type:integer[]"`
					FriendsRequest pq.Int64Array  `gorm:"type:integer[]"`
					ReaderBlocked  pq.Int64Array  `gorm:"type:integer[]"`
					PasswordHash   string
					EmailHash      int64
					Plural         bool
					FavouriteBook  string
					Books          []Book //Book added by reader
					ForgotPassword ForgotPassword
				}

				type ForgotPassword struct {
					gorm.Model
					ReaderID  uint
					RandomKey string
				}

				return tx.CreateTable(&ForgotPassword{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable(&ForgotPassword{}).Error
			},
		},
	})
	return m.Migrate()
}

func Migrate() {
	db := bdb.Connect()
	fmt.Println()
	Start(db)
	db.Close()
}
