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
		"Max 100 chars", "Max 50 chars", "Max 255 chars",
		"Max 255 chars", "Max 20 chars", "Max 100 chars", "2 chars (UF)", "Max 10 chars",
		"Obrigatorio", "Max 255 chars", "Max 50 chars",
		"BAIXA/NORMAL/ALTA/URGENTE", "Nome da categoria",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	f.SetRowHeight(sheetName, 1, 25)
	f.SetRowHeight(sheetName, 2, 20)

	maxRows := 1000
	dvPriority := excelize.NewDataValidation(true)
	dvPriority.Sqref = fmt.Sprintf("L3:L%d", maxRows)
	dvPriority.SetDropList([]string{"BAIXA", "NORMAL", "ALTA", "URGENTE"})
	dvPriority.SetError(excelize.DataValidationErrorStyleStop, "Valor invalido", "Use: BAIXA, NORMAL, ALTA ou URGENTE")
	f.AddDataValidation(sheetName, dvPriority)

	dvState := excelize.NewDataValidation(true)
	dvState.Sqref = fmt.Sprintf("G3:G%d", maxRows)
	dvState.SetDropList([]string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"})
	dvState.SetError(excelize.DataValidationErrorStyleStop, "UF invalida", "Selecione uma UF valida")
	f.AddDataValidation(sheetName, dvState)

	fieldLimits := map[string]int{
		"A": 100, "B": 50, "C": 255,
		"D": 255, "E": 20, "F": 100, "H": 10,
		"J": 255, "K": 50, "M": 100,
	}
	for col, maxLen := range fieldLimits {
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s3:%s%d", col, col, maxRows)
		dv.Type = "textLength"
		dv.Operator = "lessThanOrEqual"
		dv.Formula1 = fmt.Sprintf("%d", maxLen)
		dv.SetError(excelize.DataValidationErrorStyleWarning, "Texto muito longo", fmt.Sprintf("Maximo de %d caracteres", maxLen))
		f.AddDataValidation(sheetName, dv)
	}

	instructionsSheet := "Instruções"
	f.NewSheet(instructionsSheet)

	f.SetCellValue(instructionsSheet, "A1", "INSTRUCOES PARA IMPORTACAO DE CHAMADOS")
	f.SetCellValue(instructionsSheet, "A3", "1. Preencha os dados na aba 'Chamados' a partir da linha 3")
	f.SetCellValue(instructionsSheet, "A4", "2. O campo 'Descricao do Erro' e obrigatorio")
	f.SetCellValue(instructionsSheet, "A5", "3. Para Prioridade, use: BAIXA, NORMAL, ALTA ou URGENTE")
	f.SetCellValue(instructionsSheet, "A6", "4. A Categoria deve corresponder a uma categoria existente")
	f.SetCellValue(instructionsSheet, "A7", "5. A linha 2 contem os limites de caracteres de cada campo")
	f.SetCellValue(instructionsSheet, "A9", "CAMPOS E LIMITES:")
	f.SetCellValue(instructionsSheet, "A10", "- Referencia Externa: Max 100 caracteres (ex: RITM6261364)")
	f.SetCellValue(instructionsSheet, "A11", "- Codigo Loja: Max 50 caracteres")
	f.SetCellValue(instructionsSheet, "A12", "- Nome Loja: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A13", "- Rua: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A14", "- Numero: Max 20 caracteres")
	f.SetCellValue(instructionsSheet, "A15", "- Cidade: Max 100 caracteres")
	f.SetCellValue(instructionsSheet, "A16", "- Estado: 2 caracteres (UF) - Selecione da lista")
	f.SetCellValue(instructionsSheet, "A17", "- CEP: Max 10 caracteres (ex: 65631-391)")
	f.SetCellValue(instructionsSheet, "A18", "- Descricao do Erro*: OBRIGATORIO - Sem limite")
	f.SetCellValue(instructionsSheet, "A19", "- Contato: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A20", "- Telefone Contato: Max 50 caracteres")
	f.SetCellValue(instructionsSheet, "A21", "- Prioridade: Selecione da lista (BAIXA/NORMAL/ALTA/URGENTE)")
	f.SetCellValue(instructionsSheet, "A22", "- Categoria: Max 100 caracteres")

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
			ExternalReference: externalRef,
			StoreCode:         storeCode,
			StoreName:         storeName,
			ServiceStreet:     street,
			ServiceNumber:     number,
			ServiceCity:       city,
			ServiceState:      state,
			ServiceZipCode:    zipCode,
			ErrorDescription:  errorDesc,
			ContactName:       contactName,
			ContactPhone:      contactPhone,
			Priority:          models.TicketPriority(priority),
			Status:            models.TicketStatusOpen,
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

	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10, Color: "666666"},
		Alignment: &excelize.Alignment{WrapText: true},
	})

	headers := []string{
		"Nome*", "Tipo", "Emails", "Telefones",
		"Valor Minimo", "Observacao", "CPF", "CNPJ",
		"Banco", "Agencia", "Conta", "Tipo Conta", "Titular", "Chave Pix",
		"Rua", "Numero", "Bairro", "Cidade", "Estado", "CEP",
	}

	colWidths := map[string]float64{
		"A": 25, "B": 8, "C": 35, "D": 30,
		"E": 12, "F": 25, "G": 15, "H": 18,
		"I": 20, "J": 10, "K": 12, "L": 15, "M": 20, "N": 25,
		"O": 25, "P": 10, "Q": 15, "R": 20, "S": 8, "T": 12,
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
		"Max 255 chars", "PJ ou PF", "Separar por ;", "Separar por ;",
		"Max 50 chars", "", "Max 14 chars", "Max 18 chars",
		"Max 200 chars", "Max 50 chars", "Max 50 chars", "CORRENTE/POUPANCA", "Max 255 chars", "Max 255 chars",
		"Max 255 chars", "Max 20 chars", "Max 100 chars", "Max 100 chars", "Max 50 chars", "Max 10 chars",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	f.SetRowHeight(sheetName, 1, 25)
	f.SetRowHeight(sheetName, 2, 20)

	maxRows := 1000

	dvType := excelize.NewDataValidation(true)
	dvType.Sqref = fmt.Sprintf("B3:B%d", maxRows)
	dvType.SetDropList([]string{"PF", "PJ"})
	dvType.SetError(excelize.DataValidationErrorStyleStop, "Tipo invalido", "Use: PF ou PJ")
	f.AddDataValidation(sheetName, dvType)

	dvAccountType := excelize.NewDataValidation(true)
	dvAccountType.Sqref = fmt.Sprintf("L3:L%d", maxRows)
	dvAccountType.SetDropList([]string{"CORRENTE", "POUPANCA"})
	dvAccountType.SetError(excelize.DataValidationErrorStyleStop, "Tipo invalido", "Use: CORRENTE ou POUPANCA")
	f.AddDataValidation(sheetName, dvAccountType)

	dvState := excelize.NewDataValidation(true)
	dvState.Sqref = fmt.Sprintf("S3:S%d", maxRows)
	dvState.SetDropList([]string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"})
	dvState.SetError(excelize.DataValidationErrorStyleStop, "UF invalida", "Selecione uma UF valida")
	f.AddDataValidation(sheetName, dvState)

	fieldLimits := map[string]int{
		"A": 255, "E": 50, "G": 14, "H": 18,
		"I": 200, "J": 50, "K": 50, "M": 255, "N": 255,
		"O": 255, "P": 20, "Q": 100, "R": 100, "T": 10,
	}
	for col, maxLen := range fieldLimits {
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s3:%s%d", col, col, maxRows)
		dv.Type = "textLength"
		dv.Operator = "lessThanOrEqual"
		dv.Formula1 = fmt.Sprintf("%d", maxLen)
		dv.SetError(excelize.DataValidationErrorStyleWarning, "Texto muito longo", fmt.Sprintf("Maximo de %d caracteres", maxLen))
		f.AddDataValidation(sheetName, dv)
	}

	dvCPF := excelize.NewDataValidation(true)
	dvCPF.Sqref = fmt.Sprintf("G3:G%d", maxRows)
	dvCPF.Type = "textLength"
	dvCPF.Operator = "between"
	dvCPF.Formula1 = "11"
	dvCPF.Formula2 = "14"
	dvCPF.SetError(excelize.DataValidationErrorStyleWarning, "CPF invalido", "CPF deve ter entre 11 e 14 caracteres")
	f.AddDataValidation(sheetName, dvCPF)

	dvCNPJ := excelize.NewDataValidation(true)
	dvCNPJ.Sqref = fmt.Sprintf("H3:H%d", maxRows)
	dvCNPJ.Type = "textLength"
	dvCNPJ.Operator = "between"
	dvCNPJ.Formula1 = "14"
	dvCNPJ.Formula2 = "18"
	dvCNPJ.SetError(excelize.DataValidationErrorStyleWarning, "CNPJ invalido", "CNPJ deve ter entre 14 e 18 caracteres")
	f.AddDataValidation(sheetName, dvCNPJ)

	instructionsSheet := "Instruções"
	f.NewSheet(instructionsSheet)

	f.SetCellValue(instructionsSheet, "A1", "INSTRUCOES PARA IMPORTACAO DE TECNICOS")
	f.SetCellValue(instructionsSheet, "A3", "1. Preencha os dados na aba 'Tecnicos' a partir da linha 3")
	f.SetCellValue(instructionsSheet, "A4", "2. O campo 'Nome' e obrigatorio (max 255 caracteres)")
	f.SetCellValue(instructionsSheet, "A5", "3. Para Tipo, selecione: PF ou PJ da lista")
	f.SetCellValue(instructionsSheet, "A6", "4. Para Tipo Conta, selecione: CORRENTE ou POUPANCA da lista")
	f.SetCellValue(instructionsSheet, "A7", "5. A linha 2 contem os limites de caracteres de cada campo")
	f.SetCellValue(instructionsSheet, "A9", "CAMPOS E LIMITES:")
	f.SetCellValue(instructionsSheet, "A10", "- Nome*: Max 255 caracteres (OBRIGATORIO)")
	f.SetCellValue(instructionsSheet, "A11", "- Tipo: Selecione PF ou PJ da lista")
	f.SetCellValue(instructionsSheet, "A12", "- Emails: Lista separada por ponto e virgula (;)")
	f.SetCellValue(instructionsSheet, "A13", "- Telefones: Lista separada por ponto e virgula (;)")
	f.SetCellValue(instructionsSheet, "A14", "- Valor Minimo: Max 50 caracteres (ex: 150.00)")
	f.SetCellValue(instructionsSheet, "A15", "- Observacao: Sem limite de caracteres")
	f.SetCellValue(instructionsSheet, "A16", "- CPF: Entre 11 e 14 caracteres (apenas numeros ou formatado)")
	f.SetCellValue(instructionsSheet, "A17", "- CNPJ: Entre 14 e 18 caracteres (apenas numeros ou formatado)")
	f.SetCellValue(instructionsSheet, "A18", "- Banco: Max 200 caracteres")
	f.SetCellValue(instructionsSheet, "A19", "- Agencia: Max 50 caracteres")
	f.SetCellValue(instructionsSheet, "A20", "- Conta: Max 50 caracteres")
	f.SetCellValue(instructionsSheet, "A21", "- Tipo Conta: Selecione CORRENTE ou POUPANCA da lista")
	f.SetCellValue(instructionsSheet, "A22", "- Titular: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A23", "- Chave Pix: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A24", "- Rua: Max 255 caracteres")
	f.SetCellValue(instructionsSheet, "A25", "- Numero: Max 20 caracteres")
	f.SetCellValue(instructionsSheet, "A26", "- Bairro: Max 100 caracteres")
	f.SetCellValue(instructionsSheet, "A27", "- Cidade: Max 100 caracteres")
	f.SetCellValue(instructionsSheet, "A28", "- Estado: Selecione a UF da lista")
	f.SetCellValue(instructionsSheet, "A29", "- CEP: Max 10 caracteres (ex: 01310-100)")

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
