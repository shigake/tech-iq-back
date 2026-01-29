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
		Protection: &excelize.Protection{Locked: true},
	})

	requiredHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C65911"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Protection: &excelize.Protection{Locked: true},
	})

	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:       &excelize.Font{Italic: true, Size: 9, Color: "666666"},
		Alignment:  &excelize.Alignment{WrapText: true, Horizontal: "center"},
		Fill:       excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Protection: &excelize.Protection{Locked: true},
	})

	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "808080", Italic: true},
		Alignment: &excelize.Alignment{WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFFDE7"}, Pattern: 1},
	})

	cepStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "@"; return &s }(),
		Alignment:    &excelize.Alignment{Horizontal: "center"},
	})

	phoneStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "@"; return &s }(),
		Alignment:    &excelize.Alignment{Horizontal: "left"},
	})

	headers := []string{
		"Referencia Externa", "Codigo Loja", "Nome Loja",
		"Rua", "Numero", "Cidade", "Estado", "CEP",
		"Descricao do Erro*", "Contato", "Telefone Contato",
		"Prioridade", "Categoria",
	}

	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 25,
		"D": 35, "E": 10, "F": 20, "G": 8, "H": 14,
		"I": 50, "J": 25, "K": 18,
		"L": 15, "M": 20,
	}

	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	requiredCols := map[int]bool{8: true}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		if requiredCols[i] {
			f.SetCellStyle(sheetName, cell, cell, requiredHeaderStyle)
		} else {
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
	}

	instructions := []string{
		"Max 100 | Ex: RITM6261364", "Max 50", "Max 255",
		"Max 255", "Max 20", "Max 100", "Selecione UF", "Formato: 00000-000",
		"*OBRIGATORIO", "Max 255", "(00) 00000-0000",
		"Selecione", "Max 100",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	examples := []string{
		"RITM6261364", "LJ001", "Loja Centro",
		"Av. Paulista", "1000", "Sao Paulo", "SP", "01310-100",
		"Computador nao liga apos queda de energia", "Maria Silva", "(11) 99999-8888",
		"NORMAL", "Hardware",
	}

	for i, example := range examples {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheetName, cell, example)
		f.SetCellStyle(sheetName, cell, cell, exampleStyle)
	}

	f.SetRowHeight(sheetName, 1, 30)
	f.SetRowHeight(sheetName, 2, 25)
	f.SetRowHeight(sheetName, 3, 22)

	maxRows := 1000

	for row := 4; row <= maxRows; row++ {
		cepCell, _ := excelize.CoordinatesToCellName(8, row)
		f.SetCellStyle(sheetName, cepCell, cepCell, cepStyle)
		phoneCell, _ := excelize.CoordinatesToCellName(11, row)
		f.SetCellStyle(sheetName, phoneCell, phoneCell, phoneStyle)
	}

	dvPriority := excelize.NewDataValidation(true)
	dvPriority.Sqref = fmt.Sprintf("L4:L%d", maxRows)
	dvPriority.SetDropList([]string{"BAIXA", "NORMAL", "ALTA", "URGENTE"})
	dvPriority.SetError(excelize.DataValidationErrorStyleStop, "Valor invalido", "Selecione: BAIXA, NORMAL, ALTA ou URGENTE")
	dvPriority.SetInput("Prioridade", "Selecione a prioridade do chamado")
	f.AddDataValidation(sheetName, dvPriority)

	dvState := excelize.NewDataValidation(true)
	dvState.Sqref = fmt.Sprintf("G4:G%d", maxRows)
	dvState.SetDropList([]string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"})
	dvState.SetError(excelize.DataValidationErrorStyleStop, "UF invalida", "Selecione uma UF valida da lista")
	dvState.SetInput("Estado", "Selecione a UF")
	f.AddDataValidation(sheetName, dvState)

	fieldLimits := map[string]int{
		"A": 100, "B": 50, "C": 255,
		"D": 255, "E": 20, "F": 100,
		"J": 255, "K": 50, "M": 100,
	}
	for col, maxLen := range fieldLimits {
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s4:%s%d", col, col, maxRows)
		dv.Type = "textLength"
		dv.Operator = "lessThanOrEqual"
		dv.Formula1 = fmt.Sprintf("%d", maxLen)
		dv.SetError(excelize.DataValidationErrorStyleWarning, "Texto muito longo", fmt.Sprintf("Maximo permitido: %d caracteres", maxLen))
		f.AddDataValidation(sheetName, dv)
	}

	dvCEP := excelize.NewDataValidation(true)
	dvCEP.Sqref = fmt.Sprintf("H4:H%d", maxRows)
	dvCEP.Type = "textLength"
	dvCEP.Operator = "between"
	dvCEP.Formula1 = "8"
	dvCEP.Formula2 = "10"
	dvCEP.SetError(excelize.DataValidationErrorStyleWarning, "CEP invalido", "Use o formato: 00000-000 ou 00000000")
	dvCEP.SetInput("CEP", "Formato: 00000-000")
	f.AddDataValidation(sheetName, dvCEP)

	dvPhone := excelize.NewDataValidation(true)
	dvPhone.Sqref = fmt.Sprintf("K4:K%d", maxRows)
	dvPhone.Type = "textLength"
	dvPhone.Operator = "between"
	dvPhone.Formula1 = "10"
	dvPhone.Formula2 = "50"
	dvPhone.SetError(excelize.DataValidationErrorStyleWarning, "Telefone invalido", "Use formato: (00) 00000-0000 ou similar")
	dvPhone.SetInput("Telefone", "Formato: (00) 00000-0000")
	f.AddDataValidation(sheetName, dvPhone)

	instructionsSheet := "Instrucoes"
	f.NewSheet(instructionsSheet)

	f.SetCellValue(instructionsSheet, "A1", "INSTRUCOES PARA IMPORTACAO DE CHAMADOS")
	f.SetCellValue(instructionsSheet, "A3", "COMO USAR:")
	f.SetCellValue(instructionsSheet, "A4", "1. Preencha os dados na aba 'Chamados' a partir da linha 4 (linha 3 e exemplo)")
	f.SetCellValue(instructionsSheet, "A5", "2. O campo com cabecalho LARANJA e obrigatorio")
	f.SetCellValue(instructionsSheet, "A6", "3. A linha 2 mostra os limites de cada campo")
	f.SetCellValue(instructionsSheet, "A7", "4. A linha 3 mostra um exemplo de preenchimento")
	f.SetCellValue(instructionsSheet, "A8", "5. Campos com dropdown devem ser selecionados da lista")
	f.SetCellValue(instructionsSheet, "A10", "CAMPOS E RESTRICOES (baseados no banco de dados):")
	f.SetCellValue(instructionsSheet, "A12", "| Campo                | Tamanho Max | Formato/Mascara        | Obrigatorio |")
	f.SetCellValue(instructionsSheet, "A13", "|----------------------|-------------|------------------------|-------------|")
	f.SetCellValue(instructionsSheet, "A14", "| Referencia Externa   | 100         | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A15", "| Codigo Loja          | 50          | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A16", "| Nome Loja            | 255         | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A17", "| Rua                  | 255         | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A18", "| Numero               | 20          | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A19", "| Cidade               | 100         | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A20", "| Estado               | 2           | UF (SP, RJ, MG...)     | Nao         |")
	f.SetCellValue(instructionsSheet, "A21", "| CEP                  | 10          | 00000-000 ou 00000000  | Nao         |")
	f.SetCellValue(instructionsSheet, "A22", "| Descricao do Erro    | Ilimitado   | Texto livre            | SIM         |")
	f.SetCellValue(instructionsSheet, "A23", "| Contato              | 255         | Texto livre            | Nao         |")
	f.SetCellValue(instructionsSheet, "A24", "| Telefone Contato     | 50          | (00) 00000-0000        | Nao         |")
	f.SetCellValue(instructionsSheet, "A25", "| Prioridade           | -           | BAIXA/NORMAL/ALTA/URG  | Nao         |")
	f.SetCellValue(instructionsSheet, "A26", "| Categoria            | 100         | Nome existente         | Nao         |")
	f.SetCellValue(instructionsSheet, "A28", "DICAS:")
	f.SetCellValue(instructionsSheet, "A29", "- CPF/CNPJ: aceita com ou sem pontuacao")
	f.SetCellValue(instructionsSheet, "A30", "- CEP: aceita 01310100 ou 01310-100")
	f.SetCellValue(instructionsSheet, "A31", "- Telefone: aceita varios formatos (11999998888, (11) 99999-8888)")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "4472C4"},
	})
	f.SetCellStyle(instructionsSheet, "A1", "A1", titleStyle)

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "333333"},
	})
	f.SetCellStyle(instructionsSheet, "A3", "A3", subtitleStyle)
	f.SetCellStyle(instructionsSheet, "A10", "A10", subtitleStyle)
	f.SetCellStyle(instructionsSheet, "A28", "A28", subtitleStyle)

	tableStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Name: "Consolas", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	for i := 12; i <= 26; i++ {
		cell := fmt.Sprintf("A%d", i)
		f.SetCellStyle(instructionsSheet, cell, cell, tableStyle)
	}

	f.SetColWidth(instructionsSheet, "A", "A", 80)

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

	if len(rows) < 4 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Planilha vazia ou sem dados (preencha a partir da linha 4)",
		})
	}

	var created, errCount int
	var errorDetails []string

	for i, row := range rows {
		if i < 3 {
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
		Protection: &excelize.Protection{Locked: true},
	})

	requiredHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C65911"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Protection: &excelize.Protection{Locked: true},
	})

	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:       &excelize.Font{Italic: true, Size: 9, Color: "666666"},
		Alignment:  &excelize.Alignment{WrapText: true, Horizontal: "center"},
		Fill:       excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Protection: &excelize.Protection{Locked: true},
	})

	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "808080", Italic: true},
		Alignment: &excelize.Alignment{WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFFDE7"}, Pattern: 1},
	})

	cpfStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "@"; return &s }(),
		Alignment:    &excelize.Alignment{Horizontal: "center"},
	})

	cnpjStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "@"; return &s }(),
		Alignment:    &excelize.Alignment{Horizontal: "center"},
	})

	cepStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: func() *string { s := "@"; return &s }(),
		Alignment:    &excelize.Alignment{Horizontal: "center"},
	})

	headers := []string{
		"Nome*", "Tipo", "Emails", "Telefones",
		"Valor Minimo", "Observacao", "CPF", "CNPJ",
		"Banco", "Agencia", "Conta", "Tipo Conta", "Titular", "Chave Pix",
		"Rua", "Numero", "Bairro", "Cidade", "Estado", "CEP",
	}

	colWidths := map[string]float64{
		"A": 28, "B": 10, "C": 40, "D": 35,
		"E": 14, "F": 30, "G": 18, "H": 22,
		"I": 22, "J": 12, "K": 14, "L": 18, "M": 22, "N": 28,
		"O": 28, "P": 10, "Q": 18, "R": 22, "S": 10, "T": 14,
	}

	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	requiredCols := map[int]bool{0: true}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		if requiredCols[i] {
			f.SetCellStyle(sheetName, cell, cell, requiredHeaderStyle)
		} else {
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
	}

	instructions := []string{
		"*OBRIGATORIO Max 255", "Selecione", "Separar por ;", "(00)00000-0000; ...",
		"Ex: 150.00", "Texto livre", "000.000.000-00", "00.000.000/0000-00",
		"Max 200", "Max 50", "Max 50", "Selecione", "Max 255", "Max 255",
		"Max 255", "Max 20", "Max 100", "Max 100", "Selecione UF", "00000-000",
	}

	for i, instruction := range instructions {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, instruction)
		f.SetCellStyle(sheetName, cell, cell, instructionStyle)
	}

	examples := []string{
		"Joao da Silva", "PF", "joao@email.com; joao2@email.com", "(11) 99999-8888; (11) 3333-4444",
		"150.00", "Tecnico especializado em redes", "123.456.789-00", "12.345.678/0001-90",
		"Banco do Brasil", "1234", "12345-6", "CORRENTE", "Joao da Silva", "joao@email.com",
		"Rua das Flores", "100", "Centro", "Sao Paulo", "SP", "01310-100",
	}

	for i, example := range examples {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheetName, cell, example)
		f.SetCellStyle(sheetName, cell, cell, exampleStyle)
	}

	f.SetRowHeight(sheetName, 1, 30)
	f.SetRowHeight(sheetName, 2, 25)
	f.SetRowHeight(sheetName, 3, 22)

	maxRows := 1000

	for row := 4; row <= maxRows; row++ {
		cpfCell, _ := excelize.CoordinatesToCellName(7, row)
		f.SetCellStyle(sheetName, cpfCell, cpfCell, cpfStyle)
		cnpjCell, _ := excelize.CoordinatesToCellName(8, row)
		f.SetCellStyle(sheetName, cnpjCell, cnpjCell, cnpjStyle)
		cepCell, _ := excelize.CoordinatesToCellName(20, row)
		f.SetCellStyle(sheetName, cepCell, cepCell, cepStyle)
	}

	dvType := excelize.NewDataValidation(true)
	dvType.Sqref = fmt.Sprintf("B4:B%d", maxRows)
	dvType.SetDropList([]string{"PF", "PJ"})
	dvType.SetError(excelize.DataValidationErrorStyleStop, "Tipo invalido", "Selecione: PF ou PJ")
	dvType.SetInput("Tipo Pessoa", "PF = Pessoa Fisica, PJ = Pessoa Juridica")
	f.AddDataValidation(sheetName, dvType)

	dvAccountType := excelize.NewDataValidation(true)
	dvAccountType.Sqref = fmt.Sprintf("L4:L%d", maxRows)
	dvAccountType.SetDropList([]string{"CORRENTE", "POUPANCA"})
	dvAccountType.SetError(excelize.DataValidationErrorStyleStop, "Tipo invalido", "Selecione: CORRENTE ou POUPANCA")
	dvAccountType.SetInput("Tipo de Conta", "Selecione o tipo de conta bancaria")
	f.AddDataValidation(sheetName, dvAccountType)

	dvState := excelize.NewDataValidation(true)
	dvState.Sqref = fmt.Sprintf("S4:S%d", maxRows)
	dvState.SetDropList([]string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"})
	dvState.SetError(excelize.DataValidationErrorStyleStop, "UF invalida", "Selecione uma UF valida da lista")
	dvState.SetInput("Estado", "Selecione a UF")
	f.AddDataValidation(sheetName, dvState)

	fieldLimits := map[string]int{
		"A": 255, "E": 50,
		"I": 200, "J": 50, "K": 50, "M": 255, "N": 255,
		"O": 255, "P": 20, "Q": 100, "R": 100,
	}
	for col, maxLen := range fieldLimits {
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s4:%s%d", col, col, maxRows)
		dv.Type = "textLength"
		dv.Operator = "lessThanOrEqual"
		dv.Formula1 = fmt.Sprintf("%d", maxLen)
		dv.SetError(excelize.DataValidationErrorStyleWarning, "Texto muito longo", fmt.Sprintf("Maximo permitido: %d caracteres", maxLen))
		f.AddDataValidation(sheetName, dv)
	}

	dvCPF := excelize.NewDataValidation(true)
	dvCPF.Sqref = fmt.Sprintf("G4:G%d", maxRows)
	dvCPF.Type = "textLength"
	dvCPF.Operator = "between"
	dvCPF.Formula1 = "11"
	dvCPF.Formula2 = "14"
	dvCPF.SetError(excelize.DataValidationErrorStyleWarning, "CPF invalido", "CPF: 11 digitos (sem mascara) ou 14 (com mascara: 000.000.000-00)")
	dvCPF.SetInput("CPF", "Formato: 000.000.000-00 ou 00000000000")
	f.AddDataValidation(sheetName, dvCPF)

	dvCNPJ := excelize.NewDataValidation(true)
	dvCNPJ.Sqref = fmt.Sprintf("H4:H%d", maxRows)
	dvCNPJ.Type = "textLength"
	dvCNPJ.Operator = "between"
	dvCNPJ.Formula1 = "14"
	dvCNPJ.Formula2 = "18"
	dvCNPJ.SetError(excelize.DataValidationErrorStyleWarning, "CNPJ invalido", "CNPJ: 14 digitos (sem mascara) ou 18 (com mascara: 00.000.000/0000-00)")
	dvCNPJ.SetInput("CNPJ", "Formato: 00.000.000/0000-00 ou 00000000000000")
	f.AddDataValidation(sheetName, dvCNPJ)

	dvCEP := excelize.NewDataValidation(true)
	dvCEP.Sqref = fmt.Sprintf("T4:T%d", maxRows)
	dvCEP.Type = "textLength"
	dvCEP.Operator = "between"
	dvCEP.Formula1 = "8"
	dvCEP.Formula2 = "10"
	dvCEP.SetError(excelize.DataValidationErrorStyleWarning, "CEP invalido", "CEP: 8 digitos (sem mascara) ou 9-10 (com mascara: 00000-000)")
	dvCEP.SetInput("CEP", "Formato: 00000-000 ou 00000000")
	f.AddDataValidation(sheetName, dvCEP)

	instructionsSheet := "Instrucoes"
	f.NewSheet(instructionsSheet)

	f.SetCellValue(instructionsSheet, "A1", "INSTRUCOES PARA IMPORTACAO DE TECNICOS")
	f.SetCellValue(instructionsSheet, "A3", "COMO USAR:")
	f.SetCellValue(instructionsSheet, "A4", "1. Preencha os dados na aba 'Tecnicos' a partir da linha 4 (linha 3 e exemplo)")
	f.SetCellValue(instructionsSheet, "A5", "2. O campo com cabecalho LARANJA e obrigatorio")
	f.SetCellValue(instructionsSheet, "A6", "3. A linha 2 mostra os limites e formatos de cada campo")
	f.SetCellValue(instructionsSheet, "A7", "4. A linha 3 mostra um exemplo de preenchimento")
	f.SetCellValue(instructionsSheet, "A8", "5. Campos com dropdown devem ser selecionados da lista")
	f.SetCellValue(instructionsSheet, "A10", "CAMPOS E RESTRICOES (baseados no banco de dados):")
	f.SetCellValue(instructionsSheet, "A12", "| Campo           | Tamanho Max | Formato/Mascara                  | Obrigatorio |")
	f.SetCellValue(instructionsSheet, "A13", "|-----------------|-------------|----------------------------------|-------------|")
	f.SetCellValue(instructionsSheet, "A14", "| Nome            | 255         | Texto livre                      | SIM         |")
	f.SetCellValue(instructionsSheet, "A15", "| Tipo            | -           | PF ou PJ (dropdown)              | Nao         |")
	f.SetCellValue(instructionsSheet, "A16", "| Emails          | -           | Separar multiplos por ;          | Nao         |")
	f.SetCellValue(instructionsSheet, "A17", "| Telefones       | -           | (00) 00000-0000; Separar por ;   | Nao         |")
	f.SetCellValue(instructionsSheet, "A18", "| Valor Minimo    | 50          | Numero decimal (ex: 150.00)      | Nao         |")
	f.SetCellValue(instructionsSheet, "A19", "| Observacao      | Ilimitado   | Texto livre                      | Nao         |")
	f.SetCellValue(instructionsSheet, "A20", "| CPF             | 14          | 000.000.000-00 ou 00000000000    | Nao         |")
	f.SetCellValue(instructionsSheet, "A21", "| CNPJ            | 18          | 00.000.000/0000-00 ou 00000...   | Nao         |")
	f.SetCellValue(instructionsSheet, "A22", "| Banco           | 200         | Nome do banco                    | Nao         |")
	f.SetCellValue(instructionsSheet, "A23", "| Agencia         | 50          | Numero da agencia                | Nao         |")
	f.SetCellValue(instructionsSheet, "A24", "| Conta           | 50          | Numero da conta com digito       | Nao         |")
	f.SetCellValue(instructionsSheet, "A25", "| Tipo Conta      | -           | CORRENTE ou POUPANCA (dropdown)  | Nao         |")
	f.SetCellValue(instructionsSheet, "A26", "| Titular         | 255         | Nome do titular da conta         | Nao         |")
	f.SetCellValue(instructionsSheet, "A27", "| Chave Pix       | 255         | CPF, email, telefone ou aleatoria| Nao         |")
	f.SetCellValue(instructionsSheet, "A28", "| Rua             | 255         | Endereco do tecnico              | Nao         |")
	f.SetCellValue(instructionsSheet, "A29", "| Numero          | 20          | Numero do endereco               | Nao         |")
	f.SetCellValue(instructionsSheet, "A30", "| Bairro          | 100         | Bairro                           | Nao         |")
	f.SetCellValue(instructionsSheet, "A31", "| Cidade          | 100         | Cidade                           | Nao         |")
	f.SetCellValue(instructionsSheet, "A32", "| Estado          | 2           | UF (dropdown)                    | Nao         |")
	f.SetCellValue(instructionsSheet, "A33", "| CEP             | 10          | 00000-000 ou 00000000            | Nao         |")
	f.SetCellValue(instructionsSheet, "A35", "DICAS DE PREENCHIMENTO:")
	f.SetCellValue(instructionsSheet, "A36", "- CPF: aceita com mascara (123.456.789-00) ou sem (12345678900)")
	f.SetCellValue(instructionsSheet, "A37", "- CNPJ: aceita com mascara (12.345.678/0001-90) ou sem (12345678000190)")
	f.SetCellValue(instructionsSheet, "A38", "- CEP: aceita com mascara (01310-100) ou sem (01310100)")
	f.SetCellValue(instructionsSheet, "A39", "- Telefone: formato sugerido (11) 99999-8888, mas aceita outros")
	f.SetCellValue(instructionsSheet, "A40", "- Emails/Telefones: para multiplos valores, separe com ponto e virgula (;)")
	f.SetCellValue(instructionsSheet, "A41", "- Chave Pix: pode ser CPF, CNPJ, email, telefone ou chave aleatoria")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "4472C4"},
	})
	f.SetCellStyle(instructionsSheet, "A1", "A1", titleStyle)

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "333333"},
	})
	f.SetCellStyle(instructionsSheet, "A3", "A3", subtitleStyle)
	f.SetCellStyle(instructionsSheet, "A10", "A10", subtitleStyle)
	f.SetCellStyle(instructionsSheet, "A35", "A35", subtitleStyle)

	tableStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Name: "Consolas", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	for i := 12; i <= 33; i++ {
		cell := fmt.Sprintf("A%d", i)
		f.SetCellStyle(instructionsSheet, cell, cell, tableStyle)
	}

	f.SetColWidth(instructionsSheet, "A", "A", 80)

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

	if len(rows) < 4 {
		return c.Status(400).JSON(fiber.Map{"error": "Planilha vazia ou sem dados (preencha a partir da linha 4)"})
	}

	var created, errCount int
	var errorDetails []string

	for i, row := range rows {
		if i < 3 {
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
