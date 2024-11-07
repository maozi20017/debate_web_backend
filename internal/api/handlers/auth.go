package handlers

import (
	"debate_web/internal/repository/models"
	"debate_web/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler 處理認證相關的請求
type AuthHandler struct {
	userService *service.UserService
}

// NewAuthHandler 創建新的認證處理器
func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// LoginInput 定義登入請求的結構
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterInput 定義註冊請求的結構
type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=32"`
}

// Register 處理用戶註冊
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "資料格式不正確",
			"details": err.Error(),
		})
		return
	}

	// 檢查用戶名是否已存在
	exist, _ := h.userService.CheckUserExists(input.Username)
	if exist {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用戶名已被使用",
		})
		return
	}

	// 密碼加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "密碼加密失敗",
		})
		return
	}

	// 創建用戶
	user := &models.User{
		Username: input.Username,
		Password: string(hashedPassword),
	}

	if err := h.userService.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "創建用戶失敗",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "註冊成功",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// Login 處理用戶登入
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "資料格式不正確",
			"details": err.Error(),
		})
		return
	}

	// 查找用戶
	user, err := h.userService.GetUserByUsername(input.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用戶名或密碼錯誤",
		})
		return
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用戶名或密碼錯誤",
		})
		return
	}

	// 生成 JWT token
	token, err := h.userService.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成token失敗",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登入成功",
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}
