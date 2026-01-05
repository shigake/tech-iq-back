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

// dropOldConstraints removes old unique constraints that block empty values
// PostgreSQL allows multiple NULLs in UNIQUE columns, but not multiple empty strings
// We use partial indexes (WHERE field <> ”) in the model to allow empty values
func dropOldConstraints(db *gorm.DB) {
	constraints := []string{
		"ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_cnpj_key",
		"ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_cpf_key",
		"ALTER TABLE technicians DROP CONSTRAINT IF EXISTS technicians_cpf_key",
	}

	for _, sql := range constraints {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("⚠️ Could not drop constraint: %v", err)
		}
	}
}

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	// Drop old unique constraints that don't allow empty values
	// These will be replaced by partial unique indexes (where field <> '')
	dropOldConstraints(db)

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
		// Geolocation
		&models.TechnicianLocation{},
		&models.TechnicianLastLocation{},
		&models.GeoSettings{},
		// Security and Metrics
		&models.SecurityLog{},
		&models.RequestMetric{},
		// Financial Module
		&models.FinancialEntry{},
		&models.PaymentBatch{},
		&models.FinancialAuditLog{},
		// Stock Module
		&models.StockItem{},
		&models.StockLocation{},
		&models.StockMovement{},
		&models.StockBalance{},
		// Error Logs
		&models.ErrorLog{},
	)
	if err != nil {
		log.Println("⚠️ Migration warning (continuing anyway):", err)
		// Continue anyway - tables may already exist
	}

	// Seed default permissions and roles
	SeedAccessControl(db)

	// Seed default admin user
	SeedAdminUser(db)

	// Seed default financial categories
	SeedFinancialCategories(db)

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
		{
			Role: models.Role{
				Name:        "Técnico",
				Description: "Acesso apenas aos próprios tickets",
				IsSystem:    false,
			},
			Permissions: []string{
				"tickets.view",  // Only own tickets (filtered by user)
				"settings.view", // Can view settings/profile
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

// SeedFinancialCategories creates default financial categories
func SeedFinancialCategories(db *gorm.DB) {
	log.Println("🔄 Seeding financial categories...")

	// Check if financial categories already exist
	var count int64
	db.Model(&models.Category{}).Where("type IN ?", []string{"finance_income", "finance_expense"}).Count(&count)
	if count > 0 {
		log.Println("✅ Financial categories already exist")
		return
	}

	// Income categories
	incomeCategories := []struct {
		Name        string
		Description string
		Icon        string
		Color       string
		Subs        []string
	}{
		{"Serviços", "Receitas de serviços prestados", "service", "#4CAF50", []string{"Conclusão de OS", "Manutenção", "Instalação", "Consultoria"}},
		{"Produtos", "Receitas de venda de produtos", "product", "#2196F3", []string{"Venda de Equipamentos", "Venda de Peças"}},
		{"Outros", "Outras receitas", "money", "#FF9800", []string{"Reembolso", "Bonificação", "Ajuste"}},
	}

	for i, cat := range incomeCategories {
		category := models.Category{
			Name:        cat.Name,
			Description: cat.Description,
			Icon:        cat.Icon,
			Color:       cat.Color,
			Type:        models.CategoryTypeFinanceIncome,
			Active:      true,
			SortOrder:   i,
		}
		if err := db.Create(&category).Error; err != nil {
			log.Printf("⚠️ Failed to create income category %s: %v", cat.Name, err)
			continue
		}
		// Create subcategories
		for j, subName := range cat.Subs {
			subcat := models.Category{
				Name:      subName,
				Type:      models.CategoryTypeFinanceIncome,
				ParentID:  &category.ID,
				Active:    true,
				SortOrder: j,
			}
			db.Create(&subcat)
		}
	}

	// Expense categories
	expenseCategories := []struct {
		Name        string
		Description string
		Icon        string
		Color       string
		Subs        []string
	}{
		{"Pagamento Técnicos", "Pagamentos a técnicos", "payment", "#F44336", []string{"Comissão", "Bonificação", "Reembolso"}},
		{"Operacional", "Despesas operacionais", "operational", "#9C27B0", []string{"Combustível", "Ferramentas", "Equipamentos", "Suprimentos"}},
		{"Administrativo", "Despesas administrativas", "administrative", "#00BCD4", []string{"Aluguel", "Utilidades", "Software", "Serviços"}},
		{"Impostos", "Despesas com impostos", "tax", "#795548", []string{"Federal", "Estadual", "Municipal"}},
		{"Outros", "Outras despesas", "category", "#607D8B", []string{"Ajuste", "Perda"}},
	}

	for i, cat := range expenseCategories {
		category := models.Category{
			Name:        cat.Name,
			Description: cat.Description,
			Icon:        cat.Icon,
			Color:       cat.Color,
			Type:        models.CategoryTypeFinanceExpense,
			Active:      true,
			SortOrder:   i,
		}
		if err := db.Create(&category).Error; err != nil {
			log.Printf("⚠️ Failed to create expense category %s: %v", cat.Name, err)
			continue
		}
		// Create subcategories
		for j, subName := range cat.Subs {
			subcat := models.Category{
				Name:      subName,
				Type:      models.CategoryTypeFinanceExpense,
				ParentID:  &category.ID,
				Active:    true,
				SortOrder: j,
			}
			db.Create(&subcat)
		}
	}

	log.Println("✅ Financial categories seeded")
}
