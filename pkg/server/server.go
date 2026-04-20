package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mkhsnw/gateprint/pkg/config"
	"github.com/mkhsnw/gateprint/pkg/entity"
	"github.com/mkhsnw/gateprint/pkg/middleware"
	"github.com/mkhsnw/gateprint/pkg/service"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var currentAppKey string

func InitServerState() {
	config.InitStore()
	data := config.LoadData()
	currentAppKey = data.AppKey
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
	InitServerState()
	cfg := config.LoadConfig()

	port := cfg.PORT

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(middleware.CorsMiddleware(cfg.ALLOWED_ORIGIN))

	// Auth Middleware
	router.Use(func(c *gin.Context) {
		// Only check POST /print requests or you can check all. We check all.
		if currentAppKey != "" && c.Request.Method != "OPTIONS" {
			reqKey := c.GetHeader("App-Key")
			if reqKey != currentAppKey {
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

	EmitLog(ctx, "INFO", "Thermal Bridge started on port :"+port)
	if err := router.Run(":" + port); err != nil {
		EmitLog(ctx, "ERROR", fmt.Sprintf("Failed to start server: %v", err))
	}
}
