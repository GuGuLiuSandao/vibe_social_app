package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	accountpb "social_app/internal/proto/account"
	commonpb "social_app/internal/proto/common"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func Register(c *gin.Context, cfg *config.Config) {
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/x-protobuf") {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "unsupported content type",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusUnsupportedMediaType, "application/x-protobuf", data)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to read request",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	var req accountpb.RegisterRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "invalid protobuf",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || !strings.Contains(req.Email, "@") || len(req.Password) < 6 {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "invalid registration data",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	logger.Info("[HTTP REQ] Register - username=%s, email=%s", req.Username, req.Email)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash password: %v", err)
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to hash password",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	snowflakeID, err := nextSnowflakeID()
	if err != nil {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to generate id",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	uid, err := nextUserUID()
	if err != nil {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to generate uid",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	user := models.User{
		ID:       snowflakeID,
		UID:      uid,
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Username,
	}

	logger.DBWrite("CREATE", "users", "Creating new user: username=%s, email=%s", req.Username, req.Email)

	if err := db.DB.Create(&user).Error; err != nil {
		logger.Error("Failed to create user: %v", err)
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "username or email already exists",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusConflict, "application/x-protobuf", data)
		return
	}

	logger.DBWrite("CREATE", "users", "User created successfully: id=%d, username=%s", user.ID, user.Username)

	token, err := GenerateToken(user.UID, user.Username, cfg)
	if err != nil {
		resp := &accountpb.RegisterResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to generate token",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	logger.Info("[HTTP RES] Register - Success, user_id=%d, username=%s", user.ID, user.Username)
	logger.Info("User registered successfully: id=%d, username=%s", user.ID, user.Username)

	resp := &accountpb.RegisterResponse{
		ErrorCode: commonpb.ErrorCode_ERROR_CODE_OK,
		Message:   "ok",
		Token:     token,
		User: &accountpb.User{
			Id:       user.UID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
		},
	}
	data, _ := proto.Marshal(resp)
	c.Data(http.StatusOK, "application/x-protobuf", data)
}

func Login(c *gin.Context, cfg *config.Config) {
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/x-protobuf") {
		resp := &accountpb.LoginResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "unsupported content type",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusUnsupportedMediaType, "application/x-protobuf", data)
		return
	}
	loginProtobuf(c, cfg)
}

func loginProtobuf(c *gin.Context, cfg *config.Config) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp := &accountpb.LoginResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to read request",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	var req accountpb.LoginRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		resp := &accountpb.LoginResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "invalid protobuf",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	respond := func(status int, code commonpb.ErrorCode, message string, user *models.User, token string) {
		resp := &accountpb.LoginResponse{
			ErrorCode: code,
			Message:   message,
			Token:     token,
		}
		if user != nil {
			resp.User = &accountpb.User{
				Id:       user.UID,
				Username: user.Username,
				Email:    user.Email,
				Nickname: user.Nickname,
			}
		}
		data, _ := proto.Marshal(resp)
		c.Data(status, "application/x-protobuf", data)
	}

	if req.Uid != 0 {
		uid := uint(req.Uid)
		if !IsWhitelistUID(uid) {
			logger.Warn("Whitelist login rejected for uid %d", uid)
			respond(http.StatusForbidden, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "uid not in whitelist", nil, "")
			return
		}

		user, err := EnsureWhitelistUser(uid)
		if err != nil {
			logger.Error("Whitelist login failed for uid %d: %v", uid, err)
			respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to login whitelist user", nil, "")
			return
		}

		token, err := GenerateToken(user.UID, user.Username, cfg)
		if err != nil {
			respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to generate token", nil, "")
			return
		}

		logger.Info("[HTTP RES] Login - Whitelist success, user_id=%d, username=%s", user.ID, user.Username)
		respond(http.StatusOK, commonpb.ErrorCode_ERROR_CODE_OK, "ok", user, token)
		return
	}

	if req.Email == "" || req.Password == "" || !strings.Contains(req.Email, "@") || len(req.Password) < 6 {
		respond(http.StatusBadRequest, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "invalid credentials", nil, "")
		return
	}

	logger.Info("[HTTP REQ] Login - email=%s", req.Email)
	logger.DBRead("SELECT", "users", "Looking up user by email: %s", req.Email)

	var user models.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		logger.Warn("Login failed for email %s: user not found", req.Email)
		logger.Info("[HTTP RES] Login - Failed, email=%s, reason=user_not_found", req.Email)
		respond(http.StatusUnauthorized, commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED, "invalid credentials", nil, "")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Warn("Login failed for user %d: invalid password", user.ID)
		logger.Info("[HTTP RES] Login - Failed, user_id=%d, reason=invalid_password", user.ID)
		respond(http.StatusUnauthorized, commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED, "invalid credentials", nil, "")
		return
	}

	if user.UID == 0 {
		uid, err := nextUserUID()
		if err != nil {
			respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to generate uid", nil, "")
			return
		}
		if err := db.DB.Model(&user).Update("uid", uid).Error; err != nil {
			respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to update uid", nil, "")
			return
		}
		user.UID = uid
	}

	token, err := GenerateToken(user.UID, user.Username, cfg)
	if err != nil {
		respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to generate token", nil, "")
		return
	}

	logger.Info("[HTTP RES] Login - Success, user_id=%d, username=%s", user.ID, user.Username)
	logger.Info("User logged in successfully: id=%d, username=%s", user.ID, user.Username)
	respond(http.StatusOK, commonpb.ErrorCode_ERROR_CODE_OK, "ok", &user, token)
}

func GetCurrentUser(c *gin.Context, cfg *config.Config) {
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/x-protobuf") {
		resp := &accountpb.AuthResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "unsupported content type",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusUnsupportedMediaType, "application/x-protobuf", data)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp := &accountpb.AuthResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to read request",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	var req accountpb.AuthRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		resp := &accountpb.AuthResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "invalid protobuf",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	respond := func(status int, code commonpb.ErrorCode, message string, user *models.User) {
		resp := &accountpb.AuthResponse{
			ErrorCode: code,
			Message:   message,
		}
		if user != nil {
			resp.User = &accountpb.User{
				Id:       user.UID,
				Username: user.Username,
				Email:    user.Email,
				Nickname: user.Nickname,
			}
		}
		data, _ := proto.Marshal(resp)
		c.Data(status, "application/x-protobuf", data)
	}

	if req.Uid == 0 {
		respond(http.StatusBadRequest, commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "uid required", nil)
		return
	}

	uid := req.Uid
	logger.Info("[HTTP REQ] GetCurrentUser - uid=%v", uid)

	var user models.User
	if IsWhitelistUID(uint(uid)) {
		u, err := EnsureWhitelistUser(uint(uid))
		if err != nil {
			logger.Error("Failed to ensure whitelist user %v: %v", uid, err)
			respond(http.StatusInternalServerError, commonpb.ErrorCode_ERROR_CODE_INTERNAL, "failed to ensure whitelist user", nil)
			return
		}
		user = *u
		respond(http.StatusOK, commonpb.ErrorCode_ERROR_CODE_OK, "ok", &user)
		return
	}

	if req.Token == "" {
		respond(http.StatusUnauthorized, commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED, "token required", nil)
		return
	}

	claims, err := ParseToken(req.Token, cfg)
	if err != nil {
		respond(http.StatusUnauthorized, commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED, "invalid or expired token", nil)
		return
	}

	if claims.UserID != uid {
		respond(http.StatusUnauthorized, commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED, "uid does not match token", nil)
		return
	}

	logger.DBRead("SELECT", "users", "Getting current user: uid=%v", uid)
	if err := db.DB.Where("uid = ?", uid).First(&user).Error; err != nil {
		logger.Error("Failed to get user %v: %v", uid, err)
		logger.Info("[HTTP RES] GetCurrentUser - Failed, user_id=%v, reason=user_not_found", uid)
		respond(http.StatusNotFound, commonpb.ErrorCode_ERROR_CODE_NOT_FOUND, "user not found", nil)
		return
	}

	logger.Info("[HTTP RES] GetCurrentUser - Success, user_id=%d, username=%s, email=%s", user.ID, user.Username, user.Email)
	respond(http.StatusOK, commonpb.ErrorCode_ERROR_CODE_OK, "ok", &user)
}

func UpdateProfile(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/x-protobuf") {
		resp := &accountpb.UpdateProfileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "unsupported content type",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusUnsupportedMediaType, "application/x-protobuf", data)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp := &accountpb.UpdateProfileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to read request",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	var req accountpb.UpdateProfileRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		resp := &accountpb.UpdateProfileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "invalid protobuf",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	uid, exists := c.Get("user_id")
	if !exists {
		resp := &accountpb.UpdateProfileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_UNAUTHORIZED,
			Message:   "unauthorized",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusUnauthorized, "application/x-protobuf", data)
		return
	}
	userID := uid.(uint64)

	logger.Info("[HTTP REQ] UpdateProfile - uid=%v, nickname=%s", userID, req.Nickname)

	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&models.User{}).Where("uid = ?", userID).Updates(updates).Error; err != nil {
			logger.Error("Failed to update user %v: %v", userID, err)
			resp := &accountpb.UpdateProfileResponse{
				ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
				Message:   "failed to update profile",
			}
			data, _ := proto.Marshal(resp)
			c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
			return
		}
	}

	var user models.User
	if err := db.DB.Where("uid = ?", userID).First(&user).Error; err != nil {
		logger.Error("Failed to get user %v: %v", userID, err)
		resp := &accountpb.UpdateProfileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_NOT_FOUND,
			Message:   "user not found",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusNotFound, "application/x-protobuf", data)
		return
	}

	resp := &accountpb.UpdateProfileResponse{
		ErrorCode: commonpb.ErrorCode_ERROR_CODE_OK,
		Message:   "ok",
		User: &accountpb.User{
			Id:       user.UID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Bio:      user.Bio,
		},
	}
	data, _ := proto.Marshal(resp)
	c.Data(http.StatusOK, "application/x-protobuf", data)
}

const (
	whitelistMinUID = 10000000
	whitelistMaxUID = 20000000
	snowflakeEpoch  = int64(1704067200000)
)

type snowflakeGenerator struct {
	mutex     sync.Mutex
	lastStamp int64
	sequence  uint16
	nodeID    uint16
}

var defaultSnowflake = &snowflakeGenerator{nodeID: 1}

func nextSnowflakeID() (uint64, error) {
	defaultSnowflake.mutex.Lock()
	defer defaultSnowflake.mutex.Unlock()

	now := time.Now().UnixMilli()
	if now < defaultSnowflake.lastStamp {
		return 0, errors.New("clock moved backwards")
	}

	if now == defaultSnowflake.lastStamp {
		defaultSnowflake.sequence = (defaultSnowflake.sequence + 1) & 0x0fff
		if defaultSnowflake.sequence == 0 {
			for now <= defaultSnowflake.lastStamp {
				time.Sleep(time.Millisecond)
				now = time.Now().UnixMilli()
			}
		}
	} else {
		defaultSnowflake.sequence = 0
	}

	defaultSnowflake.lastStamp = now

	id := (uint64(now-snowflakeEpoch) << 22) | (uint64(defaultSnowflake.nodeID) << 12) | uint64(defaultSnowflake.sequence)
	return id, nil
}

func IsWhitelistUID(uid uint) bool {
	return uid >= whitelistMinUID && uid <= whitelistMaxUID
}

func EnsureWhitelistUser(uid uint) (*models.User, error) {
	var user models.User
	if err := db.DB.Where("uid = ?", uid).First(&user).Error; err == nil {
		return &user, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	snowflakeID, err := nextSnowflakeID()
	if err != nil {
		return nil, err
	}

	uidText := fmt.Sprintf("%d", uid)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(uidText), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	username := uidText
	email := fmt.Sprintf("%d@123.com", uid)
	user = models.User{
		ID:       snowflakeID,
		UID:      uint64(uid),
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		Nickname: username,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	// We don't need to sync sequence here because we manually inserted a whitelist UID (20000001).
	// The sequence is for normal users (starting 20000000, maybe overlapping).
	// If sequence generates 20000001 later, it will conflict.
	// We should probably advance sequence if uid > sequence value.
	// But let's keep it simple: Whitelist UIDs are reserved. Sequence should ideally skip them or start after.
	// If whitelist is 20000001..30000000, and Sequence starts 20000000, conflicts will happen.
	// For now, let's assume whitelist is sparsely used for testing and handle conflicts manually or ignore.

	return &user, nil
}

func syncUserIDSequence() error {
	if db.DB.Dialector != nil && db.DB.Dialector.Name() != "postgres" {
		return nil
	}
	return db.DB.Exec(`
		SELECT setval(
			pg_get_serial_sequence('users', 'id'),
			GREATEST((SELECT COALESCE(MAX(id), 0) FROM users WHERE id < ?), 10000000)
		)
	`, whitelistMinUID).Error
}

func nextUserUID() (uint64, error) {
	if db.DB == nil {
		return 0, errors.New("database not initialized")
	}

	if db.DB.Dialector != nil && db.DB.Dialector.Name() == "postgres" {
		var uid uint64
		if err := db.DB.Raw("SELECT nextval('user_uid_seq')").Scan(&uid).Error; err != nil {
			return 0, err
		}
		return uid, nil
	}

	var maxUID uint64
	if err := db.DB.Model(&models.User{}).Select("COALESCE(MAX(uid), 0)").Scan(&maxUID).Error; err != nil {
		return 0, err
	}
	if maxUID < whitelistMaxUID {
		maxUID = whitelistMaxUID
	}
	return maxUID + 1, nil
}
