package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"social_app/internal/proto/account"

	"google.golang.org/protobuf/proto"
)

func main() {
	username := fmt.Sprintf("testuser_%d", time.Now().Unix())
	password := "password123"

	// RegisterRequest
	req := &account.RegisterRequest{
		Username: username,
		Email:    username + "@example.com",
		Password: password,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	// Send POST request
	resp, err := http.Post("http://localhost:8080/api/v1/auth/register", "application/x-protobuf", bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Status: %s\nBody: %s\n", resp.Status, string(body))
		return
	}

	// RegisterResponse
	var authResp account.RegisterResponse
	if err := proto.Unmarshal(body, &authResp); err != nil {
		panic(err)
	}

	fmt.Printf("Registered User:\nID (Proto): %d\nUsername: %s\nToken: %s\n",
		authResp.User.Id, authResp.User.Username, authResp.Token)
}
