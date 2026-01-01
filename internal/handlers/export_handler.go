package handlers

import (
	"encoding/csv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shigake/tech-iq-back/internal/repositories"
)

type ExportHandler struct {
	clientRepo     repositories.ClientRepository
	technicianRepo repositories.TechnicianRepository
	ticketRepo     repositories.TicketRepository
}

func NewExportHandler(clientRepo repositories.ClientRepository, technicianRepo repositories.TechnicianRepository, ticketRepo repositories.TicketRepository) *ExportHandler {
	return &ExportHandler{
		clientRepo:     clientRepo,
		technicianRepo: technicianRepo,
		ticketRepo:     ticketRepo,
	}
}

// ExportClients exports clients data as CSV
func (h *ExportHandler) ExportClients(c *fiber.Ctx) error {
	// Get all clients with large page size
	clients, _, err := h.clientRepo.GetAll(0, 10000)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar clientes",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=clientes_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Nome Completo", "CPF", "CNPJ", "Email", "Telefone", "Rua", "Número", "Bairro", "Cidade", "Estado", "CEP", "Data de Criação"}
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

// ExportTechnicians exports technicians data as CSV
func (h *ExportHandler) ExportTechnicians(c *fiber.Ctx) error {
	// Get all technicians with large page size
	technicians, _, err := h.technicianRepo.FindAll(0, 10000)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar técnicos",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=tecnicos_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{"ID", "Nome", "CPF", "CNPJ", "Status", "Tipo", "Cidade", "Estado", "Data de Criação"}
	writer.Write(header)

	// Write data
	for _, tech := range technicians {
		// Get first email and phone if available
		emails := ""
		if len(tech.Emails) > 0 {
			emails = tech.Emails[0].Email
		}

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
		_ = emails // unused for now
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
	// Get all tickets with large page size
	tickets, _, err := h.ticketRepo.FindAll(0, 10000)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Erro ao buscar tickets",
			"error":   err.Error(),
		})
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=tickets_"+time.Now().Format("20060102_150405")+".csv")

	// Create CSV writer
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{
		"ID", "Número OS", "Descrição do Erro", "Status", "Prioridade",
		"Cliente", "Categoria", "Técnicos", "Data de Criação", "Data de Atualização",
	}
	writer.Write(header)

	// Write data
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
	writer.Write([]string{"=== EXPORTAÇÃO COMPLETA TECH-ERP ===", time.Now().Format("02/01/2006 15:04:05")})
	writer.Write([]string{""}) // Empty line

	// Export clients section
	writer.Write([]string{"=== CLIENTES ==="})
	clients, _, err := h.clientRepo.GetAll(0, 10000)
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
	writer.Write([]string{"=== TÉCNICOS ==="})
	technicians, _, err := h.technicianRepo.FindAll(0, 10000)
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
	writer.Write([]string{"=== TICKETS ==="})
	tickets, _, err := h.ticketRepo.FindAll(0, 10000)
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