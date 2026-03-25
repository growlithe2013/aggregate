package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/growlithe2013/aggregate/internal/config"
	"github.com/growlithe2013/aggregate/internal/database"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	cmdName string
	args    []string
}

type commands struct {
	cmd map[string]func(*state, command) error
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

func (c *commands) register(name string, handler func(*state, command) error) {
	c.cmd[name] = handler
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("no username error. Usage: login username")
	}
	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}
	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("Username " + cmd.args[0] + " has been set")
	return nil
}

func handlerReset(s *state, _ command) error {
	return s.db.ClearDB(context.Background())
}

func handlerRegister(s *state, cmd command) error {

	if len(cmd.args) == 0 {
		return errors.New("no username provided. Usage: register username")
	}
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	})
	if err != nil {
		return err
	}
	fmt.Println("User " + user.Name + " was created")
	err = s.cfg.SetUser(user.Name)
	return err
}

func handlerGetUsers(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, user := range users {
		fmt.Print(user)
		if user == s.cfg.Current_user_name {
			fmt.Print(" (current)")
		}
		fmt.Println()
	}
	return err
}

func handlerAgg(s *state, _ command) error {
	fmt.Println("Fetching feeds (refetch every minute)")
	for 1 == 1 {
		feeds, err := s.db.GetFeeds(context.Background())
		if err != nil {
			return err
		}
		err = s.db.ClearArticles(context.Background())
		if err != nil {
			return err
		}
		for _, feedRange := range feeds {
			feed, err := fetchFeed(context.Background(), feedRange.Url)
			if err != nil {
				return err
			}
			UUID, err := s.db.GetFeedID(context.Background(), feedRange.Url)
			for _, article := range feed.Channel.Item {
				if article.Description == "" {
					continue
				}
				pubbed, err := time.Parse(time.RFC1123Z, article.PubDate)
				if err != nil {
					return err
				}
				err = s.db.InsertArticle(context.Background(), database.InsertArticleParams{
					ID:            uuid.New(),
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
					Name:          article.Title,
					FeedUrl:       article.Link,
					FeedID:        UUID,
					PublishDate:   pubbed,
					LastFetchedAt: time.Now(),
					Description:   article.Description,
				})
				if err != nil {
					return err
				}
			}

			fmt.Println("fetched " + feedRange.Name)
		}
		fmt.Println("Refetching in 1 minute")
		time.Sleep(time.Minute)
		fmt.Println("rerunning")
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return errors.New("invalid usage. Usage: addfeed <name> <url>")
	}

	res, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	})
	if err != nil {
		return err
	}
	fmt.Println(res)
	feedId, err := s.db.GetFeedID(context.Background(), cmd.args[1])
	if err != nil {
		return err
	}
	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID:    feedId,
		UserID:    user.ID,
	})
	fmt.Println(follow)
	return err
}

func handlerFeedFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return errors.New("invalid usage. Usage: follow url")
	}
	feedId, err := s.db.GetFeedID(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	res, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID:    feedId,
		UserID:    user.ID,
	})
	if err == nil {
		fmt.Println(res)
	}
	return err
}

func handlerFeeds(s *state, _ command) error {
	res, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, feed := range res {
		user, err := s.db.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return err
		}
		fmt.Println(feed.Name)
		fmt.Println(feed.Url)
		fmt.Println(user.Name)
		fmt.Println("Added on: " + feed.CreatedAt.String())
	}
	return nil
}

func handlerFollows(s *state, _ command, user database.User) error {
	GetFeedFollows, err := s.db.GetFeedFollows(context.Background(), user.ID)
	if err != nil {
		return err
	}
	for _, feed := range GetFeedFollows {
		feedInfo, err := s.db.GetNameByID(context.Background(), feed)
		if err != nil {
			return err
		}
		fmt.Println(feedInfo)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return errors.New("invalid usage. Usage: follow url")
	}
	feedID, err := s.db.GetFeedID(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}
	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feedID,
	})
	return err
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	var list int
	var err error = nil
	if len(cmd.args) < 1 {
		list = 2
	} else {
		list, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
	}

	feedIDs, err := s.db.GetFeedIDsByUserID(context.Background(), user.ID)
	if err != nil {
		return err
	}
	posts := make([]database.Article, 0)
	for _, feedID := range feedIDs {
		postList, err := s.db.GetArticlesByFeedId(context.Background(), feedID)
		if err != nil {
			return err
		}
		for i, post := range postList {
			if i >= list {
				break
			}
			posts = append(posts, post)
		}
	}
	results := make([]database.Article, 0)
	index := 0
	if len(posts) == 0 {
		return nil
	}

	for len(results) < list {
		newest := posts[0]
		for i, post := range posts {
			if post.PublishDate.After(newest.PublishDate) {
				index = i
				newest = post
			}

		}
		results = append(results, newest)
		posts = slices.Delete(posts, index, index+1)
	}

	for _, post := range results {
		fmt.Println(post.Name)
		fmt.Println()
		fmt.Println(post.PublishDate.String())
		fmt.Println()
		fmt.Println(post.Description)
		fmt.Println()
		fmt.Println("----------")
	}

	return nil
}

func (c *commands) run(s *state, cmd command) error {

	handler, ok := c.cmd[cmd.cmdName]
	if ok {
		return handler(s, cmd)
	}

	return errors.New("command not found")
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	var feed RSSFeed

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return nil, err
	}
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i, item := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(item.Title)
		feed.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
	return &feed, nil
}

func main() {

	s := new(state)
	s.cfg = config.Read()
	db, err := sql.Open("postgres", s.cfg.Db_url)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	dbQueries := database.New(db)
	s.db = dbQueries
	cmds := commands{cmd: map[string]func(*state, command) error{}}
	if len(os.Args) < 2 {
		fmt.Println("Usage: aggregate <command>")
		os.Exit(1)
	}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerGetUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFeedFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollows))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))
	err = cmds.run(s, command{cmdName: os.Args[1], args: os.Args[2:]})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
