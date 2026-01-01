package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/shigake/tech-iq-back/internal/cache"
	"github.com/shigake/tech-iq-back/internal/config"
	"github.com/shigake/tech-iq-back/internal/database"
	"github.com/shigake/tech-iq-back/internal/handlers"
	"github.com/shigake/tech-iq-back/internal/middleware"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/shigake/tech-iq-back/internal/services"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis cache
	var redisClient *cache.RedisClient
	if cfg.CacheEnabled {
		redisClient = cache.NewRedisClient(&cache.CacheConfig{
			Host:     cfg.RedisHost,
			Port:     cfg.RedisPort,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		
		if err := redisClient.Ping(); err != nil {
			log.Printf("⚠️  Failed to connect to Redis, running without cache: %v", err)
			redisClient = nil
		} else {
			log.Println("✅ Redis cache connected successfully")
			defer redisClient.Close()
		}
	} else {
		log.Println("ℹ️  Cache disabled by configuration")
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: handlers.ErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CorsOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	technicianRepo := repositories.NewTechnicianRepository(db)
	ticketRepo := repositories.NewTicketRepository(db)
	clientRepo := repositories.NewClientRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	hierarchyRepo := repositories.NewHierarchyRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg)
	technicianService := services.NewTechnicianService(technicianRepo, redisClient)
	ticketService := services.NewTicketService(ticketRepo, technicianRepo, clientRepo, categoryRepo)
	dashboardService := services.NewDashboardService(technicianRepo, ticketRepo, clientRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	technicianHandler := handlers.NewTechnicianHandler(technicianService)
	ticketHandler := handlers.NewTicketHandler(ticketService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	clientHandler := handlers.NewClientHandler(clientRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryRepo)
	termsHandler := handlers.NewTermsHandler()
	exportHandler := handlers.NewExportHandler(clientRepo, technicianRepo, ticketRepo)
	hierarchyHandler := handlers.NewHierarchyHandler(hierarchyRepo)
	userHandler := handlers.NewUserHandler(userRepo)

	// Routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// Terms of service (public access)
	api.Get("/terms", termsHandler.GetTerms)

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/signin", authHandler.SignIn)
	auth.Post("/signup", authHandler.SignUp)
	auth.Post("/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := api.Group("", middleware.JWTProtected(cfg.JWTSecret))

	// Protected auth routes
	protected.Post("/auth/change-password", authHandler.ChangePassword)

	// User management routes (admin)
	users := protected.Group("/users")
	users.Get("/", userHandler.GetUsers)
	users.Get("/search", userHandler.SearchUsers)
	users.Get("/:id", userHandler.GetUser)
	users.Post("/", userHandler.CreateUser)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", userHandler.DeleteUser)
	users.Post("/:id/reset-password", userHandler.ResetPassword)
	users.Post("/:id/toggle-status", userHandler.ToggleUserStatus)

	// Technician routes
	technicians := protected.Group("/technicians")
	technicians.Get("/", technicianHandler.GetAll)
	technicians.Get("/:id", technicianHandler.GetByID)
	technicians.Post("/", middleware.WriteAccess(), technicianHandler.Create)
	technicians.Put("/:id", middleware.WriteAccess(), technicianHandler.Update)
	technicians.Delete("/:id", middleware.WriteAccess(), technicianHandler.Delete)
	technicians.Get("/search", technicianHandler.Search)
	technicians.Get("/by-city/:city", technicianHandler.GetByCity)
	technicians.Get("/by-state/:state", technicianHandler.GetByState)

	// Ticket routes
	tickets := protected.Group("/tickets")
	tickets.Get("/", ticketHandler.GetAll)
	tickets.Get("/:id", ticketHandler.GetByID)
	tickets.Post("/", middleware.WriteAccess(), ticketHandler.Create)
	tickets.Put("/:id", middleware.WriteAccess(), ticketHandler.Update)
	tickets.Delete("/:id", middleware.WriteAccess(), ticketHandler.Delete)
	tickets.Put("/:id/status", middleware.WriteAccess(), ticketHandler.UpdateStatus)
	tickets.Put("/:id/assign", middleware.WriteAccess(), ticketHandler.AssignTechnician)

	// Client routes
	clients := protected.Group("/clients")
	clients.Get("/", clientHandler.GetAll)
	clients.Get("/count", clientHandler.Count)
	clients.Get("/:id", clientHandler.GetByID)
	clients.Post("/", middleware.WriteAccess(), clientHandler.Create)
	clients.Put("/:id", middleware.WriteAccess(), clientHandler.Update)
	clients.Delete("/:id", middleware.WriteAccess(), clientHandler.Delete)

	// Category routes
	categories := protected.Group("/categories")
	categories.Get("/", categoryHandler.GetAll)
	categories.Get("/:id", categoryHandler.GetByID)
	categories.Post("/", middleware.WriteAccess(), categoryHandler.Create)
	categories.Put("/:id", middleware.WriteAccess(), categoryHandler.Update)
	categories.Delete("/:id", middleware.WriteAccess(), categoryHandler.Delete)

	// Dashboard routes
	dashboard := protected.Group("/dashboard")
	dashboard.Get("/stats", dashboardHandler.GetStats)
	dashboard.Get("/tickets-by-status", dashboardHandler.GetTicketsByStatus)
	dashboard.Get("/technicians-by-state", dashboardHandler.GetTechniciansByState)
	dashboard.Get("/chart", dashboardHandler.GetChartData)
	dashboard.Get("/recent-activity", dashboardHandler.GetRecentActivity)

	// Cities endpoint for technicians
	technicians.Get("/cities", technicianHandler.GetCities)

	// Terms accept (protected)
	protected.Post("/terms/accept", termsHandler.AcceptTerms)

	// Export routes (admin and employee access)
	export := protected.Group("/export", middleware.AdminOrEmployee())
	export.Get("/clients", exportHandler.ExportClients)
	export.Get("/technicians", exportHandler.ExportTechnicians)
	export.Get("/tickets", exportHandler.ExportTickets)
	export.Get("/all", exportHandler.ExportAll)

	// ==================== Hierarchy Access Control Routes ====================
	// Hierarchies
	hierarchies := protected.Group("/hierarchies")
	hierarchies.Get("/", hierarchyHandler.GetAllHierarchies)
	hierarchies.Get("/:id", hierarchyHandler.GetHierarchy)
	hierarchies.Post("/", middleware.WriteAccess(), hierarchyHandler.CreateHierarchy)
	hierarchies.Put("/:id", middleware.WriteAccess(), hierarchyHandler.UpdateHierarchy)
	hierarchies.Delete("/:id", middleware.WriteAccess(), hierarchyHandler.DeleteHierarchy)

	// Nodes (within hierarchy)
	hierarchies.Post("/:id/nodes", middleware.WriteAccess(), hierarchyHandler.CreateNode)

	// Node operations
	nodes := protected.Group("/nodes")
	nodes.Put("/:id", middleware.WriteAccess(), hierarchyHandler.UpdateNode)
	nodes.Put("/:id/move", middleware.WriteAccess(), hierarchyHandler.MoveNode)
	nodes.Delete("/:id", middleware.WriteAccess(), hierarchyHandler.DeleteNode)
	nodes.Get("/:id/members", hierarchyHandler.GetNodeMembers)
	nodes.Post("/:id/members", middleware.WriteAccess(), hierarchyHandler.AddNodeMember)

	// Memberships
	memberships := protected.Group("/memberships")
	memberships.Put("/:id", middleware.WriteAccess(), hierarchyHandler.UpdateMembership)
	memberships.Delete("/:id", middleware.WriteAccess(), hierarchyHandler.DeleteMembership)

	// Roles
	roles := protected.Group("/roles")
	roles.Get("/", hierarchyHandler.GetAllRoles)
	roles.Get("/:id", hierarchyHandler.GetRole)
	roles.Post("/", middleware.WriteAccess(), hierarchyHandler.CreateRole)
	roles.Put("/:id", middleware.WriteAccess(), hierarchyHandler.UpdateRole)
	roles.Delete("/:id", middleware.WriteAccess(), hierarchyHandler.DeleteRole)

	// Permissions
	protected.Get("/permissions", hierarchyHandler.GetAllPermissions)

	// Access simulation and history
	access := protected.Group("/access")
	access.Post("/simulate", hierarchyHandler.SimulateAccess)
	access.Get("/user/:userId", hierarchyHandler.GetUserAccess)
	access.Get("/history", hierarchyHandler.GetHistory)
	access.Post("/history/:id/revert", middleware.WriteAccess(), hierarchyHandler.RevertChange)

	// Start server
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📚 API docs: http://localhost:%s/api/v1/health", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
