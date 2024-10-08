package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"debate_web/internal/models"
	"debate_web/internal/service"
	"debate_web/internal/utils"
)

// AuthHandler 處理與認證相關的請求
type AuthHandler struct {
	userService *service.UserService
}

// NewAuthHandler 創建一個新的 AuthHandler 實例
func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// LoginInput 定義登入請求的結構
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterInput 定義註冊請求的結構
type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 處理用戶註冊
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	// 解析並驗證請求體
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 對密碼進行加密
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := models.User{
		Username: input.Username,
		Password: string(hashedPassword),
	}

	// 創建新用戶
	if err := h.userService.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "創建使用者失敗"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "使用者註冊成功"})
}

// Login 處理用戶登入
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	// 解析並驗證請求體
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 檢查用戶是否存在
	user, err := h.userService.GetUserByUsername(input.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 生成 JWT token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "獲取token失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
