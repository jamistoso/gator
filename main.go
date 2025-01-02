package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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

const dbURL = "postgres://postgres:postgres@localhost:5432/gator"

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

	// arg list: context, id, created_at, updated_at, name
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