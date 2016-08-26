package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os/user"
)

type fUser struct {
	id       int64
	username string
}

func main() {
	//User infos
	user, err := user.Current()
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	//Config
	config := oauth1.NewConfig("<CONSUMER KEY>", "<CONSUMER SECRET>")
	token := oauth1.NewToken("<ACCESS TOKEN>", "<ACCESS SECRET>")
	httpClient := config.Client(oauth1.NoContext, token)
	//Client
	client := twitter.NewClient(httpClient)
	//Cursor
	var cursor int64
	cursor = -1

	nick := flag.String("nick", "dlion92", "your nickname on twitter")
	url := flag.Bool("url", false, "true if you want to see the url in output")

	flag.Parse()

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/.goo.db", user.HomeDir))
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	defer db.Close()

	//Create a new db if not exists
	newDB := `
		CREATE TABLE IF NOT EXISTS users (id integer not null primary key, username text);
		CREATE TABLE IF NOT EXISTS usersTmp (id integer not null primary key, username text);
		`
	_, err = db.Exec(newDB)
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}

	//Delete usersTmp table
	_, err = db.Exec(`DELETE FROM usersTmp`)
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	//Count followers
	var plus int64
	plus = 0
	//Begin
	tx, err := db.Begin()
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}

	insertUsers, err := tx.Prepare("INSERT INTO usersTmp (id, username) VALUES(?, ?)")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	defer insertUsers.Close()

	color.Set(color.FgYellow)
	fmt.Println("Getting followers, please wait...")
	color.Unset()
	// Get Followers
	for cursor != 0 {
		//dlion92 followers for now
		io := &twitter.FollowerListParams{ScreenName: *nick, Cursor: int(cursor), Count: 200}
		followers, resp, err := client.Followers.List(io)
		if err != nil || resp.StatusCode != 200 {
			color.Set(color.FgRed, color.BlinkSlow)
			if resp.StatusCode == 429 { //
				log.Fatal("Too much requests in a short period, try again after some minutes (15 minutes should be fine)")
			} else {
				log.Fatal(err)
			}
			color.Unset()

		}
		//Put usernames in the db
		for _, v := range followers.Users {
			_, err = insertUsers.Exec(v.ID, fmt.Sprintf("%s", v.ScreenName))
			if err != nil {
				color.Set(color.FgRed, color.BlinkSlow)
				log.Fatal(err)
				color.Unset()
			}
			plus++
		}
		//Next Follower
		cursor = followers.NextCursor
	}
	tx.Commit()

	//Check new followers
	var followers []fUser
	rows, err := db.Query("SELECT id, username FROM usersTmp WHERE id NOT IN (SELECT id FROM users)")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var id int64
		err = rows.Scan(&id, &username)
		if err != nil {
			color.Set(color.FgRed, color.BlinkSlow)
			log.Fatal(err)
			color.Unset()
		}
		u := fUser{
			id:       id,
			username: username,
		}
		followers = append(followers, u)
	}

	//Check new unfollowers
	var unfollowers []fUser
	rows, err = db.Query("SELECT id, username FROM users WHERE id NOT IN (SELECT id FROM usersTmp)")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var id int64
		err = rows.Scan(&id, &username)
		if err != nil {
			color.Set(color.FgRed, color.BlinkSlow)
			log.Fatal(err)
			color.Unset()
		}
		u := fUser{
			id:       id,
			username: username,
		}
		unfollowers = append(unfollowers, u)
	}

	//Delete all users to update the table
	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	//Get all new followers
	rowsN, err := db.Query("SELECT id, username FROM usersTmp")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}
	defer rowsN.Close()

	tx, err = db.Begin()
	if err != nil {
		panic(err)
	}
	insertUsers, err = tx.Prepare("INSERT INTO users (id, username) VALUES(?, ?)")
	if err != nil {
		color.Set(color.FgRed, color.BlinkSlow)
		log.Fatal(err)
		color.Unset()
	}

	for rowsN.Next() {
		var username string
		var id int64
		err = rowsN.Scan(&id, &username)
		if err != nil {
			panic(err)
		}

		_, err = insertUsers.Exec(id, fmt.Sprintf("%s", username))
		if err != nil {
			color.Set(color.FgRed, color.BlinkSlow)
			log.Fatal(err)
			color.Unset()
		}
	}
	tx.Commit()

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
			if *url == true {
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
		for i := range unfollowers {
			if *url == true {
				fmt.Printf("https://twitter.com/")
			}
			fmt.Printf("%s goodbye!\n", unfollowers[i].username)
		}
		color.Unset()
	}

	if len(followers) == 0 && len(unfollowers) == 0 {
		color.Set(color.FgMagenta, color.Underline)
		fmt.Printf("%s has no new followers or unfollowers, bye!\n", *nick)
		color.Unset()
	}
}
