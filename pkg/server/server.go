package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"crypto/subtle"

	"github.com/gin-gonic/gin"
	"github.com/mkhsnw/gateprint/pkg/config"
	"github.com/mkhsnw/gateprint/pkg/entity"
	"github.com/mkhsnw/gateprint/pkg/middleware"
	"github.com/mkhsnw/gateprint/pkg/service"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var currentAppKey string
var currentServerPort string
var currentListenLocal bool
var activeServer *http.Server
var appCtx context.Context

func InitServerState() {
	config.InitStore()
	data := config.LoadData()
	currentAppKey = data.AppKey
	currentServerPort = data.ServerPort
	currentListenLocal = data.ListenLocal
	if currentServerPort == "" {
		currentServerPort = "9100"
	}
}

func SetAppKey(key string) {
	currentAppKey = key

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
		StartServer(appCtx)
	}
}

func GetServerPort() string {
	if currentServerPort == "" {
		InitServerState()
	}
	return currentServerPort
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
		time.Sleep(200 * time.Millisecond)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CorsMiddleware())

	router.Use(func(c *gin.Context) {
		if currentAppKey != "" && c.Request.Method != "OPTIONS" {
			reqKey := c.GetHeader("App-Key")
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

	host := ""
	if currentListenLocal {
		host = "127.0.0.1"
	}
	addr := host + ":" + port

	activeServer = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	EmitLog(ctx, "INFO", "Gateprint server started on "+addr)
	go func() {
		if err := activeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			EmitLog(ctx, "ERROR", fmt.Sprintf("Failed to start server: %v", err))
		}
	}()
}
