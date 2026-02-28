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

	var invalidUsers []models.User
	if err := db.GetDB().Where("uid > ?", 100000000).Find(&invalidUsers).Error; err != nil {
		fmt.Printf("Error finding invalid users: %v\n", err)
		return
	}

	fmt.Printf("Found %d users with invalid UIDs (likely Snowflake IDs)\n", len(invalidUsers))
	if len(invalidUsers) > 0 {
		for _, u := range invalidUsers {
			fmt.Printf("Deleting user: ID=%d, UID=%d\n", u.ID, u.UID)
			if err := db.GetDB().Unscoped().Delete(&u).Error; err != nil {
				fmt.Printf("Error deleting user %d: %v\n", u.ID, err)
			}
		}
	}

	var maxUID uint64
	if err := db.GetDB().Model(&models.User{}).Select("COALESCE(MAX(uid), 0)").Scan(&maxUID).Error; err != nil {
		fmt.Printf("Error getting max UID: %v\n", err)
		return
	}

	startVal := uint64(20000000)
	if maxUID > startVal {
		startVal = maxUID
	}

	fmt.Printf("Resetting user_uid_seq to %d\n", startVal)
	if err := db.GetDB().Exec(fmt.Sprintf("SELECT setval('user_uid_seq', %d)", startVal)).Error; err != nil {
		fmt.Printf("Error resetting sequence: %v\n", err)
		return
	}

	var currentSeq uint64
	if err := db.GetDB().Raw("SELECT last_value FROM user_uid_seq").Scan(&currentSeq).Error; err != nil {
		fmt.Printf("Error verifying sequence: %v\n", err)
	} else {
		fmt.Printf("Sequence reset successfully. Current value: %d\n", currentSeq)
	}
}
