package database

import (
	"log"

	"github.com/shigake/tech-iq-back/internal/config"
	"github.com/shigake/tech-iq-back/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	logLevel := logger.Silent
	if cfg.AppEnv == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Database connected successfully")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Technician{},
		&models.Client{},
		&models.Category{},
		&models.Ticket{},
		&models.TicketFile{},
		// Hierarchy Access Control
		&models.Hierarchy{},
		&models.Node{},
		&models.Role{},
		&models.Permission{},
		&models.Membership{},
		&models.AccessAuditLog{},
		// Activity Logs
		&models.ActivityLog{},
	)
	if err != nil {
		log.Println("⚠️ Migration warning (continuing anyway):", err)
		// Continue anyway - tables may already exist
	}

	// Seed default permissions and roles
	SeedAccessControl(db)
	
	// Seed default admin user
	SeedAdminUser(db)

	log.Println("✅ Migrations completed")
	return nil
}

// SeedAccessControl creates default permissions and roles
func SeedAccessControl(db *gorm.DB) {
	log.Println("🔄 Seeding access control data...")

	// Define default permissions
	permissions := []models.Permission{
		// Tickets
		{Code: "tickets.view", Name: "Ver Tickets", Category: "Tickets", Description: "Visualizar tickets"},
		{Code: "tickets.create", Name: "Criar Tickets", Category: "Tickets", Description: "Criar novos tickets"},
		{Code: "tickets.edit", Name: "Editar Tickets", Category: "Tickets", Description: "Editar tickets existentes"},
		{Code: "tickets.delete", Name: "Excluir Tickets", Category: "Tickets", Description: "Excluir tickets"},
		{Code: "tickets.assign", Name: "Atribuir Tickets", Category: "Tickets", Description: "Atribuir tickets a técnicos"},
		// Technicians
		{Code: "technicians.view", Name: "Ver Técnicos", Category: "Técnicos", Description: "Visualizar técnicos"},
		{Code: "technicians.create", Name: "Criar Técnicos", Category: "Técnicos", Description: "Cadastrar novos técnicos"},
		{Code: "technicians.edit", Name: "Editar Técnicos", Category: "Técnicos", Description: "Editar técnicos"},
		{Code: "technicians.delete", Name: "Excluir Técnicos", Category: "Técnicos", Description: "Excluir técnicos"},
		{Code: "technicians.allocate", Name: "Alocar Técnicos", Category: "Técnicos", Description: "Alocar técnicos em tickets"},
		// Clients
		{Code: "clients.view", Name: "Ver Clientes", Category: "Clientes", Description: "Visualizar clientes"},
		{Code: "clients.create", Name: "Criar Clientes", Category: "Clientes", Description: "Cadastrar novos clientes"},
		{Code: "clients.edit", Name: "Editar Clientes", Category: "Clientes", Description: "Editar clientes"},
		{Code: "clients.delete", Name: "Excluir Clientes", Category: "Clientes", Description: "Excluir clientes"},
		// Finance
		{Code: "finance.view", Name: "Ver Financeiro", Category: "Financeiro", Description: "Visualizar dados financeiros"},
		{Code: "finance.create", Name: "Lançar Financeiro", Category: "Financeiro", Description: "Criar lançamentos financeiros"},
		{Code: "finance.approve", Name: "Aprovar Financeiro", Category: "Financeiro", Description: "Aprovar lançamentos financeiros"},
		// Inventory
		{Code: "inventory.view", Name: "Ver Estoque", Category: "Estoque", Description: "Visualizar estoque"},
		{Code: "inventory.manage", Name: "Gerenciar Estoque", Category: "Estoque", Description: "Gerenciar itens do estoque"},
		// Reports
		{Code: "reports.view", Name: "Ver Relatórios", Category: "Relatórios", Description: "Visualizar relatórios"},
		{Code: "reports.export", Name: "Exportar Relatórios", Category: "Relatórios", Description: "Exportar relatórios"},
		// Settings
		{Code: "settings.view", Name: "Ver Configurações", Category: "Configurações", Description: "Visualizar configurações"},
		{Code: "settings.manage", Name: "Gerenciar Configurações", Category: "Configurações", Description: "Alterar configurações do sistema"},
		// Access Control
		{Code: "access.view", Name: "Ver Acessos", Category: "Acessos", Description: "Visualizar hierarquia de acessos"},
		{Code: "access.manage", Name: "Gerenciar Acessos", Category: "Acessos", Description: "Gerenciar acessos de usuários"},
	}

	// Insert permissions if they don't exist
	for _, perm := range permissions {
		var existing models.Permission
		if db.Where("code = ?", perm.Code).First(&existing).RowsAffected == 0 {
			db.Create(&perm)
		}
	}

	// Define default roles
	roles := []struct {
		Role        models.Role
		Permissions []string
	}{
		{
			Role: models.Role{
				Name:        "Administrador",
				Description: "Acesso total ao sistema",
				IsSystem:    true,
			},
			Permissions: []string{}, // Admin bypasses all permissions
		},
		{
			Role: models.Role{
				Name:        "Gerente",
				Description: "Gestão completa da área",
				IsSystem:    false,
			},
			Permissions: []string{
				"tickets.view", "tickets.create", "tickets.edit", "tickets.delete", "tickets.assign",
				"technicians.view", "technicians.create", "technicians.edit", "technicians.allocate",
				"clients.view", "clients.create", "clients.edit",
				"finance.view",
				"inventory.view",
				"reports.view", "reports.export",
				"access.view",
			},
		},
		{
			Role: models.Role{
				Name:        "Operador",
				Description: "Operações básicas do dia a dia",
				IsSystem:    false,
			},
			Permissions: []string{
				"tickets.view", "tickets.create", "tickets.edit",
				"technicians.view", "technicians.allocate",
				"clients.view",
				"inventory.view",
			},
		},
		{
			Role: models.Role{
				Name:        "Visualizador",
				Description: "Apenas visualização",
				IsSystem:    false,
			},
			Permissions: []string{
				"tickets.view",
				"technicians.view",
				"clients.view",
				"inventory.view",
				"reports.view",
			},
		},
	}

	// Insert roles if they don't exist
	for _, r := range roles {
		var existing models.Role
		if db.Where("name = ?", r.Role.Name).First(&existing).RowsAffected == 0 {
			// Get permission entities
			var perms []models.Permission
			if len(r.Permissions) > 0 {
				db.Where("code IN ?", r.Permissions).Find(&perms)
			}
			r.Role.Permissions = perms
			db.Create(&r.Role)
		}
	}

	log.Println("✅ Access control data seeded")
}

// SeedAdminUser creates the default admin user
func SeedAdminUser(db *gorm.DB) {
	log.Println("🔄 Checking admin user...")

	var existing models.User
	if db.Where("email = ?", "admin@techerp.com").First(&existing).RowsAffected > 0 {
		log.Println("✅ Admin user already exists")
		return
	}

	// Create admin user with hashed password
	// Password: admin123
	// Generate hash at runtime to ensure it's valid
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("⚠️ Failed to hash password: %v", err)
		return
	}

	admin := models.User{
		Email:     "admin@techerp.com",
		Password:  string(hashedPassword),
		FirstName: "Administrador",
		LastName:  "Sistema",
		FullName:  "Administrador Sistema",
		Role:      "ADMIN",
		Active:    true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("⚠️ Failed to create admin user: %v", err)
		return
	}

	log.Println("✅ Admin user created (admin@techerp.com / admin123)")
}
