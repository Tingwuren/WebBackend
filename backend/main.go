package main

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type User struct {
	gorm.Model
	Username string `gorm:"type:varchar(100);uniqueIndex"`
	Password string
	Email    string `gorm:"type:varchar(100);uniqueIndex"`
	Level    int
	Token    string
}

type Product struct {
	gorm.Model
	Name     string
	Category string
	Price    float64
	Stock    int
}
type Order struct {
	gorm.Model
	UserID    uint
	ProductID uint
	Quantity  int
	Total     float64
	Status    string
}

type Comment struct {
	gorm.Model
	UserID    uint
	ProductID uint
	Content   string
}

var db *gorm.DB

var client = redis.NewClient(&redis.Options{
	// Addr: "redis:6379",
	Addr: "127.0.0.1:6379",
	DB:   0,
})

func CheckPermission(token string) int { //0 success 1 failed
	var user User
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		return 1
	}
	if user.Level != 2 {
		return 1
	}
	return 0
}

func CheckLogin(token string) uint {
	var user User
	if err := db.Where("token = ?", token).First(&user).Error; err != nil {
		return 0
	}
	return user.ID
}

func main() {
	// db, _ = gorm.Open(sqlite.Open("/app/data/gorm.db"), &gorm.Config{})
	db, _ = gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	db.AutoMigrate(User{})
	db.AutoMigrate(Product{})
	db.AutoMigrate(Order{})
	db.AutoMigrate(Comment{})
	router := gin.Default()
	router.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 密码加密
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		user.Password = string(hashedPassword)
		// 默认用户等级为1
		user.Level = 1
		uuid := uuid.New()
		user.Token = uuid.String()
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Registration failed, please try again."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": user.Token})
	})
	router.POST("/login", func(c *gin.Context) {
		var user User
		var loginUser User
		if err := c.ShouldBindJSON(&loginUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Where("username = ?", loginUser.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist."})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginUser.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect password."})
			return
		}
		uuid := uuid.New()
		user.Token = uuid.String()
		db.Save(&user)
		c.JSON(http.StatusOK, gin.H{"data": user.Token})
	})
	router.POST("/logout", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Where("token = ?", user.Token).First(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist."})
			return
		}
		user.Token = ""
		db.Save(&user)
		c.JSON(http.StatusOK, gin.H{"data": "Logout successful."})
	})
	router.POST("/admin/product", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckPermission(tokenParts[1]) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var product Product
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Create(&product).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product creation failed, please try again."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": product})
	})
	router.GET("/admin/product/:id", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckPermission(tokenParts[1]) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var product Product
		if err := db.Where("id = ?", c.Param("id")).First(&product).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product does not exist."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": product})

	})
	router.DELETE("/admin/product/:id", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckPermission(tokenParts[1]) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var product Product
		if err := db.Where("id = ?", c.Param("id")).First(&product).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product does not exist."})
			return
		}
		db.Delete(&product)
		c.JSON(http.StatusOK, gin.H{"data": "Product deletion successful."})
	})
	router.GET("/product", func(c *gin.Context) {
		// Try to get products from cache
		val, err := client.Get("products").Result()
		if errors.Is(err, redis.Nil) {
			// Cache miss, get products from database
			var products []Product
			if err := db.Find(&products).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving products."})
				return
			}
			// Cache products in Redis, expire after 5 seconds
			productsJson, _ := json.Marshal(products)
			err = client.Set("products", productsJson, 5*time.Second).Err()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Error caching products."})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": products})
		} else if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving products from cache."})
		} else {
			// Cache hit, return products from cache
			var products []Product
			json.Unmarshal([]byte(val), &products)
			c.JSON(http.StatusOK, gin.H{"data": products})
		}
	})
	router.POST("/order", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		user := CheckLogin(tokenParts[1])
		if user == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var order Order
		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var product Product
		if err := db.First(&product, order.ProductID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product does not exist."})
			return
		}
		if product.Stock < order.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient product stock."})
			return
		}
		product.Stock -= order.Quantity
		order.Total = float64(order.Quantity) * product.Price
		order.Status = "unpaid"
		order.UserID = user
		db.Save(&product)
		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order creation failed, please try again."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	})
	router.GET("/order", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		user := CheckLogin(tokenParts[1])
		if user == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var orders []Order
		if err := db.Where("user_id = ?", user).Order("created_at desc").Find(&orders).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving orders."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": orders})

	})
	router.GET("/order/:id", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckLogin(tokenParts[1]) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var order Order
		if err := db.Where("id = ?", c.Param("id")).First(&order).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order does not exist."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	})
	router.POST("/order/pay", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckLogin(tokenParts[1]) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var payInfo struct {
			OrderID uint
		}
		if err := c.ShouldBindJSON(&payInfo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var order Order
		if err := db.First(&order, payInfo.OrderID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order does not exist."})
			return
		}
		if order.Status != "unpaid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order has been paid or cancelled."})
			return
		}
		order.Status = "paid"
		db.Save(&order)
		c.JSON(http.StatusOK, gin.H{"data": "Payment successful."})
	})
	router.GET("/comment/:productID", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		if CheckLogin(tokenParts[1]) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var comments []Comment
		if err := db.Where("product_id = ?", c.Param("productID")).Order("created_at desc").Find(&comments).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving comments."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": comments})
	})
	router.POST("/comment", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		user := CheckLogin(tokenParts[1])
		if user == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied."})
			return
		}
		var comment Comment
		if err := c.ShouldBindJSON(&comment); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		comment.UserID = user
		if err := db.Create(&comment).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Comment creation failed, please try again."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": comment})
	})
	router.Run(":8080")
}
