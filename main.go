package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var usernameDB, passwordDB, addressDB, portDB, dbName, portApp, signatureKey string

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	usernameDB = os.Getenv("usernameDB")
	if usernameDB == "" {
		log.Fatal("Username not defined")
	}

	passwordDB = os.Getenv("passwordDB")
	if passwordDB == "" {
		log.Fatal("Password not defined")
	}

	addressDB = os.Getenv("addressDB")
	if addressDB == "" {
		log.Fatal("address DB not defined")
	}

	portDB = os.Getenv("portDB")
	if portDB == "" {
		portDB = "3306"
	}

	dbName = os.Getenv("DBName")
	if dbName == "" {
		log.Fatal("Database name not defined")
	}

	portApp = os.Getenv("portApp")
	if portApp == "" {
		portApp = "9000"
	}

	signatureKey = os.Getenv("signatureKey")
	if dbName == "" {
		log.Fatal("Signature key not defined")
	}

	initDb()
}

func catch() {

	if r := recover(); r != nil {
		log.Println("Error occured", r)
	} else {
		log.Println("Application running perfectly")
	}

}

func main() {

	defer catch()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
		AllowAllOrigins:  true,
	}))
	r.GET("/healthCheck", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
		return
	})
	r.POST("/capexAsset", createCapexAsset)
	r.Use(middleware)
	r.GET("/capexTrx", getCapexTrx)
	r.POST("/capexTrx", createCapexTrx)
	r.GET("/capexTrx/:id", getCapexTrxDetail)
	r.PUT("/capexTrx/:id", updateCapexTrx)
	r.POST("/capexTrx/:id/replicate", replicateCapex)
	r.GET("/capexAsset/:id", getCapexAsset)
	r.POST("/approve", approveCapex)
	r.GET("/rules", getRules)
	r.GET("/createInfo", getCreateInfo)
	r.POST("/reject", rejectCapex)
	r.GET("/user/:id", getUser)
	r.PUT("/user/:id", updateUser)
	r.POST("/user", createUser)
	r.POST("/login", login)

	r.Run(":" + portApp)
}

func middleware(c *gin.Context) {
	path := c.Request.URL.Path
	if path == "/register" || path == "/login" {
		return
	}
	authorizationHeader := c.GetHeader("Authorization")
	if !strings.Contains(authorizationHeader, "Bearer") {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Invalid Authorization"))
		c.JSON(http.StatusUnauthorized, "Invalid Authorization")
		return
	}

	tokenString := strings.Replace(authorizationHeader, "Bearer ", "", -1)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Signing method invalid")
		} else if method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("Signning method invalid")
		}

		return []byte(signatureKey), nil
	})

	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err,
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		c.AbortWithError(http.StatusUnauthorized, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err,
		})
	}

	c.Set("ID", claims["id"])

	c.Next()
}
