// fbgdl is a Facebook Graph downloader. It cycles through as many users
// as it is told (or MaxUint64) and stores them in a database.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
)

const dbFile = "fbgraph.db"
const graphBase = "https://graph.facebook.com"

// userUrl takes a user ID and returns the Facebook graph URL for that user.
func userUrl(uid uint64) string {
	return fmt.Sprintf("%s/%d", graphBase, uid)
}

// Type GraphUser represents an entry from the Graph. It is not suitable
// for storing, but contains the data to be converted to a User type
// that can be stored in the database.
type GraphUser struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	First    string `json:"first_name"`
	Last     string `json:"last_name"`
	Link     string `json:"link"`
	Username string `json:"username"`
	Gender   string `json:"gender"`
	Locale   string `json:"locale"`
	Error    struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// Failed returns true if the UID was an invalid Graph user.
func (gu *GraphUser) Failed() bool {
	if gu.Error.Message != "" {
		return true
	}
	return false
}

// ToUser converts a GraphUser to a User.
func (gu *GraphUser) ToUser() (u *User, err error) {
	if gu.Failed() {
		err = fmt.Errorf(gu.Error.Message)
		return
	}
	u = new(User)

	n, err := strconv.ParseUint(gu.Id, 10, 64)
	if err != nil {
		return
	}

	nString := fmt.Sprintf("%d", n)
	if nString != gu.Id {
		err = fmt.Errorf("invalid id conversion")
		return
	}

	u.Id = n
	u.Name = gu.Name
	u.First = gu.First
	u.Last = gu.Last
	u.Link = gu.Link
	u.Username = gu.Username
	u.Gender = gu.Gender
	u.Locale = gu.Locale
	return
}

// Type User is a representation of a graph user suitable for storing
// in the database.
type User struct {
	Id       uint64
	Name     string
	First    string
	Last     string
	Link     string
	Username string
	Gender   string
	Locale   string
}

// Method Store is used to save a user to the database.
func (u *User) Store() (err error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(`insert into users values (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Id, u.Name, u.First, u.Last, u.Link, u.Username, u.Gender,
		u.Locale)
	return
}

// checkDatabase looks for the database file, and makes sure it has the
// appropriate table.
func checkDatabase() {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return
	}
	defer db.Close()

	var missingTable = fmt.Errorf("no such table: users")

	_, err = db.Exec("select count(*) from users")
	if err != nil && err.Error() == missingTable.Error() {
		fmt.Println("creating table")
		err = createDB()
	}
	if err != nil {
		panic("[!] fbgdl: opening profile database: " +
			err.Error())
	}
}

// createDB is responsible for creating the database.
func createDB() (err error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(`create table users
                          (id integer primary key unique not null,
                           name text,
                           first text,
                           last text,
                           link text,
                           username text,
                           gender text,
                           locale text)`)
	return
}

// fetchUser grabs a user from the Graph, storing the user in the database
// if it is a valid user. Otherwise, an error is returned.
func fetchUser(uid uint64) (u *User, err error) {
	resp, err := http.Get(userUrl(uid))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	gu := new(GraphUser)
	err = json.Unmarshal(body, &gu)
	if err != nil {
		return
	}

	u, err = gu.ToUser()
	if err != nil {
		return
	}

	err = u.Store()
	return
}

// Download the graph!
func main() {
	checkDatabase()

	fMaxUid := flag.Uint64("u", math.MaxUint64, "max uid to grab")
	flag.Parse()

	for uid := uint64(0); uid < *fMaxUid; uid++ {
		u, err := fetchUser(uid)
		if err != nil {
			logMsg := fmt.Sprintf("failed uid %d: %s", uid,
				err.Error())
			log.Println(logMsg)
		} else {
			logMsg := fmt.Sprintf("stored uid %d (%s)", uid,
				u.Username)
			log.Println(logMsg)
		}
	}
}
