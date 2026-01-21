package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/xuri/excelize/v2"
)

type ImportHandler struct {
	clientRepo   repositories.ClientRepository
	ticketRepo   repositories.TicketRepository
	categoryRepo repositories.CategoryRepository
}

func NewImportHandler(
	clientRepo repositories.ClientRepository,
	ticketRepo repositories.TicketRepository,
	categoryRepo repositories.CategoryRepository,
) *ImportHandler {
	return &ImportHandler{
		clientRepo:   clientRepo,
		ticketRepo:   ticketRepo,
		categoryRepo: categoryRepo,
	}
}

// DownloadTicketTemplate generates an Excel template for importing tickets
func (h *ImportHandler) DownloadTicketTemplate(c *fiber.Ctx) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Chamados"
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

	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10, Color: "666666"},
		Alignment: &excelize.Alignment{WrapText: true},
	})

	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D0D0D0", Style: 1},
			{Type: "top", Color: "D0D0D0", Style: 1},
			{Type: "bottom", Color: "D0D0D0", Style: 1},
			{Type: "right", Color: "D0D0D0", Style: 1},
		},
	})

	headers := []string{
		"Descrição do Erro*", "Prioridade", "Email do Cliente", "Categoria",
		"Data Início", "Data Limite", "Marca", "Modelo", "Número de Série",
	}

	colWidths := map[string]float64{
		"A": 40, "B": 15, "C": 30, "D": 20,
		"E": 15, "F": 15, "G": 15, "H": 20, "I": 20,
	}

	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	instructions := []string{
		"Obrigatório", "BAIXA, NORMAL, ALTA ou URGENTE", "Email cadastrado", "Nome da categoria",
		"DD/MM/AAAA", "DD/MM/AAAA", "Ex: Dell, HP", "Ex: Latitude 5520", "Nº de série",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	exampleData := []string{
		"Computador não liga após atualização", "ALTA", "cliente@empresa.com", "Hardware",
		"20/01/2026", "25/01/2026", "Dell", "Latitude 5520", "ABC123XYZ",
	}

	for i, value := range exampleData {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheetName, cell, value)
		f.SetCellStyle(sheetName, cell, cell, exampleStyle)
	}

	exampleData2 := []string{
		"Impressora não imprime em rede", "NORMAL", "joao@cliente.com", "Impressoras",
		"21/01/2026", "28/01/2026", "HP", "LaserJet Pro", "HP12345",
	}

	for i, value := range exampleData2 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(sheetName, cell, value)
		f.SetCellStyle(sheetName, cell, cell, exampleStyle)
	}

	f.SetRowHeight(sheetName, 1, 25)
	f.SetRowHeight(sheetName, 2, 20)

	instructionsSheet := "Instruções"
	f.NewSheet(instructionsSheet)

	f.SetCellValue(instructionsSheet, "A1", "INSTRUÇÕES PARA IMPORTAÇÃO DE CHAMADOS")
	f.SetCellValue(instructionsSheet, "A3", "1. Preencha os dados na aba 'Chamados'")
	f.SetCellValue(instructionsSheet, "A4", "2. O campo 'Descrição do Erro' é obrigatório")
	f.SetCellValue(instructionsSheet, "A5", "3. Para Prioridade, use: BAIXA, NORMAL, ALTA ou URGENTE")
	f.SetCellValue(instructionsSheet, "A6", "4. O Email do Cliente deve estar cadastrado no sistema")
	f.SetCellValue(instructionsSheet, "A7", "5. A Categoria deve corresponder a uma categoria existente")
	f.SetCellValue(instructionsSheet, "A8", "6. Datas no formato DD/MM/AAAA")
	f.SetCellValue(instructionsSheet, "A9", "7. Apague as linhas de exemplo antes de importar")
	f.SetCellValue(instructionsSheet, "A11", "CAMPOS:")
	f.SetCellValue(instructionsSheet, "A12", "- Descrição do Erro*: Descrição detalhada do problema")
	f.SetCellValue(instructionsSheet, "A13", "- Prioridade: BAIXA, NORMAL, ALTA ou URGENTE (padrão: NORMAL)")
	f.SetCellValue(instructionsSheet, "A14", "- Email do Cliente: Email do cliente cadastrado")
	f.SetCellValue(instructionsSheet, "A15", "- Categoria: Nome exato da categoria")
	f.SetCellValue(instructionsSheet, "A16", "- Data Início: Data de início do chamado")
	f.SetCellValue(instructionsSheet, "A17", "- Data Limite: Data limite para resolução")
	f.SetCellValue(instructionsSheet, "A18", "- Marca/Modelo/Série: Informações do equipamento")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "4472C4"},
	})
	f.SetCellStyle(instructionsSheet, "A1", "A1", titleStyle)
	f.SetColWidth(instructionsSheet, "A", "A", 60)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao gerar template",
			"error":   err.Error(),
		})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=template_chamados.xlsx")

	return c.Send(buf.Bytes())
}

// ImportTickets imports tickets from an Excel file
func (h *ImportHandler) ImportTickets(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Arquivo não enviado",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao abrir arquivo",
		})
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Arquivo Excel inválido",
		})
	}
	defer f.Close()

	rows, err := f.GetRows("Chamados")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Aba 'Chamados' não encontrada",
		})
	}

	if len(rows) < 2 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Planilha vazia ou sem dados",
		})
	}

	var created, errCount int
	var errorDetails []string

	for i, row := range rows {
		if i < 2 {
			continue
		}

		if len(row) == 0 || (len(row) > 0 && strings.TrimSpace(row[0]) == "") {
			continue
		}

		errorDesc := ""
		priority := "NORMAL"
		clientEmail := ""
		categoryName := ""
		startDate := ""
		dueDate := ""
		brand := ""
		model := ""
		serialNumber := ""

		if len(row) > 0 {
			errorDesc = strings.TrimSpace(row[0])
		}
		if len(row) > 1 && row[1] != "" {
			priority = strings.ToUpper(strings.TrimSpace(row[1]))
		}
		if len(row) > 2 {
			clientEmail = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			categoryName = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			startDate = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			dueDate = strings.TrimSpace(row[5])
		}
		if len(row) > 6 {
			brand = strings.TrimSpace(row[6])
		}
		if len(row) > 7 {
			model = strings.TrimSpace(row[7])
		}
		if len(row) > 8 {
			serialNumber = strings.TrimSpace(row[8])
		}

		if errorDesc == "" {
			errCount++
			errorDetails = append(errorDetails, fmt.Sprintf("Linha %d: Descrição do erro é obrigatória", i+1))
			continue
		}

		validPriorities := map[string]bool{"BAIXA": true, "NORMAL": true, "ALTA": true, "URGENTE": true}
		if !validPriorities[priority] {
			priority = "NORMAL"
		}

		var clientID string
		if clientEmail != "" {
			if client, err := h.clientRepo.FindByEmail(clientEmail); err == nil {
				clientID = client.ID
			}
		}

		var categoryID string
		if categoryName != "" {
			categories, _ := h.categoryRepo.GetAll()
			for _, cat := range categories {
				if strings.EqualFold(cat.Name, categoryName) {
					categoryID = cat.ID
					break
				}
				for _, child := range cat.Children {
					if strings.EqualFold(child.Name, categoryName) {
						categoryID = child.ID
						break
					}
				}
			}
		}

		ticket := &models.Ticket{
			ErrorDescription: errorDesc,
			Priority:         models.TicketPriority(priority),
			Status:           models.TicketStatusOpen,
			ComputerBrand:    brand,
			ComputerModel:    model,
			SerialNumber:     serialNumber,
		}

		if clientID != "" {
			ticket.ClientID = &clientID
		}

		if categoryID != "" {
			ticket.CategoryID = &categoryID
		}

		if startDate != "" {
			if t, err := time.Parse("02/01/2006", startDate); err == nil {
				ticket.StartDate = &t
			}
		}

		if dueDate != "" {
			if t, err := time.Parse("02/01/2006", dueDate); err == nil {
				ticket.DueDate = &t
			}
		}

		if err := h.ticketRepo.Create(ticket); err != nil {
			errCount++
			errorDetails = append(errorDetails, fmt.Sprintf("Linha %d: %s", i+1, err.Error()))
			continue
		}

		created++
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    fmt.Sprintf("Importação concluída: %d chamados criados, %d erros", created, errCount),
		"imported":   created,
		"errorCount": errCount,
		"errors":     errorDetails,
	})
}
