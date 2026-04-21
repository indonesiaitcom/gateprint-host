package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mkhsnw/gateprint/pkg/config"
	"github.com/mkhsnw/gateprint/pkg/entity"
	"github.com/mkhsnw/gateprint/pkg/middleware"
	"github.com/mkhsnw/gateprint/pkg/service"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"crypto/subtle"
)

var currentAppKey string
var currentServerPort string
var currentAllowedOrigin string
var activeServer *http.Server
var appCtx context.Context

func InitServerState() {
	config.InitStore()
	data := config.LoadData()
	currentAppKey = data.AppKey
	currentServerPort = data.ServerPort
	currentAllowedOrigin = data.AllowedOrigin
	if currentServerPort == "" {
		currentServerPort = "9100"
	}
	if currentAllowedOrigin == "" {
		currentAllowedOrigin = "*"
	}
}

func SetAppKey(key string) {
	currentAppKey = key

	// Persist
	data := config.LoadData()
	data.AppKey = key
	config.SaveData(data)
}

func GetAppKey() string {
	return currentAppKey
}

func SetServerPort(port string) {
	currentServerPort = port

	// Persist
	data := config.LoadData()
	data.ServerPort = port
	config.SaveData(data)

	if appCtx != nil {
		StartServer(appCtx) // Restart server on new port
	}
}

func GetServerPort() string {
	if currentServerPort == "" {
		InitServerState()
	}
	return currentServerPort
}

func SetAllowedOrigin(origin string) {
	currentAllowedOrigin = origin
	data := config.LoadData()
	data.AllowedOrigin = origin
	config.SaveData(data)
	if appCtx != nil {
		StartServer(appCtx)
	}
}

func GetAllowedOrigin() string {
	if currentAllowedOrigin == "" {
		InitServerState()
	}
	return currentAllowedOrigin
}

func EmitLog(ctx context.Context, level, message string) {
	if ctx == nil {
		fmt.Printf("[%s] %s\n", level, message)
		return
	}
	timestamp := time.Now().Format("15:04:05")
	formattedMsg := fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
	fmt.Println(formattedMsg)
	runtime.EventsEmit(ctx, "server-log", formattedMsg)
}

func StartServer(ctx context.Context) {
	appCtx = ctx
	InitServerState()

	port := currentServerPort

	if activeServer != nil {
		EmitLog(ctx, "INFO", "Stopping existing server...")
		activeServer.Shutdown(context.Background())
		time.Sleep(200 * time.Millisecond) // Let port release
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CorsMiddleware(currentAllowedOrigin))

	// Auth Middleware
	router.Use(func(c *gin.Context) {
		if currentAppKey != "" && c.Request.Method != "OPTIONS" {
			reqKey := c.GetHeader("App-Key")
			// Use constant time compare to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(reqKey), []byte(currentAppKey)) != 1 {
				EmitLog(ctx, "ERROR", fmt.Sprintf("Unauthorized access attempt from %s (Invalid App-Key)", c.ClientIP()))
				c.JSON(401, gin.H{"error": "Unauthorized: Invalid App-Key"})
				c.Abort()
				return
			}
		}
		c.Next()
	})

	router.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "online", "version": "1.0.0"})
	})

	router.POST("/print", func(c *gin.Context) {
		var job entity.PrintJob
		if err := c.ShouldBindJSON(&job); err != nil {
			EmitLog(ctx, "ERROR", "Invalid print request payload")
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		EmitLog(ctx, "INFO", "Received print job request")

		if err := service.ExecutePrint(job); err != nil {
			EmitLog(ctx, "ERROR", fmt.Sprintf("Print failed: %v", err))
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		EmitLog(ctx, "SUCCESS", "Print job completed successfully")
		c.JSON(200, gin.H{"message": "Success"})
	})

	activeServer = &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	EmitLog(ctx, "INFO", "Gateprint server started on port :"+port)
	go func() {
		if err := activeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			EmitLog(ctx, "ERROR", fmt.Sprintf("Failed to start server: %v", err))
		}
	}()
}
