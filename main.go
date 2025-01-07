package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jamistoso/gator/internal/config"
	"github.com/jamistoso/gator/internal/database"
	_ "github.com/lib/pq"
)

type command struct{
	name			string
	arguments		[]string
}

type state struct{
	db  *database.Queries
	cfg *config.Config
}

type commands struct{
	funcMap			map[string]func(*state, command) error
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

const dbURL = "postgres://postgres:postgres@localhost:5432/gator"
const dateFormatString = "Mon, 02 Jan 2006 15:04:05 -0700"

func main() {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(fmt.Errorf("error opening postgres database: %s", err))
		os.Exit(1)
	}
	dbQueries := database.New(db)

	cfg := config.Read()
	mainState := &state{
		cfg: 	&cfg,
		db:		dbQueries,
	}
	cmdMap := commands{
		funcMap: map[string]func(*state, command) error{},
	}
	cmdMap.register("login", handlerLogin)
	cmdMap.register("register", handlerRegister)
	cmdMap.register("reset", handlerReset)
	cmdMap.register("users", handlerUsers)
	cmdMap.register("agg", handlerAgg)
	cmdMap.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmdMap.register("feeds", handlerFeeds)
	cmdMap.register("follow", middlewareLoggedIn(handlerFollow))
	cmdMap.register("following", middlewareLoggedIn(handlerFollowing))
	cmdMap.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmdMap.register("browse", middlewareLoggedIn(handlerBrowse))
	args := os.Args
	if len(args) < 2 {
		fmt.Println(fmt.Errorf("command name required"))
		os.Exit(1)
	}
	mainCmd := command{
		name:		args[1],
		arguments:	args[2:],
	}
	err = cmdMap.run(mainState, mainCmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("login command requires a username argument")
	}
	name := cmd.arguments[0]

	_, err := s.db.GetUser(context.Background(), name)
	if err != nil {
		return err
	}

	if err := s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return err
	}
	fmt.Printf("User has been set to %s\n", cmd.arguments[0])
	return nil
}


func handlerUsers(s *state, cmd command) error {
	
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		outStr := user.Name
		if s.cfg.Current_user_name == user.Name {
			outStr += " (current)"
		}
		fmt.Println(outStr)
	}

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("login command requires a username argument")
	}
	name := cmd.arguments[0]

	// arg list: id, created_at, updated_at, name
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:			uuid.New(),
		CreatedAt: 	time.Now(), 
		UpdatedAt:  time.Now(), 	
		Name:		name,
	})
	if err != nil {
		return err
	}

	if err := s.cfg.SetUser(cmd.arguments[0]); err != nil {
		return err
	}
	fmt.Printf("User has been created: %s\n", user)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if err := s.db.Reset(context.Background()); err != nil {
		return err
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return fmt.Errorf("agg command requires a \"time_between_reqs\" argument")
	}
	timeBetweenReqs := cmd.arguments[0]
	timeDuration, err := time.ParseDuration(timeBetweenReqs)
	if err != nil {
		return err
	}
	fmt.Printf("Collecting feeds every %s\n", timeBetweenReqs)
	ticker := time.NewTicker(timeDuration)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 2 {
		return fmt.Errorf("addfeed command requires 2 arguments")
	}

	feedName := cmd.arguments[0]
	url := cmd.arguments[1]

	// arg list: id, created_at, updated_at, (feed)name, url, user_id
	params := database.CreateFeedParams{
		ID:			uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:	time.Now(),
		Name:		feedName,
		Url:		url,
		UserID:		uuid.NullUUID{
			UUID: 	currentUser.ID,
			Valid:	true,
		},
	}
	feed, err := s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return err
	}

	followParams := database.CreateFeedFollowParams{
		ID:			uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:	time.Now(),
		UserID: 	uuid.NullUUID{
			UUID: 	currentUser.ID,
			Valid: 	true,
		},
		FeedID: 	uuid.NullUUID{
			UUID: 	feed.ID,
			Valid: 	true,
		},
	}
	_, err = s.db.CreateFeedFollow(context.Background(), followParams)
	
	if err != nil {
		return err
	}

	fmt.Printf("Feed follow created for url \"%s\" by user \"%s\"\n", 
				feed.Url, currentUser.Name)


	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		 return err
	}

	for _, feed := range feeds {
		user, err := s.db.GetUserFromID(context.Background(), feed.UserID.UUID)
		if err != nil {
			return err
		}
		fmt.Printf("Name: %s | Url: %s | User: %s\n", 
				feed.Name, feed.Url, user.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 1 {
		return fmt.Errorf("follow command requires url argument")
	}
	url := cmd.arguments[0]
	
	feed, err := s.db.GetFeedFromURL(context.Background(), url)
	if err != nil {
		fmt.Println("feed not found")
		return err
	}

	params := database.CreateFeedFollowParams{
		ID:			uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:	time.Now(),
		UserID: 	uuid.NullUUID{
			UUID: 	currentUser.ID,
			Valid: 	true,
		},
		FeedID: 	uuid.NullUUID{
			UUID: 	feed.ID,
			Valid: 	true,
		},
	}
	_, err = s.db.CreateFeedFollow(context.Background(), params)
	
	if err != nil {
		return err
	}

	fmt.Printf("Feed follow created for url \"%s\" by user \"%s\"\n", 
				feed.Url, currentUser.Name)

	return nil
}

func handlerFollowing(s *state, cmd command, currentUser database.User) error {

	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), uuid.NullUUID{UUID: currentUser.ID, Valid: true})
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Println(feed.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 1 {
		return fmt.Errorf("unfollow command requires a \"url\" argument")
	}

	params := database.DeleteFeedFollowForUserParams{
		UserID:	uuid.NullUUID{
			UUID:	currentUser.ID,
			Valid:	true,
		},
		Url: 	cmd.arguments[0],	
	}
	err := s.db.DeleteFeedFollowForUser(context.Background(), params)
	if err != nil {
		return err
	}

	return nil
}

func handlerBrowse(s *state, cmd command, currentUser database.User) error {
	// default to 2 posts
	numPostsToGet := 2
	if len(cmd.arguments) >= 1 {
		var err error
		numPostsToGet, err = strconv.Atoi(cmd.arguments[0])
		if err != nil {
			return err
		}
	}
	userPostsParams := database.GetPostsForUserParams{
		UserID:		uuid.NullUUID{
			UUID:	currentUser.ID,
			Valid:	true,
		},
		Limit:		int32(numPostsToGet),
	}
	posts, err := s.db.GetPostsForUser(context.Background(), userPostsParams)
	if err != nil {
		return err
	}
	for _, post := range posts {
		fmt.Println(html.EscapeString(post.Title.String))
		fmt.Println(post.Url)
		fmt.Println(post.PublishedAt)
		fmt.Println(html.EscapeString(post.Description.String))
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.funcMap[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	valCmd, ok := c.funcMap[cmd.name]
	if !ok {
		return fmt.Errorf("command not found: %s", cmd.name)
	}

	err := valCmd(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, err
	}
	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RSSFeed{}, err
	}

	var rssFeed *RSSFeed
	err = xml.Unmarshal(data, &rssFeed)
	if err != nil {
		return &RSSFeed{}, err
	}

	return rssFeed, nil
}

func scrapeFeeds(s *state) error {
	dbFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}
	err = s.db.MarkFeedFetched(context.Background(), dbFeed.ID)
	if err != nil {
		return err
	}
	feed, err := fetchFeed(context.Background(), dbFeed.Url)
	if err != nil {
		return err
	}
	
	for _, item := range feed.Channel.Item {
		item.Title = html.EscapeString(item.Title)
		item.Description = html.EscapeString(item.Description)
		pubDateTime, err := time.Parse(dateFormatString, item.PubDate)
		if err != nil {
			fmt.Println(err)
			continue
		}
		postParams := database.CreatePostParams{
			ID:				uuid.New(),
			CreatedAt:  	time.Now(),
			UpdatedAt:		time.Now(),
			Title:			sql.NullString{
				String:		item.Title,
				Valid:		true,
			},
			Url:			item.Link,
			Description:	sql.NullString{
				String:		item.Description,
				Valid:		true,
			},
			PublishedAt: 	sql.NullTime{
				Time:	pubDateTime,
				Valid:	true,
			},
			FeedID: 		uuid.NullUUID{
				UUID:	dbFeed.ID,
				Valid:	true,
			},	
		}
		_, err = s.db.CreatePost(context.Background(), postParams)
		if err != nil {
			fmt.Println(err)
		}
		
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error{
	outFunction := func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
	return outFunction
}