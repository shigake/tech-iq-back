package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// FinancialEntryType represents income or expense
type FinancialEntryType string

const (
	FinancialEntryTypeIncome  FinancialEntryType = "income"
	FinancialEntryTypeExpense FinancialEntryType = "expense"
)

// FinancialEntryStatus represents the payment status
type FinancialEntryStatus string

const (
	FinancialEntryStatusPending   FinancialEntryStatus = "pending"
	FinancialEntryStatusPaid      FinancialEntryStatus = "paid"
	FinancialEntryStatusOverdue   FinancialEntryStatus = "overdue"
	FinancialEntryStatusCancelled FinancialEntryStatus = "cancelled"
)

// PaymentBatchStatus represents the batch status
type PaymentBatchStatus string

const (
	PaymentBatchStatusDraft      PaymentBatchStatus = "draft"
	PaymentBatchStatusApproved   PaymentBatchStatus = "approved"
	PaymentBatchStatusProcessing PaymentBatchStatus = "processing"
	PaymentBatchStatusPaid       PaymentBatchStatus = "paid"
	PaymentBatchStatusCancelled  PaymentBatchStatus = "cancelled"
)

// FinancialCategory represents predefined categories
type FinancialCategory struct {
	Type          FinancialEntryType `json:"type"`
	Category      string             `json:"category"`
	Subcategories []string           `json:"subcategories"`
}

// GetFinancialCategories returns all predefined categories
func GetFinancialCategories() []FinancialCategory {
	return []FinancialCategory{
		// Income categories
		{Type: FinancialEntryTypeIncome, Category: "service", Subcategories: []string{"os_completion", "maintenance", "installation", "consulting"}},
		{Type: FinancialEntryTypeIncome, Category: "product", Subcategories: []string{"equipment_sale", "parts_sale"}},
		{Type: FinancialEntryTypeIncome, Category: "other", Subcategories: []string{"refund", "bonus", "adjustment"}},
		// Expense categories
		{Type: FinancialEntryTypeExpense, Category: "technician_payment", Subcategories: []string{"commission", "bonus", "reimbursement"}},
		{Type: FinancialEntryTypeExpense, Category: "operational", Subcategories: []string{"fuel", "tools", "equipment", "supplies"}},
		{Type: FinancialEntryTypeExpense, Category: "administrative", Subcategories: []string{"rent", "utilities", "software", "services"}},
		{Type: FinancialEntryTypeExpense, Category: "tax", Subcategories: []string{"federal", "state", "municipal"}},
		{Type: FinancialEntryTypeExpense, Category: "other", Subcategories: []string{"adjustment", "loss"}},
	}
}

// FinancialEntry represents a financial entry (income or expense)
type FinancialEntry struct {
	ID          string             `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Type        FinancialEntryType `json:"type" gorm:"type:varchar(10);not null;index"`
	Category    string             `json:"category" gorm:"type:varchar(50);not null;index"`
	Subcategory string             `json:"subcategory" gorm:"type:varchar(50)"`
	Description string             `json:"description" gorm:"type:text;not null"`
	Amount      float64            `json:"amount" gorm:"type:decimal(12,2);not null"`
	Currency    string             `json:"currency" gorm:"type:varchar(3);default:BRL"`

	// Dates
	EntryDate   time.Time  `json:"entryDate" gorm:"type:date;not null;index"`
	DueDate     *time.Time `json:"dueDate" gorm:"type:date"`
	PaymentDate *time.Time `json:"paymentDate" gorm:"type:date"`

	// Status
	Status FinancialEntryStatus `json:"status" gorm:"type:varchar(20);not null;default:pending;index"`

	// Optional relationships
	TicketID     *string     `json:"ticketId" gorm:"type:uuid;index"`
	Ticket       *Ticket     `json:"ticket,omitempty" gorm:"foreignKey:TicketID"`
	TechnicianID *string     `json:"technicianId" gorm:"type:uuid;index"`
	Technician   *Technician `json:"technician,omitempty" gorm:"foreignKey:TechnicianID"`
	ClientID     *string     `json:"clientId" gorm:"type:uuid;index"`
	Client       *Client     `json:"client,omitempty" gorm:"foreignKey:ClientID"`

	// Payment info
	PaymentMethod    string `json:"paymentMethod" gorm:"type:varchar(30)"`
	PaymentReference string `json:"paymentReference" gorm:"type:varchar(100)"`

	// Attachments (stored as array of URLs)
	AttachmentURLs pq.StringArray `json:"attachmentUrls" gorm:"type:text[]"`

	// Audit
	CreatedBy   string         `json:"createdBy" gorm:"type:uuid;not null"`
	CreatedByUser *User        `json:"createdByUser,omitempty" gorm:"foreignKey:CreatedBy"`
	UpdatedBy   *string        `json:"updatedBy" gorm:"type:uuid"`
	UpdatedByUser *User        `json:"updatedByUser,omitempty" gorm:"foreignKey:UpdatedBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Optimistic locking
	Version int `json:"version" gorm:"default:1"`

	// Payment batches (many-to-many)
	PaymentBatches []PaymentBatch `json:"paymentBatches,omitempty" gorm:"many2many:payment_batch_entries;joinForeignKey:entry_id;joinReferences:batch_id"`
}

func (f *FinancialEntry) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	if f.Currency == "" {
		f.Currency = "BRL"
	}
	if f.Status == "" {
		f.Status = FinancialEntryStatusPending
	}
	return nil
}

func (FinancialEntry) TableName() string {
	return "financial_entries"
}

// PaymentBatch represents a batch of payments to be processed together
type PaymentBatch struct {
	ID          string `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string `json:"name" gorm:"type:varchar(100);not null"`
	Description string `json:"description" gorm:"type:text"`

	// Reference period
	PeriodStart time.Time `json:"periodStart" gorm:"type:date;not null"`
	PeriodEnd   time.Time `json:"periodEnd" gorm:"type:date;not null"`

	// Calculated totals
	TotalAmount  float64 `json:"totalAmount" gorm:"type:decimal(12,2);default:0"`
	EntriesCount int     `json:"entriesCount" gorm:"default:0"`

	// Status
	Status PaymentBatchStatus `json:"status" gorm:"type:varchar(20);not null;default:draft;index"`

	// Approval
	ApprovedBy   *string    `json:"approvedBy" gorm:"type:uuid"`
	ApprovedByUser *User    `json:"approvedByUser,omitempty" gorm:"foreignKey:ApprovedBy"`
	ApprovedAt   *time.Time `json:"approvedAt"`

	// Payment
	PaidAt           *time.Time `json:"paidAt"`
	PaymentReference string     `json:"paymentReference" gorm:"type:varchar(100)"`

	// Entries (many-to-many)
	Entries []FinancialEntry `json:"entries,omitempty" gorm:"many2many:payment_batch_entries;joinForeignKey:batch_id;joinReferences:entry_id"`

	// Audit
	CreatedBy     string         `json:"createdBy" gorm:"type:uuid;not null"`
	CreatedByUser *User          `json:"createdByUser,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (p *PaymentBatch) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Status == "" {
		p.Status = PaymentBatchStatusDraft
	}
	return nil
}

func (PaymentBatch) TableName() string {
	return "payment_batches"
}

// FinancialAuditLog represents an audit log for financial operations
type FinancialAuditLog struct {
	ID          string    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityType  string    `json:"entityType" gorm:"type:varchar(30);not null;index"`
	EntityID    string    `json:"entityId" gorm:"type:uuid;not null;index"`
	Action      string    `json:"action" gorm:"type:varchar(20);not null"`
	Changes     string    `json:"changes" gorm:"type:jsonb"` // JSON string of changes
	PerformedBy string    `json:"performedBy" gorm:"type:uuid;not null"`
	PerformedByUser *User `json:"performedByUser,omitempty" gorm:"foreignKey:PerformedBy"`
	PerformedAt time.Time `json:"performedAt" gorm:"default:now();index"`
	IPAddress   string    `json:"ipAddress" gorm:"type:varchar(45)"`
	UserAgent   string    `json:"userAgent" gorm:"type:text"`
}

func (f *FinancialAuditLog) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

func (FinancialAuditLog) TableName() string {
	return "financial_audit_logs"
}

// FinancialDashboard represents the financial dashboard data
type FinancialDashboard struct {
	Summary struct {
		TotalIncome  float64 `json:"totalIncome"`
		TotalExpense float64 `json:"totalExpense"`
		Balance      float64 `json:"balance"`
	} `json:"summary"`
	ByCategory struct {
		Income  map[string]float64 `json:"income"`
		Expense map[string]float64 `json:"expense"`
	} `json:"byCategory"`
	PendingPayments int64            `json:"pendingPayments"`
	OverdueCount    int64            `json:"overdueCount"`
	RecentEntries   []FinancialEntry `json:"recentEntries"`
}

// CashFlowPeriod represents a period in the cash flow report
type CashFlowPeriod struct {
	Period  string  `json:"period"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Balance float64 `json:"balance"`
}

// CashFlowReport represents the cash flow report data
type CashFlowReport struct {
	Periods []CashFlowPeriod `json:"periods"`
}

// TechnicianPaymentReport represents technician payment summary
type TechnicianPaymentReport struct {
	TechnicianID   string  `json:"technicianId"`
	TechnicianName string  `json:"technicianName"`
	Total          float64 `json:"total"`
	EntriesCount   int     `json:"entriesCount"`
}

// TechnicianPaymentsReport represents the full report
type TechnicianPaymentsReport struct {
	Technicians []TechnicianPaymentReport `json:"technicians"`
}

// CreateFinancialEntryRequest represents the request to create a financial entry
type CreateFinancialEntryRequest struct {
	Type             FinancialEntryType `json:"type" validate:"required,oneof=income expense"`
	Category         string             `json:"category" validate:"required"`
	Subcategory      string             `json:"subcategory"`
	Description      string             `json:"description" validate:"required"`
	Amount           float64            `json:"amount" validate:"required,gt=0"`
	EntryDate        string             `json:"entryDate" validate:"required"` // Format: YYYY-MM-DD
	DueDate          string             `json:"dueDate"`
	TicketID         string             `json:"ticketId"`
	TechnicianID     string             `json:"technicianId"`
	ClientID         string             `json:"clientId"`
	PaymentMethod    string             `json:"paymentMethod"`
	PaymentReference string             `json:"paymentReference"`
	AttachmentURLs   []string           `json:"attachmentUrls"`
}

// UpdateFinancialEntryRequest represents the request to update a financial entry
type UpdateFinancialEntryRequest struct {
	Type             FinancialEntryType `json:"type"`
	Category         string             `json:"category"`
	Subcategory      string             `json:"subcategory"`
	Description      string             `json:"description"`
	Amount           float64            `json:"amount"`
	EntryDate        string             `json:"entryDate"`
	DueDate          string             `json:"dueDate"`
	TicketID         string             `json:"ticketId"`
	TechnicianID     string             `json:"technicianId"`
	ClientID         string             `json:"clientId"`
	PaymentMethod    string             `json:"paymentMethod"`
	PaymentReference string             `json:"paymentReference"`
	AttachmentURLs   []string           `json:"attachmentUrls"`
	Version          int                `json:"version" validate:"required"` // For optimistic locking
}

// UpdateFinancialEntryStatusRequest represents the request to update entry status
type UpdateFinancialEntryStatusRequest struct {
	Status           FinancialEntryStatus `json:"status" validate:"required,oneof=pending paid overdue cancelled"`
	PaymentDate      string               `json:"paymentDate"`
	PaymentReference string               `json:"paymentReference"`
}

// CreatePaymentBatchRequest represents the request to create a payment batch
type CreatePaymentBatchRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	PeriodStart string `json:"periodStart" validate:"required"` // Format: YYYY-MM-DD
	PeriodEnd   string `json:"periodEnd" validate:"required"`   // Format: YYYY-MM-DD
}

// AddBatchEntriesRequest represents the request to add entries to a batch
type AddBatchEntriesRequest struct {
	EntryIDs []string `json:"entryIds" validate:"required,min=1"`
}

// PayBatchRequest represents the request to mark a batch as paid
type PayBatchRequest struct {
	PaymentReference string `json:"paymentReference"`
}

// FinancialEntryFilter represents filters for querying financial entries
type FinancialEntryFilter struct {
	Type         FinancialEntryType   `query:"type"`
	Status       FinancialEntryStatus `query:"status"`
	Category     string               `query:"category"`
	StartDate    string               `query:"startDate"`
	EndDate      string               `query:"endDate"`
	TechnicianID string               `query:"technicianId"`
	ClientID     string               `query:"clientId"`
	TicketID     string               `query:"ticketId"`
	Page         int                  `query:"page"`
	Limit        int                  `query:"limit"`
}

// PaymentBatchFilter represents filters for querying payment batches
type PaymentBatchFilter struct {
	Status      PaymentBatchStatus `query:"status"`
	PeriodStart string             `query:"periodStart"`
	PeriodEnd   string             `query:"periodEnd"`
	Page        int                `query:"page"`
	Limit       int                `query:"limit"`
}

// DashboardFilter represents filters for the financial dashboard
type DashboardFilter struct {
	Period    string `query:"period"` // today, week, month, quarter, year, custom
	StartDate string `query:"startDate"`
	EndDate   string `query:"endDate"`
}

// CashFlowFilter represents filters for the cash flow report
type CashFlowFilter struct {
	StartDate string `query:"startDate" validate:"required"`
	EndDate   string `query:"endDate" validate:"required"`
	GroupBy   string `query:"groupBy"` // day, week, month
}

// TechnicianPaymentsFilter represents filters for the technician payments report
type TechnicianPaymentsFilter struct {
	StartDate    string `query:"startDate" validate:"required"`
	EndDate      string `query:"endDate" validate:"required"`
	TechnicianID string `query:"technicianId"`
}
