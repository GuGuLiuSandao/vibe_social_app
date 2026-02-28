package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"social_app/internal/logger"
	commonpb "social_app/internal/proto/common"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func UploadAvatar(c *gin.Context) {
	// 1. Parse Multipart Form
	file, err := c.FormFile("file")
	if err != nil {
		resp := &commonpb.UploadFileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "file is required",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	// 2. Validate File
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		resp := &commonpb.UploadFileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "only image files are allowed",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	if file.Size > 5*1024*1024 { // 5MB
		resp := &commonpb.UploadFileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message:   "file size too large (max 5MB)",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusBadRequest, "application/x-protobuf", data)
		return
	}

	// 3. Save File
	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		logger.Error("Failed to create upload directory: %v", err)
		resp := &commonpb.UploadFileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "internal server error",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	dst := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		logger.Error("Failed to save uploaded file: %v", err)
		resp := &commonpb.UploadFileResponse{
			ErrorCode: commonpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message:   "failed to save file",
		}
		data, _ := proto.Marshal(resp)
		c.Data(http.StatusInternalServerError, "application/x-protobuf", data)
		return
	}

	// 4. Return URL
	// Construct public URL
	// Assuming server is running on config.ServerPort
	// Or we can just return relative path if frontend handles base URL
	// Or absolute URL.
	// For simplicity, let's use relative path "/uploads/avatars/" + filename
	// But frontend needs full URL for avatar preview usually?
	// Actually, if we serve static files at /uploads, then http://host:port/uploads/avatars/... works.
	// We should probably return the full URL if possible, or just the path.
	// Let's return the full URL if we can get the host.
	// Gin context has Request.Host.
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/uploads/avatars/%s", scheme, c.Request.Host, filename)

	resp := &commonpb.UploadFileResponse{
		ErrorCode: commonpb.ErrorCode_ERROR_CODE_OK,
		Message:   "upload success",
		Url:       url,
	}
	data, _ := proto.Marshal(resp)
	c.Data(http.StatusOK, "application/x-protobuf", data)
}
