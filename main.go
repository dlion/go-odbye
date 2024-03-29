package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/fatih/color"
	"github.com/kirsle/configdir"
	_ "github.com/mattn/go-sqlite3"
)

type fUser struct {
	id        int64
	username  string
	firstSeen *time.Time
}

type Conf struct {
	Nick           string `json:"nick,omitempty"`
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	AccessToken    string `json:"accessToken"`
	AccessSecret   string `json:"accessSecret"`
}

func main() {
	configPath := configdir.LocalConfig("go-odbye")
	err := configdir.MakePath(configPath) // Ensure it exists.
	if err != nil {
		log.Fatal(err)
	}

	//Config
	file, err := os.ReadFile(filepath.Join(configPath, "goodbye.json"))
	if err != nil {
		log.Fatal(err)
	}

	var configuration Conf
	err = json.Unmarshal(file, &configuration)
	if err != nil {
		log.Fatal(err)
	}
	client := getTwitterClient(configuration)

	defaultNick := "dlion92"
	if configuration.Nick != "" {
		defaultNick = configuration.Nick
	}

	nick := flag.String("nick", defaultNick, "your nickname on twitter")
	url := flag.Bool("url", true, "true if you want to see the url in output")

	flag.Parse()

	db, err := sql.Open("sqlite3", filepath.Join(configPath, "goo.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//Create a new db if not exists
	newDB := `
		CREATE TABLE IF NOT EXISTS users (id integer not null primary key, idUser integer, username text, firstSeen integer);
		CREATE UNIQUE INDEX IF NOT EXISTS "idUser_idx" ON "users" ("idUser");

		CREATE TABLE IF NOT EXISTS usersTmp (id integer not null primary key, idUser integer, username text, firstSeen integer);
		CREATE UNIQUE INDEX IF NOT EXISTS "idUser_idx_tmp" ON "usersTmp" ("idUser");
		`
	_, err = db.Exec(newDB)
	if err != nil {
		log.Fatal(err)
	}

	//Delete usersTmp table
	_, err = db.Exec(`DELETE FROM usersTmp`)
	if err != nil {
		log.Fatal(err)
	}

	//Begin
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	insertUsers, err := tx.Prepare("INSERT INTO usersTmp (idUser, username, firstSeen) VALUES(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer insertUsers.Close()

	color.Set(color.FgYellow)
	fmt.Println("Getting followers, please wait...")
	color.Unset()

	//Count followers
	plus := 0
	//Cursor
	var cursor int64 = -1
	// Get Followers
	for cursor != 0 {
		//dlion92 followers for now
		io := &twitter.FollowerListParams{ScreenName: *nick, Cursor: int64(cursor), Count: 200}
		followers, resp, err := client.Followers.List(io)
		if err != nil || resp.StatusCode != 200 {
			if resp.StatusCode == 429 { //
				log.Fatal("Too much requests in a short period, try again after some minutes (15 minutes should be fine)")
			} else {
				log.Fatal(err)
			}
		}
		//Put usernames in the db
		for _, v := range followers.Users {
			_, err = insertUsers.Exec(v.ID, v.ScreenName, time.Now().Unix())
			if err != nil {
				log.Fatal(err)
			}
			plus++
		}
		//Next Follower
		cursor = followers.NextCursor
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	followers, err := getNewFollowers(db)
	if err != nil {
		log.Fatal(err)
	}

	unfollowers, err := getUnfollowers(db)
	if err != nil {
		log.Fatal(err)
	}

	tx, err = db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT OR IGNORE INTO users (idUser, username, firstSeen) SELECT idUser, username, firstSeen FROM usersTmp")
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec("DELETE FROM users WHERE idUser NOT IN (SELECT idUser FROM usersTmp)")
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	db.Close()

	color.Set(color.FgHiBlue)
	fmt.Printf("%s has %d followers currently\n", *nick, plus)
	color.Unset()

	if len(followers) > 0 {
		color.Set(color.FgHiCyan)
		fmt.Printf("%s has %d new followers! :)\n", *nick, len(followers))
		color.Unset()
		color.Set(color.FgGreen)
		for i := range followers {
			if *url {
				fmt.Printf("https://twitter.com/")
			}

			fmt.Printf("%s welcome!\n", followers[i].username)
		}
		color.Unset()
	}

	if len(unfollowers) > 0 {
		color.Set(color.FgHiCyan)
		fmt.Printf("%s has %d new unfollowers! :(\n", *nick, len(unfollowers))
		color.Unset()
		color.Set(color.FgHiRed)
		for _, f := range unfollowers {
			if *url {
				fmt.Printf("https://twitter.com/")
			}
			fmt.Printf("%s goodbye!", f.username)
			if f.firstSeen != nil {
				fmt.Printf(" Followed since %s", f.firstSeen.Format(time.RFC822))
			}
			fmt.Println()
		}
		color.Unset()
	}

	if len(followers) == 0 && len(unfollowers) == 0 {
		color.Set(color.FgMagenta, color.Underline)
		fmt.Printf("%s has no new followers or unfollowers, bye!\n", *nick)
		color.Unset()
	}
}

func getUnfollowers(db *sql.DB) ([]fUser, error) {
	//Check new unfollowers
	var unfollowers []fUser
	rows, err := db.Query("SELECT idUser, username, firstSeen FROM users WHERE idUser NOT IN (SELECT idUser FROM usersTmp)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t *int64
		u := fUser{}
		err = rows.Scan(&u.id, &u.username, &t)
		if err != nil {
			return nil, err
		}

		if t != nil {
			x := time.Unix(*t, 0)
			u.firstSeen = &x
		}

		unfollowers = append(unfollowers, u)
	}
	return unfollowers, err
}

func getNewFollowers(db *sql.DB) ([]fUser, error) {
	//Check new followers
	var followers []fUser
	rows, err := db.Query("SELECT idUser, username, firstSeen FROM usersTmp WHERE idUser NOT IN (SELECT idUser FROM users)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t *int64
		u := fUser{}
		err = rows.Scan(&u.id, &u.username, &t)
		if err != nil {
			return nil, err
		}

		if t != nil {
			x := time.Unix(*t, 0)
			u.firstSeen = &x
		}

		followers = append(followers, u)
	}
	return followers, err
}

func getTwitterClient(configuration Conf) *twitter.Client {
	config := oauth1.NewConfig(configuration.ConsumerKey, configuration.ConsumerSecret)
	token := oauth1.NewToken(configuration.AccessToken, configuration.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	//Client
	return twitter.NewClient(httpClient)
}
