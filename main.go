package main

import (
	"encoding/csv"
	"fmt"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"gopkg.in/auth0.v5"
	"gopkg.in/auth0.v5/management"
	"os"
	"time"
)

type RewardfulFriend struct {
	Id     string
	Email  string
	Tokens string
}

type Result struct {
	Status  string
	Time    string
	Message string
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		color.Red(".env file is missing!")
		os.Exit(0)
	}
}

func main() {
	if len(os.Args) == 1 {
		color.Red("Expected path to CSV file! Aborting...")
		os.Exit(0)
	}

	csvContent := ReadCsv(os.Args[1])
	rwdFriends := createRewardfulFriendsList(csvContent)

	auth0domain := os.Getenv("AUTH0_DOMAIN")
	auth0client := os.Getenv("AUTH0_CLIENT_ID")
	auth0secret := os.Getenv("AUTH0_CLIENT_SECRET")

	m, err := management.New(auth0domain, management.WithClientCredentials(auth0client, auth0secret))
	if err != nil {
		panic(err)
	}

	start := time.Now()
	results := make(chan Result)
	c := time.NewTicker(200 * time.Millisecond)
	for _, user := range rwdFriends {
		go getAuth0UserAndUpdateMetadata(c, m, user, results)
	}
	defer c.Stop()

	for range rwdFriends {
		fmt.Println(<-results)
	}

	fmt.Printf("Finished in %s", time.Since(start))
}

func getAuth0UserAndUpdateMetadata(c *time.Ticker, m *management.Management, user RewardfulFriend, results chan Result) {
	<-c.C

	res, err := m.User.ListByEmail(user.Email)
	if err != nil {
		results <- Result{
			time.Now().Format(time.RFC1123Z),
			"- ERROR -",
			fmt.Sprintf("Error: %s when requesting user %s", err, user.Email),
		}
		return
	}

	if len(res) == 0 {
		results <- Result{
			time.Now().Format(time.RFC1123Z),
			"- NOT FOUND -",
			fmt.Sprintf("User %s not found", user.Email),
		}
		return
	}

	for _, data := range res {
		uu := &management.User{
			UserMetadata: map[string]interface{}{
				"user_id":         data.UserMetadata["user_id"],
				"rewardful_id":    user.Id,
				"rewardful_token": user.Tokens,
			},
		}
		err := m.User.Update(auth0.StringValue(data.ID), uu)
		if err != nil {
			results <- Result{
				time.Now().Format(time.RFC1123Z),
				"- ERROR -",
				fmt.Sprintf("Error: %s when requesting user %s", err, user.Email),
			}
			return
		}

		results <- Result{
			time.Now().Format(time.RFC1123Z),
			"- OK -",
			fmt.Sprintf("User updated: %s", user.Email),
		}
	}
}

func ReadCsv(path string) [][]string {
	file, err := os.Open(path)

	if err != nil {
		panic(err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	reader := csv.NewReader(file)
	reader.Comma = ','

	data, err := reader.ReadAll()

	if err != nil {
		panic(err)
	}

	return data
}

func createRewardfulFriendsList(data [][]string) []RewardfulFriend {
	var rewardfulFriendsList []RewardfulFriend
	for i, line := range data {
		if i > 0 { // omit header line
			var rec RewardfulFriend
			for j, field := range line {
				switch j {
				case 0:
					rec.Id = field
				case 4:
					rec.Email = field
				case 11:
					rec.Tokens = field
				}
			}
			rewardfulFriendsList = append(rewardfulFriendsList, rec)
		}
	}
	return rewardfulFriendsList
}
