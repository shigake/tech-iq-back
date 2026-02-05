package main

import (
	"log"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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

// Build info - injected at compile time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	CommitSHA = "unknown"
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

	// Security headers (XSS, Content-Type sniffing, etc)
	app.Use(helmet.New())

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	technicianRepo := repositories.NewTechnicianRepository(db)
	ticketRepo := repositories.NewTicketRepository(db)
	clientRepo := repositories.NewClientRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	hierarchyRepo := repositories.NewHierarchyRepository(db)
	activityLogRepo := repositories.NewActivityLogRepository(db)
	geoRepo := repositories.NewGeoRepository(db)
	securityLogRepo := repositories.NewSecurityLogRepository(db)
	financialRepo := repositories.NewFinancialRepository(db)
	stockRepo := repositories.NewStockRepository(db)
	errorLogRepo := repositories.NewErrorLogRepository(db)
	cityRepo := repositories.NewCityRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, securityLogRepo, hierarchyRepo, cfg)
	technicianService := services.NewTechnicianService(technicianRepo, redisClient)
	ticketService := services.NewTicketService(ticketRepo, technicianRepo, clientRepo, categoryRepo, hierarchyRepo)
	dashboardService := services.NewDashboardService(technicianRepo, ticketRepo, clientRepo)
	activityLogService := services.NewActivityLogService(activityLogRepo)
	hierarchyService := services.NewHierarchyService(hierarchyRepo)
	cityService := services.NewCityService(cityRepo)
	geoService := services.NewGeoService(geoRepo, userRepo, technicianRepo, ticketRepo, hierarchyService, redisClient, cityService)
	geocodingService := services.NewGeocodingService(technicianRepo)
	securityLogService := services.NewSecurityLogService(securityLogRepo)
	systemMetricsService := services.NewSystemMetricsService(db, redisClient, userRepo, ticketRepo, securityLogRepo)
	financialService := services.NewFinancialService(financialRepo, categoryRepo)
	stockService := services.NewStockService(stockRepo)
	errorLogService := services.NewErrorLogService(errorLogRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	technicianHandler := handlers.NewTechnicianHandlerWithAudit(technicianService, db)
	ticketHandler := handlers.NewTicketHandler(ticketService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	clientHandler := handlers.NewClientHandler(clientRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryRepo)
	termsHandler := handlers.NewTermsHandler()
	exportHandler := handlers.NewExportHandler(clientRepo, technicianRepo, ticketRepo, stockRepo, financialRepo, categoryRepo)
	importHandler := handlers.NewImportHandler(clientRepo, ticketRepo, categoryRepo, technicianRepo)
	hierarchyHandler := handlers.NewHierarchyHandler(hierarchyRepo)
	userHandler := handlers.NewUserHandler(userRepo)
	activityLogHandler := handlers.NewActivityLogHandler(activityLogService)
	geoHandler := handlers.NewGeoHandler(geoService, geocodingService)
	securityLogHandler := handlers.NewSecurityLogHandler(securityLogService)
	adminHandler := handlers.NewAdminHandler(systemMetricsService)
	financialHandler := handlers.NewFinancialHandler(financialService, categoryRepo)
	stockHandler := handlers.NewStockHandler(stockService)
	errorLogHandler := handlers.NewErrorLogHandler(errorLogService)

	middleware.SetHierarchyRepository(hierarchyRepo)

	// Error logging middleware (add before routes)
	app.Use(middleware.ErrorLoggerMiddleware(errorLogService))

	// Routes
	api := app.Group("/api/v1")

	// Terms of service (public access)
	api.Get("/terms", termsHandler.GetTerms)

	// Health check (public)
	api.Get("/health", adminHandler.GetHealthCheck)

	// Version info (public)
	api.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version":    Version,
			"build_time": BuildTime,
			"commit":     CommitSHA,
			"go_version": runtime.Version(),
		})
	})

	// Rate limiter for auth routes (5 requests per minute per IP)
	authLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again later.",
			})
		},
	})

	// Auth routes (public) with rate limiting
	auth := api.Group("/auth")
	auth.Post("/signin", authLimiter, authHandler.SignIn)
	auth.Post("/signup", authLimiter, authHandler.SignUp)
	auth.Post("/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := api.Group("", middleware.JWTProtected(cfg.JWTSecret))

	// Protected auth routes
	protected.Post("/auth/change-password", authHandler.ChangePassword)
	protected.Get("/auth/me", authHandler.GetMe)

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
	technicians.Post("/", middleware.RequirePermission("technicians.create"), technicianHandler.Create)
	technicians.Put("/:id", middleware.RequirePermission("technicians.edit"), technicianHandler.Update)
	technicians.Delete("/:id", middleware.RequirePermission("technicians.delete"), technicianHandler.Delete)
	technicians.Get("/search", technicianHandler.Search)
	technicians.Get("/by-city/:city", technicianHandler.GetByCity)
	technicians.Get("/by-state/:state", technicianHandler.GetByState)

	// Ticket routes
	tickets := protected.Group("/tickets")
	tickets.Get("/", ticketHandler.GetAll)
	tickets.Get("/:id", ticketHandler.GetByID)
	tickets.Post("/", middleware.RequirePermission("tickets.create"), ticketHandler.Create)
	tickets.Put("/:id", middleware.RequirePermission("tickets.edit"), ticketHandler.Update)
	tickets.Delete("/:id", middleware.RequirePermission("tickets.delete"), ticketHandler.Delete)
	tickets.Put("/:id/status", middleware.RequirePermission("tickets.edit"), ticketHandler.UpdateStatus)
	tickets.Put("/:id/assign", middleware.RequirePermission("tickets.assign"), ticketHandler.AssignTechnician)
	tickets.Post("/:id/sign", middleware.RequirePermission("tickets.edit"), ticketHandler.SignTicket)
	tickets.Delete("/:id/sign", middleware.RequirePermission("tickets.delete"), ticketHandler.DeleteSignature)

	// Client routes
	clients := protected.Group("/clients")
	clients.Get("/", clientHandler.GetAll)
	clients.Get("/count", clientHandler.Count)
	clients.Get("/:id", clientHandler.GetByID)
	clients.Post("/", middleware.RequirePermission("clients.create"), clientHandler.Create)
	clients.Put("/:id", middleware.RequirePermission("clients.edit"), clientHandler.Update)
	clients.Delete("/:id", middleware.RequirePermission("clients.delete"), clientHandler.Delete)

	// Category routes
	categories := protected.Group("/categories")
	categories.Get("/", categoryHandler.GetAll)
	categories.Get("/:id", categoryHandler.GetByID)
	categories.Post("/", middleware.RequirePermission("settings.manage"), categoryHandler.Create)
	categories.Put("/:id", middleware.RequirePermission("settings.manage"), categoryHandler.Update)
	categories.Delete("/:id", middleware.RequirePermission("settings.manage"), categoryHandler.Delete)

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

	// Export routes (requires export permission)
	export := protected.Group("/export", middleware.RequirePermission("reports.export"))
	export.Get("/clients", exportHandler.ExportClients)
	export.Get("/technicians", exportHandler.ExportTechnicians)
	export.Get("/tickets", exportHandler.ExportTickets)
	export.Get("/all", exportHandler.ExportAll)
	export.Get("/stock/items", exportHandler.ExportStockItems)
	export.Get("/stock/locations", exportHandler.ExportStockLocations)
	export.Get("/stock/movements", exportHandler.ExportStockMovements)
	export.Get("/financial/entries", exportHandler.ExportFinancialEntries)
	export.Get("/categories", exportHandler.ExportCategories)

	// Import routes (requires import permission)
	importRoutes := protected.Group("/import", middleware.RequireAnyPermission("technicians.create", "tickets.create", "clients.create"))
	importRoutes.Get("/tickets/template", importHandler.DownloadTicketTemplate)
	importRoutes.Post("/tickets", importHandler.ImportTickets)
	importRoutes.Get("/technicians/template", importHandler.DownloadTechnicianTemplate)
	importRoutes.Post("/technicians", importHandler.ImportTechnicians)

	// ==================== Hierarchy Access Control Routes ====================
	// Hierarchies
	hierarchies := protected.Group("/hierarchies")
	hierarchies.Get("/", hierarchyHandler.GetAllHierarchies)
	hierarchies.Get("/:id", hierarchyHandler.GetHierarchy)
	hierarchies.Post("/", middleware.RequirePermission("access.manage"), hierarchyHandler.CreateHierarchy)
	hierarchies.Put("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.UpdateHierarchy)
	hierarchies.Delete("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.DeleteHierarchy)

	// Nodes (within hierarchy)
	hierarchies.Post("/:id/nodes", middleware.RequirePermission("access.manage"), hierarchyHandler.CreateNode)

	// Node operations
	nodes := protected.Group("/nodes")
	nodes.Get("/", hierarchyHandler.GetAllNodes)
	nodes.Put("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.UpdateNode)
	nodes.Put("/:id/move", middleware.RequirePermission("access.manage"), hierarchyHandler.MoveNode)
	nodes.Delete("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.DeleteNode)
	nodes.Get("/:id/members", hierarchyHandler.GetNodeMembers)
	nodes.Post("/:id/members", middleware.RequirePermission("access.manage"), hierarchyHandler.AddNodeMember)

	// Memberships
	memberships := protected.Group("/memberships")
	memberships.Put("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.UpdateMembership)
	memberships.Delete("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.DeleteMembership)

	// Roles
	roles := protected.Group("/roles")
	roles.Get("/", hierarchyHandler.GetAllRoles)
	roles.Get("/:id", hierarchyHandler.GetRole)
	roles.Post("/", middleware.RequirePermission("access.manage"), hierarchyHandler.CreateRole)
	roles.Put("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.UpdateRole)
	roles.Delete("/:id", middleware.RequirePermission("access.manage"), hierarchyHandler.DeleteRole)

	// Permissions
	protected.Get("/permissions", hierarchyHandler.GetAllPermissions)

	// Activity logs
	activityLogs := protected.Group("/activity-logs")
	activityLogs.Get("/", activityLogHandler.GetActivityLogs)
	activityLogs.Get("/me", activityLogHandler.GetMyActivityLogs)
	activityLogs.Get("/recent", activityLogHandler.GetRecentActivityLogs)
	activityLogs.Get("/:id", activityLogHandler.GetActivityLogByID)

	// Access simulation and history
	access := protected.Group("/access")
	access.Post("/simulate", hierarchyHandler.SimulateAccess)
	access.Get("/user/:userId", hierarchyHandler.GetUserAccess)
	access.Get("/history", hierarchyHandler.GetHistory)
	access.Post("/history/:id/revert", middleware.RequirePermission("access.manage"), hierarchyHandler.RevertChange)

	// ==================== Geolocation Routes ====================
	geo := protected.Group("/geo")
	// Technician endpoints (send location)
	geo.Post("/locations", geoHandler.CreateLocation)
	geo.Post("/locations/batch", geoHandler.CreateBatchLocations)
	// Manager endpoints (view locations)
	geo.Get("/technicians/last", geoHandler.GetTechniciansLastLocations)
	geo.Get("/technicians/:id/history", geoHandler.GetTechnicianHistory)
	geo.Get("/tickets/:id/locations", geoHandler.GetTicketLocations)
	// Admin endpoints (settings and cache)
	geo.Get("/settings", geoHandler.GetGeoSettings)
	geo.Put("/settings", middleware.RequirePermission("settings.manage"), geoHandler.UpdateGeoSettings)
	geo.Post("/cache/refresh", middleware.RequirePermission("settings.manage"), geoHandler.RefreshGeoCache)
	geo.Get("/cache/status", geoHandler.GetGeoCacheStatus)
	// Geocoding endpoints
	geo.Post("/geocode/batch", middleware.RequirePermission("settings.manage"), geoHandler.GeocodeAllTechnicians)
	geo.Get("/geocode/status", geoHandler.GetGeocodingStatus)
	geo.Post("/geocode/:id", middleware.RequirePermission("technicians.edit"), geoHandler.GeocodeSingleTechnician)
	// Cities endpoints
	geo.Post("/cities/load", middleware.RequirePermission("settings.manage"), geoHandler.LoadCities)
	geo.Get("/cities/count", geoHandler.GetCityCount)

	// ==================== Admin Routes ====================
	admin := protected.Group("/admin")
	// Security logs (admin only)
	admin.Get("/security-logs", securityLogHandler.GetSecurityLogs)
	admin.Get("/security-logs/recent", securityLogHandler.GetRecentSecurityLogs)
	admin.Get("/security-logs/stats", securityLogHandler.GetSecurityStats)
	// System metrics (admin only)
	admin.Get("/system-metrics", adminHandler.GetSystemMetrics)

	// ==================== Error Logs Routes ====================
	// Frontend errors (any authenticated user can submit)
	protected.Post("/errors/frontend", errorLogHandler.CreateFromFrontend)

	// Admin routes for managing error logs
	errors := protected.Group("/errors", middleware.RequirePermission("settings.manage"))
	errors.Get("/", errorLogHandler.GetAll)
	errors.Get("/stats", errorLogHandler.GetStats)
	errors.Get("/:id", errorLogHandler.GetByID)
	errors.Post("/:id/resolve", errorLogHandler.Resolve)
	errors.Post("/bulk-resolve", errorLogHandler.BulkResolve)
	errors.Delete("/:id", errorLogHandler.Delete)
	errors.Post("/cleanup", errorLogHandler.Cleanup)

	// ==================== Financial Routes ====================
	financial := protected.Group("/financial")
	// Categories (public info)
	financial.Get("/categories", financialHandler.GetCategories)
	// Dashboard and reports
	financial.Get("/dashboard", financialHandler.GetDashboard)
	financial.Get("/reports/cash-flow", financialHandler.GetCashFlowReport)
	financial.Get("/reports/technician-payments", financialHandler.GetTechnicianPaymentsReport)
	// Financial entries
	entries := financial.Group("/entries")
	entries.Get("/", financialHandler.ListEntries)
	entries.Get("/:id", financialHandler.GetEntry)
	entries.Post("/", middleware.RequirePermission("finance.create"), financialHandler.CreateEntry)
	entries.Put("/:id", middleware.RequirePermission("finance.edit"), financialHandler.UpdateEntry)
	entries.Patch("/:id/status", middleware.RequirePermission("finance.edit"), financialHandler.UpdateEntryStatus)
	entries.Delete("/:id", middleware.RequirePermission("finance.delete"), financialHandler.DeleteEntry)
	// Payment batches (requires finance.manage permission)
	batches := financial.Group("/batches", middleware.RequirePermission("finance.manage"))
	batches.Get("/", financialHandler.ListBatches)
	batches.Get("/:id", financialHandler.GetBatch)
	batches.Post("/", financialHandler.CreateBatch)
	batches.Delete("/:id", financialHandler.DeleteBatch)
	batches.Post("/:id/entries", financialHandler.AddEntriesToBatch)
	batches.Delete("/:id/entries/:entryId", financialHandler.RemoveEntryFromBatch)
	batches.Patch("/:id/approve", financialHandler.ApproveBatch)
	batches.Patch("/:id/pay", financialHandler.PayBatch)

	// ==================== Stock Module Routes ====================
	stockHandler.RegisterRoutes(app, middleware.JWTProtected(cfg.JWTSecret))

	// ==================== Audit Routes ====================
	handlers.RegisterAuditRoutes(api.Group("", middleware.JWTProtected(cfg.JWTSecret)), db)

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
