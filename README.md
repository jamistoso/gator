Requirements:
* Go
* Postgres

Gator CLI Installation:
Run the following command: ```go install github.com/jamistoso/gator```

Gator Config File Setup:
1. In your home directory, create a file named ".gatorconfig.json".
2. Determine your postgres connection string. It should be within the following layout: ```protocol://username:password@host:port/database```
3. Within ~/.gatorconfig.json, add the following values: 
```
{"Db_url":"CONNECTION_STRING",
"Current_user_name":"USERNAME"}
```

Running Gator:
1. In the command line, type the following: ```gator COMMAND```. 
"COMMAND" has the following options:  
* login: Logs in to the provided user account. ```Requires a username argument```
* register: Create a user with the provided user name and logs in to the user account. ```Requires a username argument```
* reset: Deletes all 
* users: Lists all user accounts
* addfeed: Adds a feed to watch and links it to the currently logged in user. ```Requires a "feed_name" and "url" argument```
* feeds: Lists all feeds in the database
* agg: Aggregate posts for all feeds linked to the currently logged in user
* follow: Follow a given feed by linking it to the current user. ```Requires a "url" argument```
* following: List the title of all feeds that the current user follows.
* unfollow: Unfollow a given feed by unlinking it from the current user. ```Requires a "url" argument``
* browse: Lists the most recent posts from the feeds that the currently logged in user follows. ```Takes an optional "numPosts" option, defaults to 2```

