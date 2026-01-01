package models

// Address represents an embedded address structure
type Address struct {
	Street       string `json:"street" gorm:"type:varchar(255)"`
	Number       string `json:"number" gorm:"type:varchar(20)"`
	Complement   string `json:"complement" gorm:"type:varchar(100)"`
	Neighborhood string `json:"neighborhood" gorm:"type:varchar(100)"`
	City         string `json:"city" gorm:"type:varchar(100);index"`
	State        string `json:"state" gorm:"type:varchar(2);index"`
	ZipCode      string `json:"zipCode" gorm:"type:varchar(10)"`
	Country      string `json:"country" gorm:"type:varchar(50);default:Brasil"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Content       interface{} `json:"content"`
	Page          int         `json:"page"`
	Size          int         `json:"size"`
	TotalElements int64       `json:"totalElements"`
	TotalPages    int         `json:"totalPages"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalTechnicians  int64 `json:"totalTechnicians"`
	ActiveTechnicians int64 `json:"activeTechnicians"`
	TotalTickets      int64 `json:"totalTickets"`
	OpenTickets       int64 `json:"openTickets"`
	PendingTickets    int64 `json:"pendingTickets"`
	InProgressTickets int64 `json:"inProgressTickets"`
	ClosedTickets     int64 `json:"closedTickets"`
	TotalClients      int64 `json:"totalClients"`
}

// TicketsByStatus represents tickets grouped by status
type TicketsByStatus struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// TechniciansByState represents technicians grouped by state
type TechniciansByState struct {
	State string `json:"state"`
	Count int64  `json:"count"`
}

// RecentActivity represents a recent activity item for the dashboard
type RecentActivity struct {
	ID          string `json:"id"`
	Type        string `json:"type"`        // "technician", "ticket", "client"
	Action      string `json:"action"`      // "created", "updated"
	Title       string `json:"title"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}
