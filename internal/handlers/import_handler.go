package handlers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/repositories"
	"github.com/xuri/excelize/v2"
)

type ImportHandler struct {
	clientRepo     repositories.ClientRepository
	ticketRepo     repositories.TicketRepository
	categoryRepo   repositories.CategoryRepository
	technicianRepo repositories.TechnicianRepository
}

func NewImportHandler(
	clientRepo repositories.ClientRepository,
	ticketRepo repositories.TicketRepository,
	categoryRepo repositories.CategoryRepository,
	technicianRepo repositories.TechnicianRepository,
) *ImportHandler {
	return &ImportHandler{
		clientRepo:     clientRepo,
		ticketRepo:     ticketRepo,
		categoryRepo:   categoryRepo,
		technicianRepo: technicianRepo,
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
		"Referencia Externa", "Codigo Loja", "Nome Loja",
		"Rua", "Numero", "Cidade", "Estado", "CEP",
		"Descricao do Erro*", "Contato", "Telefone Contato",
		"Prioridade", "Categoria",
	}

	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 25,
		"D": 35, "E": 10, "F": 20, "G": 8, "H": 12,
		"I": 50, "J": 25, "K": 18,
		"L": 12, "M": 20,
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
		"Ex: RITM6261364", "Ex: 7962", "Ex: TIMON",
		"Ex: AV PRES MEDICI", "Ex: 268", "Ex: TIMON", "Ex: MA", "Ex: 65631-391",
		"Obrigatorio", "Ex: George Franklin", "Ex: 11999999999",
		"BAIXA/NORMAL/ALTA/URGENTE", "Nome da categoria",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	exampleData := []string{
		"RITM6261364", "7962", "TIMON",
		"AV PRES MEDICI", "268", "TIMON", "MA", "65631-391",
		"Cabo de rede do Caixa 5 com problemas", "George Franklin", "86999999999",
		"NORMAL", "Infraestrutura",
	}

	for i, value := range exampleData {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheetName, cell, value)
		f.SetCellStyle(sheetName, cell, cell, exampleStyle)
	}

	exampleData2 := []string{
		"RITM6261400", "8100", "TERESINA",
		"AV FREI SERAFIM", "1500", "TERESINA", "PI", "64001-020",
		"Impressora nao imprime em rede", "Maria Silva", "86988888888",
		"ALTA", "Impressoras",
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

	f.SetCellValue(instructionsSheet, "A1", "INSTRUCOES PARA IMPORTACAO DE CHAMADOS")
	f.SetCellValue(instructionsSheet, "A3", "1. Preencha os dados na aba 'Chamados'")
	f.SetCellValue(instructionsSheet, "A4", "2. O campo 'Descricao do Erro' e obrigatorio")
	f.SetCellValue(instructionsSheet, "A5", "3. Para Prioridade, use: BAIXA, NORMAL, ALTA ou URGENTE")
	f.SetCellValue(instructionsSheet, "A6", "4. A Categoria deve corresponder a uma categoria existente")
	f.SetCellValue(instructionsSheet, "A7", "5. Apague as linhas de exemplo antes de importar")
	f.SetCellValue(instructionsSheet, "A9", "CAMPOS:")
	f.SetCellValue(instructionsSheet, "A10", "- Referencia Externa: Numero do chamado do cliente (ex: RITM6261364)")
	f.SetCellValue(instructionsSheet, "A11", "- Codigo Loja: Codigo identificador da loja")
	f.SetCellValue(instructionsSheet, "A12", "- Nome Loja: Nome ou cidade da loja")
	f.SetCellValue(instructionsSheet, "A13", "- Endereco: Rua, Numero, Cidade, Estado, CEP do local de atendimento")
	f.SetCellValue(instructionsSheet, "A14", "- Descricao do Erro*: Descricao detalhada do problema (OBRIGATORIO)")
	f.SetCellValue(instructionsSheet, "A15", "- Contato: Nome da pessoa para procurar no local")
	f.SetCellValue(instructionsSheet, "A16", "- Telefone Contato: Telefone do contato")
	f.SetCellValue(instructionsSheet, "A17", "- Prioridade: BAIXA, NORMAL, ALTA ou URGENTE (padrao: NORMAL)")
	f.SetCellValue(instructionsSheet, "A18", "- Categoria: Nome exato da categoria")

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

		externalRef := ""
		storeCode := ""
		storeName := ""
		street := ""
		number := ""
		city := ""
		state := ""
		zipCode := ""
		errorDesc := ""
		contactName := ""
		contactPhone := ""
		priority := "NORMAL"
		categoryName := ""

		if len(row) > 0 {
			externalRef = strings.TrimSpace(row[0])
		}
		if len(row) > 1 {
			storeCode = strings.TrimSpace(row[1])
		}
		if len(row) > 2 {
			storeName = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			street = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			number = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			city = strings.TrimSpace(row[5])
		}
		if len(row) > 6 {
			state = strings.TrimSpace(row[6])
		}
		if len(row) > 7 {
			zipCode = strings.TrimSpace(row[7])
		}
		if len(row) > 8 {
			errorDesc = strings.TrimSpace(row[8])
		}
		if len(row) > 9 {
			contactName = strings.TrimSpace(row[9])
		}
		if len(row) > 10 {
			contactPhone = strings.TrimSpace(row[10])
		}
		if len(row) > 11 && row[11] != "" {
			priority = strings.ToUpper(strings.TrimSpace(row[11]))
		}
		if len(row) > 12 {
			categoryName = strings.TrimSpace(row[12])
		}

		if errorDesc == "" {
			errCount++
			errorDetails = append(errorDetails, fmt.Sprintf("Linha %d: Descricao do erro e obrigatoria", i+1))
			continue
		}

		validPriorities := map[string]bool{"BAIXA": true, "NORMAL": true, "ALTA": true, "URGENTE": true}
		if !validPriorities[priority] {
			priority = "NORMAL"
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
			ExternalReference:   externalRef,
			StoreCode:           storeCode,
			StoreName:           storeName,
			ServiceStreet:       street,
			ServiceNumber:       number,
			ServiceCity:         city,
			ServiceState:        state,
			ServiceZipCode:      zipCode,
			ErrorDescription:    errorDesc,
			ContactName:         contactName,
			ContactPhone:        contactPhone,
			Priority:            models.TicketPriority(priority),
			Status:              models.TicketStatusOpen,
		}

		if categoryID != "" {
			ticket.CategoryID = &categoryID
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
		"message":    fmt.Sprintf("Importacao concluida: %d chamados criados, %d erros", created, errCount),
		"imported":   created,
		"errorCount": errCount,
		"errors":     errorDetails,
	})
}

func (h *ImportHandler) DownloadTechnicianTemplate(c *fiber.Ctx) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Tecnicos"
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

	headers := []string{
		"Nome*", "Tipo", "Emails", "Telefones",
		"Valor Minimo", "Observacao", "CPF", "CNPJ",
		"Banco", "Agencia", "Conta", "Tipo Conta", "Titular", "Chave Pix",
		"Rua", "Numero", "Bairro", "Cidade", "Estado", "CEP",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10, Color: "666666"},
		Alignment: &excelize.Alignment{WrapText: true},
	})

	instructions := []string{
		"Obrigatorio", "PJ ou PF", "Separar por ;", "Separar por ;",
		"Ex: 150.00", "", "Apenas numeros", "Apenas numeros",
		"", "", "", "CORRENTE ou POUPANCA", "", "",
		"", "", "", "", "Ex: SP", "Ex: 01310-100",
	}

	for i, inst := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, inst)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	example := []string{
		"Joao Silva", "PF", "joao@email.com; joao2@email.com", "11999998888; 11988887777",
		"150.00", "Tecnico experiente", "12345678900", "",
		"Banco do Brasil", "1234", "12345-6", "CORRENTE", "Joao Silva", "joao@email.com",
		"Rua das Flores", "100", "Centro", "Sao Paulo", "SP", "01310-100",
	}

	for i, val := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheetName, cell, val)
	}

	colWidths := []float64{25, 8, 35, 30, 12, 25, 15, 18, 20, 10, 12, 15, 20, 25, 25, 10, 15, 20, 8, 12}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar template"})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=template_tecnicos.xlsx")
	return c.Send(buf.Bytes())
}

func (h *ImportHandler) ImportTechnicians(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Arquivo nao enviado"})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao abrir arquivo"})
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Arquivo Excel invalido"})
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Planilha vazia"})
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao ler linhas"})
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

		name := ""
		techType := "PF"
		emailsStr := ""
		phonesStr := ""
		minValue := 0.0
		observation := ""
		cpf := ""
		cnpj := ""
		bankName := ""
		agency := ""
		accountNumber := ""
		accountType := ""
		accountHolder := ""
		pixKey := ""
		street := ""
		number := ""
		neighborhood := ""
		city := ""
		state := ""
		zipCode := ""

		if len(row) > 0 {
			name = strings.TrimSpace(row[0])
		}
		if len(row) > 1 {
			techType = strings.ToUpper(strings.TrimSpace(row[1]))
		}
		if len(row) > 2 {
			emailsStr = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			phonesStr = strings.TrimSpace(row[3])
		}
		if len(row) > 4 && row[4] != "" {
			fmt.Sscanf(strings.Replace(row[4], ",", ".", -1), "%f", &minValue)
		}
		if len(row) > 5 {
			observation = strings.TrimSpace(row[5])
		}
		if len(row) > 6 {
			cpf = strings.TrimSpace(row[6])
		}
		if len(row) > 7 {
			cnpj = strings.TrimSpace(row[7])
		}
		if len(row) > 8 {
			bankName = strings.TrimSpace(row[8])
		}
		if len(row) > 9 {
			agency = strings.TrimSpace(row[9])
		}
		if len(row) > 10 {
			accountNumber = strings.TrimSpace(row[10])
		}
		if len(row) > 11 {
			accountType = strings.ToUpper(strings.TrimSpace(row[11]))
		}
		if len(row) > 12 {
			accountHolder = strings.TrimSpace(row[12])
		}
		if len(row) > 13 {
			pixKey = strings.TrimSpace(row[13])
		}
		if len(row) > 14 {
			street = strings.TrimSpace(row[14])
		}
		if len(row) > 15 {
			number = strings.TrimSpace(row[15])
		}
		if len(row) > 16 {
			neighborhood = strings.TrimSpace(row[16])
		}
		if len(row) > 17 {
			city = strings.TrimSpace(row[17])
		}
		if len(row) > 18 {
			state = strings.TrimSpace(row[18])
		}
		if len(row) > 19 {
			zipCode = strings.TrimSpace(row[19])
		}

		if name == "" {
			errCount++
			errorDetails = append(errorDetails, fmt.Sprintf("Linha %d: Nome e obrigatorio", i+1))
			continue
		}

		if techType != "PJ" && techType != "PF" {
			techType = "PF"
		}

		var emails models.EmailArray
		if emailsStr != "" {
			for _, e := range strings.Split(emailsStr, ";") {
				e = strings.TrimSpace(e)
				if e != "" {
					emails = append(emails, models.EmailEntry{Email: e, Type: "principal"})
				}
			}
		}

		var phones models.PhoneArray
		if phonesStr != "" {
			for _, p := range strings.Split(phonesStr, ";") {
				p = strings.TrimSpace(p)
				if p != "" {
					phones = append(phones, models.PhoneEntry{Number: p, Type: "principal"})
				}
			}
		}

		minValueStr := ""
		if minValue > 0 {
			minValueStr = fmt.Sprintf("%.2f", minValue)
		}

		tech := &models.Technician{
			FullName:      name,
			Type:          techType,
			Status:        "ATIVO",
			Emails:        emails,
			Phones:        phones,
			MinCallValue:  minValueStr,
			Observation:   observation,
			CPF:           cpf,
			CNPJ:          cnpj,
			BankName:      bankName,
			Agency:        agency,
			AccountNumber: accountNumber,
			AccountType:   accountType,
			AccountHolder: accountHolder,
			PixKey:        pixKey,
			Street:        street,
			Number:        number,
			Neighborhood:  neighborhood,
			City:          city,
			State:         state,
			ZipCode:       zipCode,
		}

		if err := h.technicianRepo.Create(tech); err != nil {
			errCount++
			errorDetails = append(errorDetails, fmt.Sprintf("Linha %d: %s", i+1, err.Error()))
			continue
		}

		created++
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    fmt.Sprintf("Importacao concluida: %d tecnicos criados, %d erros", created, errCount),
		"imported":   created,
		"errorCount": errCount,
		"errors":     errorDetails,
	})
}
