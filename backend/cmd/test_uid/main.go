package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	accountpb "social_app/internal/proto/account"
	"time"

	"google.golang.org/protobuf/proto"
)

func main() {
	username := fmt.Sprintf("testuid_%d", time.Now().Unix())
	email := fmt.Sprintf("%s@example.com", username)
	password := "password123"

	req := &accountpb.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post("http://localhost:8080/api/v1/auth/register", "application/x-protobuf", bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var registerResp accountpb.RegisterResponse
	if err := proto.Unmarshal(body, &registerResp); err != nil {
		panic(err)
	}

	fmt.Printf("Registered User: %+v\n", registerResp.User)
	fmt.Printf("Token: %s\n", registerResp.Token)
}
