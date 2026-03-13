package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shigake/tech-iq-back/internal/models"
	"github.com/shigake/tech-iq-back/internal/services"
	"gorm.io/gorm"
)

type TicketHandler struct {
	service            services.TicketService
	activityLogService services.ActivityLogService
	validate           *validator.Validate
	db                 *gorm.DB
}

func NewTicketHandler(service services.TicketService) *TicketHandler {
	return &TicketHandler{
		service:  service,
		validate: validator.New(),
	}
}

func NewTicketHandlerWithActivityLog(service services.TicketService, activityLogService services.ActivityLogService, db *gorm.DB) *TicketHandler {
	return &TicketHandler{
		service:            service,
		activityLogService: activityLogService,
		validate:           validator.New(),
		db:                 db,
	}
}

func (h *TicketHandler) logActivity(c *fiber.Ctx, action, resourceID, description string) {
	if h.activityLogService == nil {
		return
	}
	userID, _ := c.Locals("userId").(string)
	go h.activityLogService.LogAction(userID, action, "ticket", resourceID, description, c.IP(), c.Get("User-Agent"))
}

// GetAll returns paginated list of tickets with filters
func (h *TicketHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))

	// Parse filters
	filters := &models.TicketFilters{
		Status:       c.Query("status"),
		Priority:     c.Query("priority"),
		NodeID:       c.Query("nodeId"),
		ClientID:     c.Query("clientId"),
		CategoryID:   c.Query("categoryId"),
		TechnicianID: c.Query("technicianId"),
		Search:       c.Query("search"),
		DateFrom:     c.Query("dateFrom"),
		DateTo:       c.Query("dateTo"),
	}

	// Get user context for role-based filtering
	userID, _ := c.Locals("userId").(string)
	userRole, _ := c.Locals("userRole").(string)

	response, err := h.service.GetAllForUser(page, size, filters, userID, userRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tickets",
		})
	}

	return c.JSON(response)
}

// GetByID returns a ticket by ID
func (h *TicketHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	ticket, err := h.service.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	return c.JSON(ticket)
}

// Create creates a new ticket
func (h *TicketHandler) Create(c *fiber.Ctx) error {
	var req models.CreateTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	ticket, err := h.service.Create(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.logActivity(c, "create", ticket.ID, fmt.Sprintf("Ticket #%s criado", ticket.OSNumber))

	return c.Status(fiber.StatusCreated).JSON(ticket)
}

// Update updates a ticket
func (h *TicketHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	var req models.CreateTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	ticket, err := h.service.Update(id, &req)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	h.logActivity(c, "update", ticket.ID, fmt.Sprintf("Ticket #%s atualizado", ticket.OSNumber))

	return c.JSON(ticket)
}

// Delete deletes a ticket
func (h *TicketHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	ticket, _ := h.service.GetByID(id)

	if err := h.service.Delete(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	if ticket != nil {
		h.logActivity(c, "delete", id, fmt.Sprintf("Ticket #%s removido", ticket.OSNumber))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateStatus updates the ticket status
func (h *TicketHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	var req models.UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateStatus(id, req.Status); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.logActivity(c, "update", id, fmt.Sprintf("Status do ticket alterado para %s", req.Status))

	return c.JSON(fiber.Map{"message": "Status updated successfully"})
}

// AssignTechnician assigns technicians to a ticket
func (h *TicketHandler) AssignTechnician(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	var req models.AssignTechnicianRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.AssignTechnicians(id, req.TechnicianIDs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.logActivity(c, "update", id, "Técnicos atribuídos ao ticket")

	return c.JSON(fiber.Map{"message": "Technicians assigned successfully"})
}

func (h *TicketHandler) SignTicket(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	var req models.SignTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TechnicianSignature == "" || req.ClientSignature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Both technician and client signatures are required",
		})
	}

	ticket, err := h.service.SignTicket(id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.logActivity(c, "sign", ticket.ID, fmt.Sprintf("Ticket #%s assinado", ticket.OSNumber))

	return c.JSON(ticket)
}

func (h *TicketHandler) DeleteSignature(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	ticket, err := h.service.DeleteSignature(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Signature deleted successfully",
		"ticket":  ticket,
	})
}

func (h *TicketHandler) UploadFiles(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	if ticketID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	if _, err := h.service.GetByID(ticketID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No files provided",
		})
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
		"application/pdf": true,
	}

	uploadDir := filepath.Join("uploads", "tickets", ticketID)
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	var savedFiles []models.TicketFile
	for _, file := range files {
		contentType := file.Header.Get("Content-Type")
		if !allowedTypes[contentType] {
			continue
		}

		if file.Size > 10*1024*1024 {
			continue
		}

		ext := filepath.Ext(file.Filename)
		safeFilename := uuid.New().String() + ext
		filePath := filepath.Join(uploadDir, safeFilename)

		if err := c.SaveFile(file, filePath); err != nil {
			continue
		}

		ticketFile := models.TicketFile{
			TicketID: ticketID,
			FileName: file.Filename,
			FilePath: filePath,
			FileType: contentType,
			FileSize: file.Size,
		}

		if err := h.db.Create(&ticketFile).Error; err != nil {
			os.Remove(filePath)
			continue
		}

		savedFiles = append(savedFiles, ticketFile)
	}

	if len(savedFiles) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No valid files were uploaded",
		})
	}

	h.logActivity(c, "upload", ticketID, fmt.Sprintf("%d arquivo(s) anexado(s) ao ticket", len(savedFiles)))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"files": savedFiles,
		"count": len(savedFiles),
	})
}

func (h *TicketHandler) GetFiles(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	if ticketID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ticket ID",
		})
	}

	var files []models.TicketFile
	if err := h.db.Where("ticket_id = ?", ticketID).Order("created_at DESC").Find(&files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch files",
		})
	}

	return c.JSON(files)
}

func (h *TicketHandler) DeleteFile(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	fileID := c.Params("fileId")

	var file models.TicketFile
	if err := h.db.Where("id = ? AND ticket_id = ?", fileID, ticketID).First(&file).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	os.Remove(file.FilePath)

	if err := h.db.Delete(&file).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete file",
		})
	}

	h.logActivity(c, "delete", ticketID, fmt.Sprintf("Arquivo %s removido do ticket", file.FileName))

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *TicketHandler) DownloadFile(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	fileID := c.Params("fileId")

	var file models.TicketFile
	if err := h.db.Where("id = ? AND ticket_id = ?", fileID, ticketID).First(&file).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "File not found on disk",
		})
	}

	c.Set("Content-Type", file.FileType)
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", strings.ReplaceAll(file.FileName, "\"", "_")))
	return c.SendFile(file.FilePath)
}
