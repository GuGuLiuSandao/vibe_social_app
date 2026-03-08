package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/logger"
	"social_app/internal/models"
	accountpb "social_app/internal/proto/account"
	commonpb "social_app/internal/proto/common"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	logger.Init()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gormDB.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.DB = gormDB
}

func TestEnsureWhitelistUserCreatesDefaults(t *testing.T) {
	setupTestDB(t)

	uid := uint(20000000)
	user, err := EnsureWhitelistUser(uid)
	if err != nil {
		t.Fatalf("ensure whitelist user: %v", err)
	}

	expected := "20000000"
	if user.Username != expected {
		t.Fatalf("username mismatch: %s", user.Username)
	}
	if user.Nickname != expected {
		t.Fatalf("nickname mismatch: %s", user.Nickname)
	}
	if user.Email != "20000000@123.com" {
		t.Fatalf("email mismatch: %s", user.Email)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(expected)); err != nil {
		t.Fatalf("password mismatch: %v", err)
	}
}

func TestRegisterWithProtobuf(t *testing.T) {
	setupTestDB(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	reqBody, err := proto.Marshal(&accountpb.RegisterRequest{
		Username: "neo",
		Email:    "neo@example.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	c.Request = req

	cfg := &config.Config{JWTSecret: "test-secret"}
	Register(c, cfg)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp accountpb.RegisterResponse
	if err := proto.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK {
		t.Fatalf("unexpected error_code: %v", resp.ErrorCode)
	}
	if resp.Token == "" {
		t.Fatalf("missing token")
	}
	if resp.User == nil || resp.User.Email != "neo@example.com" {
		t.Fatalf("unexpected user")
	}
}

func TestLoginRejectsJSON(t *testing.T) {
	setupTestDB(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	body := []byte(`{"uid":20000001}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	cfg := &config.Config{JWTSecret: "test-secret"}
	Login(c, cfg)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestLoginWithProtobuf(t *testing.T) {
	setupTestDB(t)

	hashed, err := bcrypt.GenerateFromPassword([]byte("secret12"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		UID:      10000001,
		Username: "neo",
		Email:    "neo@example.com",
		Password: string(hashed),
		Nickname: "neo",
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	reqBody, err := proto.Marshal(&accountpb.LoginRequest{
		Email:    "neo@example.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	c.Request = req

	cfg := &config.Config{JWTSecret: "test-secret"}
	Login(c, cfg)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp accountpb.LoginResponse
	if err := proto.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK {
		t.Fatalf("unexpected error_code: %v", resp.ErrorCode)
	}
	if resp.Token == "" {
		t.Fatalf("missing token")
	}
	if resp.User == nil || resp.User.Email != "neo@example.com" {
		t.Fatalf("unexpected user")
	}
}

func TestLoginWhitelistWithProtobufUID(t *testing.T) {
	setupTestDB(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	reqBody, err := proto.Marshal(&accountpb.LoginRequest{
		Uid: 20000000,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	c.Request = req

	cfg := &config.Config{JWTSecret: "test-secret"}
	Login(c, cfg)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp accountpb.LoginResponse
	if err := proto.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK {
		t.Fatalf("unexpected error_code: %v", resp.ErrorCode)
	}
	if resp.Token == "" {
		t.Fatalf("missing token")
	}
	if resp.User == nil || resp.User.Email != "20000000@123.com" {
		t.Fatalf("unexpected user")
	}
}

func TestGetCurrentUserWithProtobuf(t *testing.T) {
	setupTestDB(t)

	user := models.User{
		UID:      10000001,
		Username: "neo",
		Email:    "neo@example.com",
		Password: "secret12",
		Nickname: "neo",
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cfg := &config.Config{JWTSecret: "test-secret"}
	token, err := GenerateToken(user.ID, user.Username, cfg)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	reqBody, err := proto.Marshal(&accountpb.AuthRequest{
		Token: token,
		Uid:   uint64(user.ID),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/me", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	c.Request = req

	GetCurrentUser(c, cfg)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var resp accountpb.AuthResponse
	if err := proto.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ErrorCode != commonpb.ErrorCode_ERROR_CODE_OK {
		t.Fatalf("unexpected error_code: %v", resp.ErrorCode)
	}
	if resp.User == nil || resp.User.Email != "neo@example.com" {
		t.Fatalf("unexpected user")
	}
}
