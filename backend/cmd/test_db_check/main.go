package main

import (
	"fmt"
	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
)

func main() {
	logger.Init()

	cfg := config.Load()
	if err := db.Init(cfg); err != nil {
		panic(err)
	}

	var user models.User
	// Find the user with UID 20000001
	if err := db.GetDB().Where("uid = ?", 20000001).First(&user).Error; err != nil {
		fmt.Printf("Error finding user: %v\n", err)
		return
	}

	fmt.Printf("DB User:\nID (Snowflake): %d\nUID (Sequence): %d\nUsername: %s\n",
		user.ID, user.UID, user.Username)

	// Verify ID is Snowflake (large) and UID is Sequence (small)
	if user.ID < 100000000 {
		fmt.Printf("FAIL: ID is too small for Snowflake: %d\n", user.ID)
	} else {
		fmt.Printf("PASS: ID looks like Snowflake\n")
	}

	if user.UID != 20000001 {
		fmt.Printf("FAIL: UID mismatch: expected 20000001, got %d\n", user.UID)
	} else {
		fmt.Printf("PASS: UID matches sequence\n")
	}
}
