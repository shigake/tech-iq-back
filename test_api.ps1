# ========================================
# Tech ERP Backend API Tests with curl (PowerShell)
# ========================================
# Execute: .\test_api.ps1
# ========================================

param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$TestEmail = $env:TEST_EMAIL,
    [string]$TestPassword = $env:TEST_PASSWORD
)

if (-not $TestEmail) { $TestEmail = "admin@example.com" }
if (-not $TestPassword) { $TestPassword = "changeme" }

$ErrorActionPreference = "Continue"

# Configuration
$ApiUrl = "$BaseUrl/api/v1"

# Counters
$script:Passed = 0
$script:Failed = 0
$script:Skipped = 0

# Token storage
$script:AccessToken = ""
$script:RefreshToken = ""

# Test IDs for cleanup
$script:UserId = ""
$script:TechnicianId = ""
$script:ClientId = ""
$script:TicketId = ""
$script:CategoryId = ""
$script:HierarchyId = ""
$script:NodeId = ""
$script:FinancialEntryId = ""
$script:StockItemId = ""
$script:StockLocationId = ""

# ========================================
# Utility Functions
# ========================================

function Write-Info($message) {
    Write-Host "[INFO] $message" -ForegroundColor Cyan
}

function Write-Pass($message) {
    Write-Host "[PASS] $message" -ForegroundColor Green
    $script:Passed++
}

function Write-Fail($message) {
    Write-Host "[FAIL] $message" -ForegroundColor Red
    $script:Failed++
}

function Write-Skip($message) {
    Write-Host "[SKIP] $message" -ForegroundColor Yellow
    $script:Skipped++
}

function Write-Section($message) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "$message" -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Yellow
}

function Test-Status {
    param(
        [int]$Expected,
        [int]$Actual,
        [string]$TestName
    )
    
    if ($Actual -eq $Expected) {
        Write-Pass "$TestName (HTTP $Actual)"
        return $true
    } else {
        Write-Fail "$TestName (Expected HTTP $Expected, got $Actual)"
        return $false
    }
}

function Invoke-ApiRequest {
    param(
        [string]$Method,
        [string]$Endpoint,
        [object]$Body = $null,
        [switch]$NoAuth
    )
    
    $headers = @{
        "Content-Type" = "application/json"
    }
    
    if (-not $NoAuth -and $script:AccessToken) {
        $headers["Authorization"] = "Bearer $($script:AccessToken)"
    }
    
    $params = @{
        Uri = "$ApiUrl$Endpoint"
        Method = $Method
        Headers = $headers
        UseBasicParsing = $true
    }
    
    if ($Body) {
        $params["Body"] = ($Body | ConvertTo-Json -Compress)
    }
    
    try {
        $response = Invoke-WebRequest @params -ErrorAction Stop
        return @{
            StatusCode = $response.StatusCode
            Content = $response.Content | ConvertFrom-Json -ErrorAction SilentlyContinue
            RawContent = $response.Content
        }
    } catch {
        $statusCode = 0
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        return @{
            StatusCode = $statusCode
            Content = $null
            Error = $_.Exception.Message
        }
    }
}

# ========================================
# PUBLIC ENDPOINTS
# ========================================

Write-Section "PUBLIC ENDPOINTS"

# Health Check
Write-Info "Testing Health Check..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/health" -NoAuth
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /health"

# Version
Write-Info "Testing Version..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/version" -NoAuth
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /version"

# Terms
Write-Info "Testing Terms..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/terms" -NoAuth
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /terms"

# ========================================
# AUTHENTICATION
# ========================================

Write-Section "AUTHENTICATION"

# Sign In
Write-Info "Testing Sign In..."
$loginBody = @{
    email = $TestEmail
    password = $TestPassword
}

try {
    $response = Invoke-WebRequest -Uri "$ApiUrl/auth/signin" -Method POST -Body ($loginBody | ConvertTo-Json) -ContentType "application/json" -UseBasicParsing
    $content = $response.Content | ConvertFrom-Json
    # API returns "token" and optionally "refreshToken"
    if ($content.token) {
        $script:AccessToken = $content.token
    } elseif ($content.access_token) {
        $script:AccessToken = $content.access_token
    }
    # Try camelCase first (new format), then snake_case (old format)
    if ($content.refreshToken) {
        $script:RefreshToken = $content.refreshToken
    } elseif ($content.refresh_token) {
        $script:RefreshToken = $content.refresh_token
    }
    Write-Pass "POST /auth/signin (HTTP $($response.StatusCode))"
    Write-Info "Token obtained: $($script:AccessToken.Substring(0, [Math]::Min(50, $script:AccessToken.Length)))..."
    if ($script:RefreshToken) {
        Write-Info "Refresh token obtained: $($script:RefreshToken.Substring(0, [Math]::Min(50, $script:RefreshToken.Length)))..."
    }
} catch {
    Write-Fail "POST /auth/signin - Cannot continue without token"
    Write-Host $_.Exception.Message
    exit 1
}

# Refresh Token
Write-Info "Testing Refresh Token..."
try {
    if ($script:RefreshToken) {
        # Use refresh token in body (new method)
        $refreshBody = @{ refreshToken = $script:RefreshToken }
        $response = Invoke-WebRequest -Uri "$ApiUrl/auth/refresh" -Method POST -Body ($refreshBody | ConvertTo-Json) -ContentType "application/json" -UseBasicParsing
    } else {
        # Fallback: use access token in header (backwards compatibility)
        $headers = @{ Authorization = "Bearer $script:AccessToken" }
        $response = Invoke-WebRequest -Uri "$ApiUrl/auth/refresh" -Method POST -Headers $headers -ContentType "application/json" -UseBasicParsing
    }
    $content = $response.Content | ConvertFrom-Json
    Write-Pass "POST /auth/refresh (HTTP $($response.StatusCode))"
    # Update token if new one is returned
    if ($content.token) {
        $script:AccessToken = $content.token
        Write-Info "Token refreshed successfully"
    }
    # Update refresh token if rotated
    if ($content.refreshToken) {
        $script:RefreshToken = $content.refreshToken
        Write-Info "Refresh token rotated"
    }
} catch {
    $statusCode = 0
    if ($_.Exception.Response) {
        $statusCode = [int]$_.Exception.Response.StatusCode
    }
    Write-Fail "POST /auth/refresh (HTTP $statusCode)"
}

Write-Skip "POST /auth/change-password (Would change password)"

# ========================================
# USERS
# ========================================

Write-Section "USERS"

# List Users
Write-Info "Testing List Users..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/users"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /users"

# Search Users
Write-Info "Testing Search Users..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/users/search?q=admin"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /users/search"

# Create User
Write-Info "Testing Create User..."
$timestamp = Get-Date -Format "yyyyMMddHHmmss"
$userBody = @{
    email = "testuser_$timestamp@test.com"
    password = "Test@123456"
    firstName = "Test"
    lastName = "User"
    role = "EMPLOYEE"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/users" -Body $userBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /users (HTTP $($response.StatusCode))"
    $script:UserId = $response.Content.id
    Write-Info "Created user ID: $($script:UserId)"
} else {
    Write-Fail "POST /users (HTTP $($response.StatusCode))"
}

# Get User by ID
if ($script:UserId) {
    Write-Info "Testing Get User by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/users/$($script:UserId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /users/:id"
    
    # Update User
    Write-Info "Testing Update User..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/users/$($script:UserId)" -Body @{ name = "Test User Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /users/:id"
    
    # Toggle Status
    Write-Info "Testing Toggle User Status..."
    $response = Invoke-ApiRequest -Method POST -Endpoint "/users/$($script:UserId)/toggle-status"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "POST /users/:id/toggle-status"
}

# ========================================
# CATEGORIES
# ========================================

Write-Section "CATEGORIES"

# Create Category
Write-Info "Testing Create Category..."
$categoryBody = @{
    name = "Test Category $timestamp"
    description = "Test category description"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/categories" -Body $categoryBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /categories (HTTP $($response.StatusCode))"
    $script:CategoryId = $response.Content.id
    Write-Info "Created category ID: $($script:CategoryId)"
} else {
    Write-Fail "POST /categories (HTTP $($response.StatusCode))"
}

# List Categories
Write-Info "Testing List Categories..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/categories"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /categories"

if ($script:CategoryId) {
    Write-Info "Testing Get Category by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/categories/$($script:CategoryId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /categories/:id"
    
    Write-Info "Testing Update Category..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/categories/$($script:CategoryId)" -Body @{ name = "Test Category Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /categories/:id"
}

# ========================================
# CLIENTS
# ========================================

Write-Section "CLIENTS"

# Create Client PJ
Write-Info "Testing Create Client PJ..."
$clientBody = @{
    fullName = "Test Company $timestamp"
    email = "company_$timestamp@test.com"
    phone = "11999999999"
    cnpj = [string](Get-Random -Maximum 99999999999999).ToString().PadLeft(14, '0')
    type = "PJ"
    address = "Test Street, 123"
    city = "São Paulo"
    state = "SP"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/clients" -Body $clientBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /clients (PJ) (HTTP $($response.StatusCode))"
    $script:ClientId = $response.Content.id
    Write-Info "Created client ID: $($script:ClientId)"
} else {
    Write-Fail "POST /clients (PJ) (HTTP $($response.StatusCode))"
}

# Create Client PF
Write-Info "Testing Create Client PF..."
$clientPfBody = @{
    fullName = "Test Person $timestamp"
    email = "person_$timestamp@test.com"
    phone = "11988888888"
    cpf = [string](Get-Random -Maximum 99999999999).ToString().PadLeft(11, '0')
    type = "PF"
    address = "Test Avenue, 456"
    city = "Rio de Janeiro"
    state = "RJ"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/clients" -Body $clientPfBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /clients (PF) (HTTP $($response.StatusCode))"
} else {
    Write-Fail "POST /clients (PF) (HTTP $($response.StatusCode))"
}

# List Clients
Write-Info "Testing List Clients..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/clients"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /clients"

# Count Clients
Write-Info "Testing Count Clients..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/clients/count"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /clients/count"

if ($script:ClientId) {
    Write-Info "Testing Get Client by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/clients/$($script:ClientId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /clients/:id"
    
    Write-Info "Testing Update Client..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/clients/$($script:ClientId)" -Body @{ name = "Test Company Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /clients/:id"
}

# ========================================
# TECHNICIANS
# ========================================

Write-Section "TECHNICIANS"

# Create Technician
Write-Info "Testing Create Technician..."
$techBody = @{
    fullName = "Test Technician $timestamp"
    email = "tech_$timestamp@test.com"
    phone = "11977777777"
    cpf = [string](Get-Random -Maximum 99999999999).ToString().PadLeft(11, '0')
    address = "Tech Street, 789"
    city = "Campinas"
    state = "SP"
    status = "active"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/technicians" -Body $techBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /technicians (HTTP $($response.StatusCode))"
    $script:TechnicianId = $response.Content.id
    Write-Info "Created technician ID: $($script:TechnicianId)"
} else {
    Write-Fail "POST /technicians (HTTP $($response.StatusCode))"
}

# List Technicians
Write-Info "Testing List Technicians..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/technicians"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /technicians"

# Get by State
Write-Info "Testing Get Technicians by State..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/technicians/by-state/SP"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /technicians/by-state/:state"

# Get by City
Write-Info "Testing Get Technicians by City..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/technicians/by-city/Campinas"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /technicians/by-city/:city"

if ($script:TechnicianId) {
    Write-Info "Testing Get Technician by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/technicians/$($script:TechnicianId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /technicians/:id"
    
    Write-Info "Testing Update Technician..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/technicians/$($script:TechnicianId)" -Body @{ name = "Test Technician Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /technicians/:id"
}

# ========================================
# TICKETS
# ========================================

Write-Section "TICKETS"

# Create Ticket
Write-Info "Testing Create Ticket..."
$ticketBody = @{
    title = "Test Ticket $timestamp"
    description = "Test ticket description"
    priority = "medium"
    status = "open"
}
if ($script:ClientId) { $ticketBody["client_id"] = $script:ClientId }
if ($script:CategoryId) { $ticketBody["category_id"] = $script:CategoryId }

$response = Invoke-ApiRequest -Method POST -Endpoint "/tickets" -Body $ticketBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /tickets (HTTP $($response.StatusCode))"
    $script:TicketId = $response.Content.id
    Write-Info "Created ticket ID: $($script:TicketId)"
} else {
    Write-Fail "POST /tickets (HTTP $($response.StatusCode))"
}

# List Tickets
Write-Info "Testing List Tickets..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/tickets"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /tickets"

# List with filter
Write-Info "Testing List Tickets with Status Filter..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/tickets?status=open"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /tickets?status=open"

if ($script:TicketId) {
    Write-Info "Testing Get Ticket by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/tickets/$($script:TicketId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /tickets/:id"
    
    Write-Info "Testing Update Ticket..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/tickets/$($script:TicketId)" -Body @{ title = "Test Ticket Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /tickets/:id"
    
    Write-Info "Testing Update Ticket Status..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/tickets/$($script:TicketId)/status" -Body @{ status = "in_progress" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /tickets/:id/status"
    
    if ($script:TechnicianId) {
        Write-Info "Testing Assign Technician..."
        $response = Invoke-ApiRequest -Method PUT -Endpoint "/tickets/$($script:TicketId)/assign" -Body @{ technician_id = $script:TechnicianId }
        Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /tickets/:id/assign"
    }
}

# ========================================
# DASHBOARD
# ========================================

Write-Section "DASHBOARD"

Write-Info "Testing Dashboard Stats..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/dashboard/stats"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /dashboard/stats"

Write-Info "Testing Tickets by Status..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/dashboard/tickets-by-status"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /dashboard/tickets-by-status"

Write-Info "Testing Technicians by State..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/dashboard/technicians-by-state"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /dashboard/technicians-by-state"

Write-Info "Testing Chart Data..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/dashboard/chart"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /dashboard/chart"

Write-Info "Testing Recent Activity..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/dashboard/recent-activity"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /dashboard/recent-activity"

# ========================================
# HIERARCHIES
# ========================================

Write-Section "HIERARCHIES"

Write-Info "Testing Create Hierarchy..."
$hierarchyBody = @{
    name = "Test Hierarchy $timestamp"
    description = "Test hierarchy for API testing"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/hierarchies" -Body $hierarchyBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /hierarchies (HTTP $($response.StatusCode))"
    $script:HierarchyId = $response.Content.id
    Write-Info "Created hierarchy ID: $($script:HierarchyId)"
} else {
    Write-Fail "POST /hierarchies (HTTP $($response.StatusCode))"
}

Write-Info "Testing List Hierarchies..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/hierarchies"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /hierarchies"

if ($script:HierarchyId) {
    Write-Info "Testing Get Hierarchy by ID..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/hierarchies/$($script:HierarchyId)"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /hierarchies/:id"
    
    Write-Info "Testing Update Hierarchy..."
    $response = Invoke-ApiRequest -Method PUT -Endpoint "/hierarchies/$($script:HierarchyId)" -Body @{ name = "Test Hierarchy Updated" }
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "PUT /hierarchies/:id"
    
    Write-Info "Testing Create Node..."
    $nodeBody = @{
        name = "Test Node $timestamp"
        node_type = "department"
    }
    $response = Invoke-ApiRequest -Method POST -Endpoint "/hierarchies/$($script:HierarchyId)/nodes" -Body $nodeBody
    if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
        Write-Pass "POST /hierarchies/:id/nodes (HTTP $($response.StatusCode))"
        $script:NodeId = $response.Content.id
        Write-Info "Created node ID: $($script:NodeId)"
    } else {
        Write-Fail "POST /hierarchies/:id/nodes (HTTP $($response.StatusCode))"
    }
}

Write-Info "Testing List Nodes..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/nodes"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /nodes"

Write-Info "Testing List Roles..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/roles"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /roles"

Write-Info "Testing List Permissions..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/permissions"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /permissions"

# ========================================
# ACTIVITY LOGS
# ========================================

Write-Section "ACTIVITY LOGS"

Write-Info "Testing Get Activity Logs..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/activity-logs"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /activity-logs"

Write-Info "Testing Get Recent Activity Logs..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/activity-logs/recent"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /activity-logs/recent"

# ========================================
# GEOLOCATION
# ========================================

Write-Section "GEOLOCATION"

Write-Info "Testing Get Technicians Last Locations..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/geo/technicians/last"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /geo/technicians/last"

Write-Info "Testing Get Geo Settings..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/geo/settings"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /geo/settings"

if ($script:TechnicianId) {
    Write-Info "Testing Create Location..."
    $locationBody = @{
        technician_id = $script:TechnicianId
        latitude = -23.5505
        longitude = -46.6333
        accuracy = 10.5
    }
    $response = Invoke-ApiRequest -Method POST -Endpoint "/geo/locations" -Body $locationBody
    if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
        Write-Pass "POST /geo/locations (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "POST /geo/locations (HTTP $($response.StatusCode))"
    }
    
    Write-Info "Testing Get Technician History..."
    $response = Invoke-ApiRequest -Method GET -Endpoint "/geo/technicians/$($script:TechnicianId)/history"
    Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /geo/technicians/:id/history"
}

# ========================================
# ADMIN ROUTES
# ========================================

Write-Section "ADMIN ROUTES"

Write-Info "Testing Get Security Logs..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/admin/security-logs"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /admin/security-logs"

Write-Info "Testing Get Recent Security Logs..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/admin/security-logs/recent"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /admin/security-logs/recent"

Write-Info "Testing Get Security Stats..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/admin/security-logs/stats"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /admin/security-logs/stats"

Write-Info "Testing Get System Metrics..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/admin/system-metrics"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /admin/system-metrics"

# ========================================
# FINANCIAL
# ========================================

Write-Section "FINANCIAL"

Write-Info "Testing Get Financial Categories..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/categories"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/categories"

Write-Info "Testing Get Financial Dashboard..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/dashboard"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/dashboard"

Write-Info "Testing Get Cash Flow Report..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/reports/cash-flow"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/reports/cash-flow"

Write-Info "Testing Get Technician Payments Report..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/reports/technician-payments"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/reports/technician-payments"

Write-Info "Testing Create Financial Entry..."
$entryBody = @{
    type = "expense"
    amount = 150.00
    description = "Test expense $timestamp"
    due_date = (Get-Date).AddDays(7).ToString("yyyy-MM-dd")
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/financial/entries" -Body $entryBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /financial/entries (HTTP $($response.StatusCode))"
    $script:FinancialEntryId = $response.Content.id
    Write-Info "Created financial entry ID: $($script:FinancialEntryId)"
} else {
    Write-Fail "POST /financial/entries (HTTP $($response.StatusCode))"
}

Write-Info "Testing List Financial Entries..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/entries"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/entries"

Write-Info "Testing List Batches..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/financial/batches"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /financial/batches"

# ========================================
# STOCK
# ========================================

Write-Section "STOCK"

Write-Info "Testing Create Stock Location..."
$locationBody = @{
    name = "Test Location $timestamp"
    description = "Test stock location"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/stock/locations" -Body $locationBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /stock/locations (HTTP $($response.StatusCode))"
    $script:StockLocationId = $response.Content.id
    Write-Info "Created stock location ID: $($script:StockLocationId)"
} else {
    Write-Fail "POST /stock/locations (HTTP $($response.StatusCode))"
}

Write-Info "Testing List Stock Locations..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/stock/locations"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /stock/locations"

Write-Info "Testing Create Stock Item..."
$itemBody = @{
    name = "Test Item $timestamp"
    sku = "TEST-$timestamp"
    description = "Test stock item"
    unit = "un"
    min_quantity = 5
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/stock/items" -Body $itemBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /stock/items (HTTP $($response.StatusCode))"
    $script:StockItemId = $response.Content.id
    Write-Info "Created stock item ID: $($script:StockItemId)"
} else {
    Write-Fail "POST /stock/items (HTTP $($response.StatusCode))"
}

Write-Info "Testing List Stock Items..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/stock/items"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /stock/items"

Write-Info "Testing List Stock Movements..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/stock/movements"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /stock/movements"

Write-Info "Testing List Stock Balances..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/stock/balances"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /stock/balances"

# ========================================
# EXPORTS
# ========================================

Write-Section "EXPORTS"

Write-Info "Testing Export Clients..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/export/clients"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /export/clients"

Write-Info "Testing Export Technicians..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/export/technicians"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /export/technicians"

Write-Info "Testing Export Tickets..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/export/tickets"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /export/tickets"

Write-Info "Testing Export Categories..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/export/categories"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /export/categories"

# ========================================
# ERROR LOGS
# ========================================

Write-Section "ERROR LOGS"

Write-Info "Testing Create Frontend Error..."
$errorBody = @{
    error_type = "javascript_error"
    message = "Test error from PowerShell"
    stack_trace = "Error: Test`n    at test.js:1:1"
    url = "/test"
    user_agent = "PowerShell/test"
}
$response = Invoke-ApiRequest -Method POST -Endpoint "/errors/frontend" -Body $errorBody
if ($response.StatusCode -eq 201 -or $response.StatusCode -eq 200) {
    Write-Pass "POST /errors/frontend (HTTP $($response.StatusCode))"
} else {
    Write-Fail "POST /errors/frontend (HTTP $($response.StatusCode))"
}

Write-Info "Testing Get Error Logs..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/errors"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /errors"

Write-Info "Testing Get Error Stats..."
$response = Invoke-ApiRequest -Method GET -Endpoint "/errors/stats"
Test-Status -Expected 200 -Actual $response.StatusCode -TestName "GET /errors/stats"

# ========================================
# CLEANUP
# ========================================

Write-Section "CLEANUP (DELETE operations)"

if ($script:TicketId) {
    Write-Info "Testing Delete Ticket..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/tickets/$($script:TicketId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /tickets/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /tickets/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:ClientId) {
    Write-Info "Testing Delete Client..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/clients/$($script:ClientId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /clients/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /clients/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:TechnicianId) {
    Write-Info "Testing Delete Technician..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/technicians/$($script:TechnicianId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /technicians/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /technicians/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:CategoryId) {
    Write-Info "Testing Delete Category..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/categories/$($script:CategoryId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /categories/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /categories/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:StockItemId) {
    Write-Info "Testing Delete Stock Item..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/stock/items/$($script:StockItemId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /stock/items/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /stock/items/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:StockLocationId) {
    Write-Info "Testing Delete Stock Location..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/stock/locations/$($script:StockLocationId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /stock/locations/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /stock/locations/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:NodeId) {
    Write-Info "Testing Delete Node..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/nodes/$($script:NodeId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /nodes/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /nodes/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:HierarchyId) {
    Write-Info "Testing Delete Hierarchy..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/hierarchies/$($script:HierarchyId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /hierarchies/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /hierarchies/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:FinancialEntryId) {
    Write-Info "Testing Delete Financial Entry..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/financial/entries/$($script:FinancialEntryId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /financial/entries/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /financial/entries/:id (HTTP $($response.StatusCode))"
    }
}

if ($script:UserId) {
    Write-Info "Testing Delete User..."
    $response = Invoke-ApiRequest -Method DELETE -Endpoint "/users/$($script:UserId)"
    if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 204) {
        Write-Pass "DELETE /users/:id (HTTP $($response.StatusCode))"
    } else {
        Write-Fail "DELETE /users/:id (HTTP $($response.StatusCode))"
    }
}

# ========================================
# SUMMARY
# ========================================

Write-Section "TEST SUMMARY"

$total = $script:Passed + $script:Failed + $script:Skipped

Write-Host ""
Write-Host "Passed: $($script:Passed)" -ForegroundColor Green
Write-Host "Failed: $($script:Failed)" -ForegroundColor Red
Write-Host "Skipped: $($script:Skipped)" -ForegroundColor Yellow
Write-Host "Total: $total"
Write-Host ""

if ($script:Failed -eq 0) {
    Write-Host "✅ All tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "❌ Some tests failed" -ForegroundColor Red
    exit 1
}
