package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/xuri/excelize/v2"
)

type ExportHandler struct {
	clientRepo     repositories.ClientRepository
	technicianRepo repositories.TechnicianRepository
	ticketRepo     repositories.TicketRepository
	stockRepo      repositories.StockRepository
	financialRepo  *repositories.FinancialRepository
	categoryRepo   repositories.CategoryRepository
}

func NewExportHandler(
	clientRepo repositories.ClientRepository,
	technicianRepo repositories.TechnicianRepository,
	ticketRepo repositories.TicketRepository,
	stockRepo repositories.StockRepository,
	financialRepo *repositories.FinancialRepository,
	categoryRepo repositories.CategoryRepository,
) *ExportHandler {
	return &ExportHandler{
		clientRepo:     clientRepo,
		technicianRepo: technicianRepo,
		ticketRepo:     ticketRepo,
		stockRepo:      stockRepo,
		financialRepo:  financialRepo,
		categoryRepo:   categoryRepo,
	}
}

// ExportClients exports clients data as CSV
func (h *ExportHandler) ExportClients(c *fiber.Ctx) error {
	clients, err := h.clientRepo.GetAllWithoutPagination()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar clientes",
			"error":   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=clientes_"+time.Now().Format("20060102_150405")+".csv")

	var csvData strings.Builder
	csvData.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&csvData)

	header := []string{"ID", "Nome Completo", "CPF", "CNPJ", "Email", "Telefone", "Rua", "Numero", "Bairro", "Cidade", "Estado", "CEP", "Data de Criacao"}
	writer.Write(header)

	// Write data
	for _, client := range clients {
		record := []string{
			client.ID,
			client.FullName,
			client.CPF,
			client.CNPJ,
			client.Email,
			client.Phone,
			client.Street,
			client.Number,
			client.Neighborhood,
			client.City,
			client.State,
			client.ZipCode,
			client.CreatedAt.Format("02/01/2006 15:04:05"),
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

func (h *ExportHandler) ExportTechnicians(c *fiber.Ctx) error {
	search := c.Query("search")
	status := c.Query("status")
	techType := c.Query("type")
	city := c.Query("city")
	state := c.Query("state")

	var technicians []models.Technician
	var err error

	if search != "" || status != "" || techType != "" || city != "" || state != "" {
		technicians, _, err = h.technicianRepo.SearchWithFilters(search, status, techType, city, state, 0, 100000)
	} else {
		technicians, err = h.technicianRepo.GetAll()
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar técnicos",
			"error":   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=tecnicos_"+time.Now().Format("20060102_150405")+".csv")

	var csvData strings.Builder
	csvData.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&csvData)

	header := []string{
		"Nome", "Status", "Tipo", "Emails", "Telefones",
		"Valor Minimo", "Observacao", "CPF", "CNPJ",
		"Banco", "Agencia", "Conta", "Tipo Conta", "Titular", "Chave Pix",
		"Infraestrutura", "Hardware", "Software", "Impressoras", "Help Desk",
		"CFTV", "Cabeamento", "Banco de Dados", "Redes/Firewall",
		"Rua", "Numero", "Bairro", "Cidade", "Estado", "CEP", "Complemento",
	}
	writer.Write(header)

	skillNames := []string{
		"Infraestrutura", "Hardware", "Software", "Impressoras", "Help Desk",
		"CFTV", "Cabeamento", "Banco de Dados", "Redes/Firewall",
	}

	for _, tech := range technicians {
		var emailList []string
		for _, e := range tech.Emails {
			emailList = append(emailList, e.Email)
		}
		emails := strings.Join(emailList, "; ")

		var phoneList []string
		for _, p := range tech.Phones {
			phoneList = append(phoneList, p.Number)
		}
		phones := strings.Join(phoneList, "; ")

		var skills []string
		for _, skillName := range skillNames {
			if tech.Skills[skillName] {
				skills = append(skills, "Sim")
			} else {
				skills = append(skills, "Nao")
			}
		}

		record := []string{
			tech.FullName, tech.Status, tech.Type, emails, phones,
			tech.MinCallValue, tech.Observation, tech.CPF, tech.CNPJ,
			tech.BankName, tech.Agency, tech.AccountNumber, tech.AccountType, tech.AccountHolder, tech.PixKey,
		}
		record = append(record, skills...)
		record = append(record, tech.Street, tech.Number, tech.Neighborhood, tech.City, tech.State, tech.ZipCode, tech.Complement)

		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportTickets exports tickets data as CSV
func (h *ExportHandler) ExportTickets(c *fiber.Ctx) error {
	tickets, err := h.ticketRepo.GetAllWithoutPagination()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar tickets",
			"error":   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=tickets_"+time.Now().Format("20060102_150405")+".csv")

	var csvData strings.Builder
	csvData.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&csvData)

	header := []string{
		"Referencia Externa", "Codigo Loja", "Nome Loja", "Rua", "Numero",
		"Cidade", "Estado", "CEP", "Descricao do Erro", "Contato", "Telefone Contato",
		"Prioridade", "Categoria", "Numero OS", "Status", "Data de Criacao",
	}
	writer.Write(header)

	for _, ticket := range tickets {
		categoryName := ""
		if ticket.Category != nil {
			categoryName = ticket.Category.Name
		}

		record := []string{
			ticket.ExternalReference,
			ticket.StoreCode,
			ticket.StoreName,
			ticket.ServiceStreet,
			ticket.ServiceNumber,
			ticket.ServiceCity,
			ticket.ServiceState,
			ticket.ServiceZipCode,
			ticket.ErrorDescription,
			ticket.ContactName,
			ticket.ContactPhone,
			string(ticket.Priority),
			categoryName,
			ticket.OSNumber,
			string(ticket.Status),
			ticket.CreatedAt.Format("02/01/2006 15:04:05"),
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportAll exports all data in a single CSV file with multiple sheets simulation
func (h *ExportHandler) ExportAll(c *fiber.Ctx) error {
	// For CSV, we'll create separate sections
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Add a header indicating this is a complete export
	writer.Write([]string{"[ EXPORTAÇÃO COMPLETA TECH-ERP ]", time.Now().Format("02/01/2006 15:04:05")})
	writer.Write([]string{""}) // Empty line

	// Export clients section
	writer.Write([]string{"[ CLIENTES ]"})
	clients, err := h.clientRepo.GetAllWithoutPagination()
	if err == nil {
		clientHeader := []string{"ID", "Nome Completo", "CPF", "CNPJ", "Email", "Telefone", "Cidade", "Estado", "CEP", "Data de Criação"}
		writer.Write(clientHeader)

		for _, client := range clients {
			record := []string{
				client.ID,
				client.FullName,
				client.CPF,
				client.CNPJ,
				client.Email,
				client.Phone,
				client.City,
				client.State,
				client.ZipCode,
				client.CreatedAt.Format("02/01/2006 15:04:05"),
			}
			writer.Write(record)
		}
	}

	writer.Write([]string{""}) // Empty line

	// Export technicians section
	writer.Write([]string{"[ TÉCNICOS ]"})
	technicians, err := h.technicianRepo.GetAll()
	if err == nil {
		techHeader := []string{"ID", "Nome", "CPF", "CNPJ", "Status", "Tipo", "Cidade", "Estado", "Data de Criação"}
		writer.Write(techHeader)

		for _, tech := range technicians {
			record := []string{
				tech.ID,
				tech.FullName,
				tech.CPF,
				tech.CNPJ,
				tech.Status,
				tech.Type,
				tech.City,
				tech.State,
				tech.CreatedAt.Format("02/01/2006 15:04:05"),
			}
			writer.Write(record)
		}
	}

	writer.Write([]string{""}) // Empty line

	// Export tickets section
	writer.Write([]string{"[ TICKETS ]"})
	tickets, err := h.ticketRepo.GetAllWithoutPagination()
	if err == nil {
		ticketHeader := []string{
			"ID", "Número OS", "Descrição do Erro", "Status", "Prioridade",
			"Cliente", "Categoria", "Técnicos", "Data de Criação", "Data de Atualização",
		}
		writer.Write(ticketHeader)

		for _, ticket := range tickets {
			clientName := ""
			if ticket.Client != nil {
				clientName = ticket.Client.FullName
			}

			technicianNames := ""
			for i, tech := range ticket.Technicians {
				if i > 0 {
					technicianNames += ", "
				}
				technicianNames += tech.FullName
			}

			categoryName := ""
			if ticket.Category != nil {
				categoryName = ticket.Category.Name
			}

			record := []string{
				ticket.ID,
				ticket.OSNumber,
				ticket.ErrorDescription,
				string(ticket.Status),
				string(ticket.Priority),
				clientName,
				categoryName,
				technicianNames,
				ticket.CreatedAt.Format("02/01/2006 15:04:05"),
				ticket.UpdatedAt.Format("02/01/2006 15:04:05"),
			}
			writer.Write(record)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=backup_completo_"+time.Now().Format("20060102_150405")+".csv")

	return c.SendString(csvData.String())
}

// ExportStockItems exports stock items data as CSV
func (h *ExportHandler) ExportStockItems(c *fiber.Ctx) error {
	// Get all stock items
	filter := models.StockItemFilter{
		Page:     0,
		PageSize: 1000000,
	}
	result, err := h.stockRepo.ListItems(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar itens de estoque",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=estoque_itens_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "SKU", "Nome", "Descrição", "Categoria", "Unidade", "Qtd Mínima", "Ativo", "Data de Criação"}
	writer.Write(header)

	// Write data
	for _, item := range result.Data {
		description := ""
		if item.Description != nil {
			description = *item.Description
		}
		category := ""
		if item.Category != nil {
			category = *item.Category
		}
		record := []string{
			item.ID,
			item.SKU,
			item.Name,
			description,
			category,
			item.Unit,
			fmt.Sprintf("%d", item.MinQty),
			fmt.Sprintf("%t", item.IsActive),
			item.CreatedAt.Format("02/01/2006 15:04:05"),
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportStockLocations exports stock locations data as CSV
func (h *ExportHandler) ExportStockLocations(c *fiber.Ctx) error {
	// Get all stock locations
	filter := models.StockLocationFilter{
		Page:     0,
		PageSize: 1000000,
	}
	result, err := h.stockRepo.ListLocations(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar locais de estoque",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=estoque_locais_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Nome", "Tipo", "Ativo", "Data de Criação"}
	writer.Write(header)

	// Write data
	for _, loc := range result.Data {
		record := []string{
			loc.ID,
			loc.Name,
			string(loc.Type),
			fmt.Sprintf("%t", loc.IsActive),
			loc.CreatedAt.Format("02/01/2006 15:04:05"),
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportStockMovements exports stock movements data as CSV
func (h *ExportHandler) ExportStockMovements(c *fiber.Ctx) error {
	// Get all stock movements
	filter := models.StockMovementFilter{
		Page:     0,
		PageSize: 1000000,
	}
	result, err := h.stockRepo.ListMovements(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar movimentações de estoque",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=estoque_movimentacoes_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Tipo", "Item", "De Local", "Para Local", "Quantidade", "Observações", "Executado Por", "Data"}
	writer.Write(header)

	// Write data
	for _, mov := range result.Data {
		itemName := ""
		if mov.Item != nil {
			itemName = mov.Item.Name
		}
		fromLocation := ""
		if mov.FromLocation != nil {
			fromLocation = mov.FromLocation.Name
		}
		toLocation := ""
		if mov.ToLocation != nil {
			toLocation = mov.ToLocation.Name
		}
		notes := ""
		if mov.Notes != nil {
			notes = *mov.Notes
		}

		record := []string{
			mov.ID,
			string(mov.Type),
			itemName,
			fromLocation,
			toLocation,
			fmt.Sprintf("%d", mov.Quantity),
			notes,
			mov.PerformedBy,
			mov.PerformedAt.Format("02/01/2006 15:04:05"),
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportFinancialEntries exports financial entries data as CSV
func (h *ExportHandler) ExportFinancialEntries(c *fiber.Ctx) error {
	// Get all financial entries
	filter := models.FinancialEntryFilter{
		Page:  0,
		Limit: 1000000,
	}
	entries, _, err := h.financialRepo.ListEntries(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar lançamentos financeiros",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=financeiro_lancamentos_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Tipo", "Categoria", "Subcategoria", "Descrição", "Valor", "Status", "Data Lançamento", "Data Vencimento", "Data Pagamento", "Cliente", "Técnico"}
	writer.Write(header)

	// Write data
	for _, entry := range entries {
		clientName := ""
		if entry.Client != nil {
			clientName = entry.Client.FullName
		}
		technicianName := ""
		if entry.Technician != nil {
			technicianName = entry.Technician.FullName
		}
		paymentDate := ""
		if entry.PaymentDate != nil {
			paymentDate = entry.PaymentDate.Format("02/01/2006")
		}
		dueDate := ""
		if entry.DueDate != nil {
			dueDate = entry.DueDate.Format("02/01/2006")
		}

		record := []string{
			entry.ID,
			string(entry.Type),
			entry.Category,
			entry.Subcategory,
			entry.Description,
			fmt.Sprintf("%.2f", entry.Amount),
			string(entry.Status),
			entry.EntryDate.Format("02/01/2006"),
			dueDate,
			paymentDate,
			clientName,
			technicianName,
		}
		writer.Write(record)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

// ExportCategories exports categories data as CSV
func (h *ExportHandler) ExportCategories(c *fiber.Ctx) error {
	// Get all categories
	categories, err := h.categoryRepo.GetAll()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar categorias",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=categorias_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Nome", "Tipo", "Descrição", "Cor", "Ícone", "Ordem", "Ativo"}
	writer.Write(header)

	// Write data (including children)
	for _, cat := range categories {
		record := []string{
			cat.ID,
			cat.Name,
			string(cat.Type),
			cat.Description,
			cat.Color,
			cat.Icon,
			fmt.Sprintf("%d", cat.SortOrder),
			fmt.Sprintf("%t", cat.Active),
		}
		writer.Write(record)

		// Write children
		for _, child := range cat.Children {
			childRecord := []string{
				child.ID,
				"  └── " + child.Name,
				string(child.Type),
				child.Description,
				child.Color,
				child.Icon,
				fmt.Sprintf("%d", child.SortOrder),
				fmt.Sprintf("%t", child.Active),
			}
			writer.Write(childRecord)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar CSV",
			"error":   err.Error(),
		})
	}

	return c.SendString(csvData.String())
}

func (h *ExportHandler) ExportStockBalancesXLSX(c *fiber.Ctx) error {
	filter := models.StockBalanceFilter{
		Page:     1,
		PageSize: 1000000,
	}
	result, err := h.stockRepo.ListBalances(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar saldos de estoque",
			"error":   err.Error(),
		})
	}

	itemsFilter := models.StockItemFilter{
		Page:     0,
		PageSize: 1000000,
	}
	itemsResult, err := h.stockRepo.ListItems(itemsFilter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar itens de estoque",
			"error":   err.Error(),
		})
	}

	itemsMap := make(map[string]models.StockItem)
	for _, item := range itemsResult.Data {
		itemsMap[item.ID] = item
	}

	balancesByLocation := make(map[string][]models.StockBalanceResponse)
	for _, balance := range result.Data {
		key := balance.LocationName
		balancesByLocation[key] = append(balancesByLocation[key], balance)
	}

	locationNames := make([]string, 0, len(balancesByLocation))
	for name := range balancesByLocation {
		locationNames = append(locationNames, name)
	}
	sort.Strings(locationNames)

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	numberStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	f.DeleteSheet("Sheet1")

	for _, locationName := range locationNames {
		balances := balancesByLocation[locationName]

		sheetName := locationName
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}
		sheetName = strings.ReplaceAll(sheetName, "/", "-")
		sheetName = strings.ReplaceAll(sheetName, "\\", "-")
		sheetName = strings.ReplaceAll(sheetName, "?", "")
		sheetName = strings.ReplaceAll(sheetName, "*", "")
		sheetName = strings.ReplaceAll(sheetName, "[", "(")
		sheetName = strings.ReplaceAll(sheetName, "]", ")")
		sheetName = strings.ReplaceAll(sheetName, ":", "-")

		f.NewSheet(sheetName)

		headers := []string{"Item", "Quantidade", "Unidade", "Local de Estoque", "SKU", "Ativo"}
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 1)
			f.SetCellValue(sheetName, cell, header)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		for row, balance := range balances {
			rowNum := row + 2

			item, hasItem := itemsMap[balance.ItemID]
			isActive := "Sim"
			if hasItem && !item.IsActive {
				isActive = "Não"
			}

			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), balance.ItemName)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), balance.Quantity)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), balance.ItemUnit)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), balance.LocationName)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), balance.ItemSKU)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), isActive)

			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), dataStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("B%d", rowNum), fmt.Sprintf("B%d", rowNum), numberStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("C%d", rowNum), dataStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), dataStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("E%d", rowNum), fmt.Sprintf("E%d", rowNum), dataStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("F%d", rowNum), fmt.Sprintf("F%d", rowNum), numberStyle)
		}

		f.SetColWidth(sheetName, "A", "A", 40)
		f.SetColWidth(sheetName, "B", "B", 12)
		f.SetColWidth(sheetName, "C", "C", 10)
		f.SetColWidth(sheetName, "D", "D", 25)
		f.SetColWidth(sheetName, "E", "E", 20)
		f.SetColWidth(sheetName, "F", "F", 8)
	}

	if len(locationNames) > 0 {
		firstSheetName := locationNames[0]
		if len(firstSheetName) > 31 {
			firstSheetName = firstSheetName[:31]
		}
		firstSheetName = strings.ReplaceAll(firstSheetName, "/", "-")
		firstSheetName = strings.ReplaceAll(firstSheetName, "\\", "-")
		firstSheetName = strings.ReplaceAll(firstSheetName, "?", "")
		firstSheetName = strings.ReplaceAll(firstSheetName, "*", "")
		firstSheetName = strings.ReplaceAll(firstSheetName, "[", "(")
		firstSheetName = strings.ReplaceAll(firstSheetName, "]", ")")
		firstSheetName = strings.ReplaceAll(firstSheetName, ":", "-")
		
		sheetIdx, _ := f.GetSheetIndex(firstSheetName)
		f.SetActiveSheet(sheetIdx)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar arquivo Excel",
			"error":   err.Error(),
		})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=estoque_por_local_"+time.Now().Format("20060102_150405")+".xlsx")

	return c.Send(buffer.Bytes())
}

func (h *ExportHandler) getCutoffReports(c *fiber.Ctx) ([]models.TechnicianCutoffReport, error) {
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	technicianID := c.Query("technicianId")

	if startDateStr == "" || endDateStr == "" {
		return nil, fmt.Errorf("startDate e endDate são obrigatórios")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, fmt.Errorf("formato de startDate inválido, esperado YYYY-MM-DD")
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, fmt.Errorf("formato de endDate inválido, esperado YYYY-MM-DD")
	}

	return h.financialRepo.GetTechnicianCutoffReport(startDate, endDate, technicianID)
}

func (h *ExportHandler) ExportCutoffReportXLSX(c *fiber.Ctx) error {
	reports, err := h.getCutoffReports(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Relatório de Corte"
	f.SetSheetName("Sheet1", sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	techHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9E2F3"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    164,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		NumFmt:    164,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	row := 1

	if len(reports) > 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Relatório de Corte - Técnicos")
		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 14},
			Alignment: &excelize.Alignment{Horizontal: "left"},
		})
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
		row++

		period := fmt.Sprintf("Período: %s a %s", reports[0].PeriodStart, reports[0].PeriodEnd)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), period)
		row += 2
	}

	var grandTotal float64

	for _, report := range reports {
		f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row))
		techInfo := report.TechnicianName
		if report.CPF != "" {
			techInfo += " - CPF: " + report.CPF
		} else if report.CNPJ != "" {
			techInfo += " - CNPJ: " + report.CNPJ
		}
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), techInfo)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row), techHeaderStyle)
		row++

		if report.BankName != "" || report.PixKey != "" {
			bankInfo := ""
			if report.BankName != "" {
				bankInfo = fmt.Sprintf("Banco: %s | Ag: %s | Cc: %s", report.BankName, report.Agency, report.AccountNumber)
			}
			if report.PixKey != "" {
				if bankInfo != "" {
					bankInfo += " | "
				}
				bankInfo += "PIX: " + report.PixKey
			}
			f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row))
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), bankInfo)
			row++
		}

		headers := []string{"Nº OS", "Cliente", "Data Fechamento", "Assinatura", "Valor Aceito"}
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheetName, cell, header)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
		row++

		for _, entry := range report.Entries {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.OSNumber)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.ClientName)
			f.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)

			closedAt := ""
			if entry.ClosedAt != nil {
				closedAt = entry.ClosedAt.Format("02/01/2006")
			}
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), closedAt)
			f.SetCellStyle(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

			sig := "Não"
			if entry.HasSignature {
				sig = "Sim"
			}
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), sig)
			f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.AcceptedValue)
			f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), currencyStyle)

			row++
		}

		f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row))
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Total - "+report.TechnicianName)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), totalStyle)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), report.TotalAmount)
		f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), totalStyle)
		grandTotal += report.TotalAmount
		row += 2
	}

	f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row))
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "TOTAL GERAL")
	grandTotalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		NumFmt:    164,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), grandTotalStyle)
	f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), grandTotal)
	f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), grandTotalStyle)

	f.SetColWidth(sheetName, "A", "A", 18)
	f.SetColWidth(sheetName, "B", "B", 30)
	f.SetColWidth(sheetName, "C", "C", 18)
	f.SetColWidth(sheetName, "D", "D", 12)
	f.SetColWidth(sheetName, "E", "E", 18)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar arquivo Excel"})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=relatorio_corte_"+time.Now().Format("20060102_150405")+".xlsx")

	return c.Send(buffer.Bytes())
}

func (h *ExportHandler) ExportCutoffReportPDF(c *fiber.Ctx) error {
	reports, err := h.getCutoffReports(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetMargins(10, 10, 10)

	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "Relatorio de Corte - Tecnicos", "", 1, "C", false, 0, "")

	if len(reports) > 0 {
		pdf.SetFont("Helvetica", "", 10)
		period := fmt.Sprintf("Periodo: %s a %s", reports[0].PeriodStart, reports[0].PeriodEnd)
		pdf.CellFormat(0, 7, period, "", 1, "C", false, 0, "")
	}
	pdf.Ln(5)

	var grandTotal float64

	for _, report := range reports {
		if pdf.GetY() > 170 {
			pdf.AddPage()
		}

		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(217, 226, 243)
		techInfo := report.TechnicianName
		if report.CPF != "" {
			techInfo += " - CPF: " + report.CPF
		} else if report.CNPJ != "" {
			techInfo += " - CNPJ: " + report.CNPJ
		}
		pdf.CellFormat(0, 8, techInfo, "1", 1, "L", true, 0, "")

		if report.BankName != "" || report.PixKey != "" {
			pdf.SetFont("Helvetica", "", 8)
			bankInfo := ""
			if report.BankName != "" {
				bankInfo = fmt.Sprintf("Banco: %s | Ag: %s | Cc: %s", report.BankName, report.Agency, report.AccountNumber)
			}
			if report.PixKey != "" {
				if bankInfo != "" {
					bankInfo += " | "
				}
				bankInfo += "PIX: " + report.PixKey
			}
			pdf.CellFormat(0, 6, bankInfo, "LR", 1, "L", false, 0, "")
		}

		colWidths := []float64{40, 80, 40, 30, 40, 47}

		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(68, 114, 196)
		pdf.SetTextColor(255, 255, 255)
		headers := []string{"N OS", "Cliente", "Data Fech.", "Assinatura", "Valor Aceito", ""}
		for i, header := range headers {
			if i < len(colWidths) && colWidths[i] > 0 && header != "" {
				pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", true, 0, "")
			}
		}
		pdf.Ln(-1)
		pdf.SetTextColor(0, 0, 0)

		pdf.SetFont("Helvetica", "", 9)
		for _, entry := range report.Entries {
			if pdf.GetY() > 185 {
				pdf.AddPage()
			}

			closedAt := ""
			if entry.ClosedAt != nil {
				closedAt = entry.ClosedAt.Format("02/01/2006")
			}
			sig := "Nao"
			if entry.HasSignature {
				sig = "Sim"
			}

			pdf.CellFormat(colWidths[0], 6, entry.OSNumber, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colWidths[1], 6, truncateStr(entry.ClientName, 40), "1", 0, "L", false, 0, "")
			pdf.CellFormat(colWidths[2], 6, closedAt, "1", 0, "C", false, 0, "")
			pdf.CellFormat(colWidths[3], 6, sig, "1", 0, "C", false, 0, "")
			pdf.CellFormat(colWidths[4], 6, fmt.Sprintf("R$ %.2f", entry.AcceptedValue), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
		}

		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(226, 239, 218)
		pdf.CellFormat(colWidths[0]+colWidths[1]+colWidths[2]+colWidths[3], 7,
			"Total - "+report.TechnicianName, "1", 0, "R", true, 0, "")
		pdf.CellFormat(colWidths[4], 7, fmt.Sprintf("R$ %.2f", report.TotalAmount), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)
		pdf.Ln(4)

		grandTotal += report.TotalAmount
	}

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(190, 9, "TOTAL GERAL", "1", 0, "R", true, 0, "")
	pdf.CellFormat(40, 9, fmt.Sprintf("R$ %.2f", grandTotal), "1", 0, "R", true, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.SetFont("Helvetica", "I", 7)
	pdf.Ln(15)
	pdf.CellFormat(0, 5, fmt.Sprintf("Gerado em %s", time.Now().Format("02/01/2006 15:04:05")), "", 0, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar PDF"})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=relatorio_corte_"+time.Now().Format("20060102_150405")+".pdf")

	return c.Send(buf.Bytes())
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
