package main

import (
	"fmt"
	"social_app/internal/db"
	"social_app/internal/config"
)

func main() {
    cfg := config.Load()
	db.Init(cfg)

	var uid uint64
	if err := db.GetDB().Raw("SELECT last_value FROM user_uid_seq").Scan(&uid).Error; err != nil {
		fmt.Printf("Error getting sequence: %v\n", err)
	} else {
		fmt.Printf("Current Sequence Value: %d\n", uid)
	}
    
    var count int64
    db.GetDB().Table("users").Count(&count)
    fmt.Printf("User Count: %d\n", count)

    // Check last user
    var user struct {
        ID uint64
        UID uint64
    }
    db.GetDB().Table("users").Order("created_at desc").First(&user)
    fmt.Printf("Last User: ID=%d, UID=%d\n", user.ID, user.UID)
}
