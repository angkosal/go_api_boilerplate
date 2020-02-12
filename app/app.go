package app

import (
	"fmt"
	"go_api_boilerplate/configs"
	"go_api_boilerplate/domain/user"
	"go_api_boilerplate/gql"
	"go_api_boilerplate/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "go_api_boilerplate/docs" // docs is generated by Swag CLI

	"go_api_boilerplate/controllers"
	"go_api_boilerplate/repositories/userrepo"
	"go_api_boilerplate/services/authservice"
	"go_api_boilerplate/services/userservice"

	_ "github.com/lib/pq" // For Postgres setup
)

var (
	router = gin.Default()
)

// @title Go API Boilerplate Swagger
// @version 1.0
// @description This is Go API Boilerplate
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://github.com/yhagio/go_api_boilerplate/blob/master/LICENSE

// @host localhost:3000
// @BasePath /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func Run() {
	// ====== Swagger setup (http://localhost:3000/swagger/index.html) ============
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ====== Setup configs ============
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	config := configs.GetConfig()

	// Connects to PostgresDB
	db, err := gorm.Open(
		config.Postgres.Dialect(),
		config.Postgres.GetPostgresConnectionInfo(),
	)
	if err != nil {
		panic(err)
	}

	// Migration
	// db.DropTableIfExists(&user.User{})
	db.AutoMigrate(&user.User{})
	defer db.Close()

	// ====== Setup infra ==============

	// ====== Setup repositories =======
	userRepo := userrepo.NewUserRepo(db)

	// ====== Setup services ===========
	userService := userservice.NewUserService(userRepo, config.Pepper)
	authService := authservice.NewAuthService(config.JWTSecret)

	// ====== Setup controllers ========
	userCtl := controllers.NewUserController(userService, authService)

	// ====== Setup middlewares ========
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// ====== Setup routes =============
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	router.GET("/graphql", gql.PlaygroundHandler("/query"))
	router.POST("/query",
		middlewares.SetUserContext(config.JWTSecret),
		gql.GraphqlHandler(userService, authService))

	api := router.Group("/api")

	api.POST("/register", userCtl.Register)
	api.POST("/login", userCtl.Login)
	api.POST("/forgot_password", userCtl.ForgotPassword) // TODO
	api.POST("/reset_password", userCtl.ResetPassword)   // TODO

	user := api.Group("/users")

	user.GET("/:id", userCtl.GetByID)

	account := api.Group("/account")
	account.Use(middlewares.RequireLoggedIn(config.JWTSecret))
	{
		account.GET("/profile", userCtl.GetProfile)
		account.PUT("/profile", userCtl.Update)
	}

	// Run
	// port := fmt.Sprintf(":%s", viper.Get("APP_PORT"))
	port := fmt.Sprintf(":%s", config.Port)
	router.Run(port)
}
