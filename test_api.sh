#!/bin/bash

# ========================================
# Tech ERP Backend API Tests with curl
# ========================================
# Execute: chmod +x test_api.sh && ./test_api.sh
# ========================================

# Don't exit on errors - we want to run all tests

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"
TEST_EMAIL="${TEST_EMAIL:-admin@example.com}"
TEST_PASSWORD="${TEST_PASSWORD:-changeme}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0

# Token storage
ACCESS_TOKEN=""
REFRESH_TOKEN=""

# Test IDs for cleanup
USER_ID=""
TECHNICIAN_ID=""
CLIENT_ID=""
TICKET_ID=""
CATEGORY_ID=""
HIERARCHY_ID=""
NODE_ID=""
FINANCIAL_ENTRY_ID=""
STOCK_ITEM_ID=""
STOCK_LOCATION_ID=""

# ========================================
# Utility Functions
# ========================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((SKIPPED++))
}

log_section() {
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
}

# Check HTTP status code
check_status() {
    local expected=$1
    local actual=$2
    local test_name=$3
    
    if [ "$actual" -eq "$expected" ]; then
        log_success "$test_name (HTTP $actual)"
        return 0
    else
        log_fail "$test_name (Expected HTTP $expected, got $actual)"
        return 1
    fi
}

# Make authenticated request
auth_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Content-Type: application/json" \
            "$API_URL$endpoint"
    else
        curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_URL$endpoint"
    fi
}

# Extract status code from response
get_status() {
    echo "$1" | tail -n 1
}

# Extract body from response
get_body() {
    echo "$1" | sed '$d'
}

# ========================================
# PUBLIC ENDPOINTS
# ========================================

log_section "PUBLIC ENDPOINTS"

# Health Check
log_info "Testing Health Check..."
response=$(curl -s -w "\n%{http_code}" "$API_URL/health")
status=$(get_status "$response")
check_status 200 "$status" "GET /health"

# Version
log_info "Testing Version..."
response=$(curl -s -w "\n%{http_code}" "$API_URL/version")
status=$(get_status "$response")
check_status 200 "$status" "GET /version"

# Terms
log_info "Testing Terms..."
response=$(curl -s -w "\n%{http_code}" "$API_URL/terms")
status=$(get_status "$response")
check_status 200 "$status" "GET /terms"

# ========================================
# AUTHENTICATION
# ========================================

log_section "AUTHENTICATION"

# Sign In (get token for subsequent tests)
log_info "Testing Sign In..."
response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "$API_URL/auth/signin")
status=$(get_status "$response")
body=$(get_body "$response")

if [ "$status" -eq 200 ]; then
    log_success "POST /auth/signin (HTTP $status)"
    # API returns "token" and optionally "refreshToken"
    ACCESS_TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -z "$ACCESS_TOKEN" ]; then
        ACCESS_TOKEN=$(echo "$body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    fi
    # Try camelCase first (new format), then snake_case (old format)
    REFRESH_TOKEN=$(echo "$body" | grep -o '"refreshToken":"[^"]*"' | cut -d'"' -f4)
    if [ -z "$REFRESH_TOKEN" ]; then
        REFRESH_TOKEN=$(echo "$body" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
    fi
    log_info "Token obtained: ${ACCESS_TOKEN:0:50}..."
    if [ -n "$REFRESH_TOKEN" ]; then
        log_info "Refresh token obtained: ${REFRESH_TOKEN:0:50}..."
    fi
else
    log_fail "POST /auth/signin (HTTP $status) - Cannot continue without token"
    echo "$body"
    exit 1
fi

# Refresh Token
log_info "Testing Refresh Token..."
if [ -n "$REFRESH_TOKEN" ]; then
    # Use refresh token in body (new method)
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refreshToken\": \"$REFRESH_TOKEN\"}")
else
    # Fallback: use access token in header (backwards compatibility)
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/refresh" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
fi
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 200 ]; then
    log_success "POST /auth/refresh (HTTP $status)"
    # Update token if new one is returned
    NEW_TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$NEW_TOKEN" ]; then
        ACCESS_TOKEN="$NEW_TOKEN"
        log_info "Token refreshed successfully"
    fi
    # Update refresh token if rotated
    NEW_REFRESH=$(echo "$body" | grep -o '"refreshToken":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$NEW_REFRESH" ]; then
        REFRESH_TOKEN="$NEW_REFRESH"
        log_info "Refresh token rotated"
    fi
else
    log_fail "POST /auth/refresh (HTTP $status)"
    echo "$body"
fi

# Change Password (skip - would change the password)
log_skip "POST /auth/change-password (Would change password)"

# ========================================
# USERS (requires admin)
# ========================================

log_section "USERS"

# List Users
log_info "Testing List Users..."
response=$(auth_request GET "/users")
status=$(get_status "$response")
check_status 200 "$status" "GET /users"

# Search Users
log_info "Testing Search Users..."
response=$(auth_request GET "/users/search?q=admin")
status=$(get_status "$response")
check_status 200 "$status" "GET /users/search"

# Create User
log_info "Testing Create User..."
TIMESTAMP=$(date +%s)
response=$(auth_request POST "/users" '{
    "email": "testuser_'"$TIMESTAMP"'@test.com",
    "password": "Test@123456",
    "firstName": "Test",
    "lastName": "User",
    "role": "EMPLOYEE"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /users (HTTP $status)"
    USER_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created user ID: $USER_ID"
else
    log_fail "POST /users (HTTP $status)"
    echo "$body"
fi

# Get User by ID
if [ -n "$USER_ID" ]; then
    log_info "Testing Get User by ID..."
    response=$(auth_request GET "/users/$USER_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /users/:id"
    
    # Update User
    log_info "Testing Update User..."
    response=$(auth_request PUT "/users/$USER_ID" '{
        "name": "Test User Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /users/:id"
    
    # Toggle User Status
    log_info "Testing Toggle User Status..."
    response=$(auth_request POST "/users/$USER_ID/toggle-status")
    status=$(get_status "$response")
    check_status 200 "$status" "POST /users/:id/toggle-status"
    
    # Reset Password (skip - needs specific handling)
    log_skip "POST /users/:id/reset-password (Would reset password)"
fi

# ========================================
# CATEGORIES
# ========================================

log_section "CATEGORIES"

# Create Category (needed for tickets)
log_info "Testing Create Category..."
response=$(auth_request POST "/categories" '{
    "name": "Test Category '$(date +%s)'",
    "description": "Test category description"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /categories (HTTP $status)"
    CATEGORY_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created category ID: $CATEGORY_ID"
else
    log_fail "POST /categories (HTTP $status)"
    echo "$body"
fi

# List Categories
log_info "Testing List Categories..."
response=$(auth_request GET "/categories")
status=$(get_status "$response")
check_status 200 "$status" "GET /categories"

# Get Category by ID
if [ -n "$CATEGORY_ID" ]; then
    log_info "Testing Get Category by ID..."
    response=$(auth_request GET "/categories/$CATEGORY_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /categories/:id"
    
    # Update Category
    log_info "Testing Update Category..."
    response=$(auth_request PUT "/categories/$CATEGORY_ID" '{
        "name": "Test Category Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /categories/:id"
fi

# ========================================
# CLIENTS
# ========================================

log_section "CLIENTS"

# Create Client PJ
log_info "Testing Create Client PJ..."
TIMESTAMP=$(date +%s)
CNPJ_RAND=$(printf '%014d' $((RANDOM * RANDOM)))
response=$(auth_request POST "/clients" '{
    "fullName": "Test Company '"$TIMESTAMP"'",
    "email": "company_'"$TIMESTAMP"'@test.com",
    "phone": "11999999999",
    "cnpj": "'"$CNPJ_RAND"'",
    "type": "PJ",
    "address": "Test Street, 123",
    "city": "São Paulo",
    "state": "SP"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /clients (PJ) (HTTP $status)"
    CLIENT_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created client ID: $CLIENT_ID"
else
    log_fail "POST /clients (PJ) (HTTP $status)"
    echo "$body"
fi

# Create Client PF
log_info "Testing Create Client PF..."
TIMESTAMP2=$(date +%s)
CPF_RAND2=$(printf '%011d' $((RANDOM * RANDOM)))
response=$(auth_request POST "/clients" '{
    "fullName": "Test Person '"$TIMESTAMP2"'",
    "email": "person_'"$TIMESTAMP2"'@test.com",
    "phone": "11988888888",
    "cpf": "'"$CPF_RAND2"'",
    "type": "PF",
    "address": "Test Avenue, 456",
    "city": "Rio de Janeiro",
    "state": "RJ"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /clients (PF) (HTTP $status)"
else
    log_fail "POST /clients (PF) (HTTP $status)"
    echo "$body"
fi

# List Clients
log_info "Testing List Clients..."
response=$(auth_request GET "/clients")
status=$(get_status "$response")
check_status 200 "$status" "GET /clients"

# Count Clients
log_info "Testing Count Clients..."
response=$(auth_request GET "/clients/count")
status=$(get_status "$response")
check_status 200 "$status" "GET /clients/count"

# Get Client by ID
if [ -n "$CLIENT_ID" ]; then
    log_info "Testing Get Client by ID..."
    response=$(auth_request GET "/clients/$CLIENT_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /clients/:id"
    
    # Update Client
    log_info "Testing Update Client..."
    response=$(auth_request PUT "/clients/$CLIENT_ID" '{
        "name": "Test Company Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /clients/:id"
fi

# ========================================
# TECHNICIANS
# ========================================

log_section "TECHNICIANS"

# Create Technician
log_info "Testing Create Technician..."
TIMESTAMP=$(date +%s)
CPF_RAND=$(printf '%011d' $((RANDOM * RANDOM)))
response=$(auth_request POST "/technicians" '{
    "fullName": "Test Technician '"$TIMESTAMP"'",
    "email": "tech_'"$TIMESTAMP"'@test.com",
    "phone": "11977777777",
    "cpf": "'"$CPF_RAND"'",
    "address": "Tech Street, 789",
    "city": "Campinas",
    "state": "SP",
    "status": "active"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /technicians (HTTP $status)"
    TECHNICIAN_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created technician ID: $TECHNICIAN_ID"
else
    log_fail "POST /technicians (HTTP $status)"
    echo "$body"
fi

# List Technicians
log_info "Testing List Technicians..."
response=$(auth_request GET "/technicians")
status=$(get_status "$response")
check_status 200 "$status" "GET /technicians"

# Get Technicians by State
log_info "Testing Get Technicians by State..."
response=$(auth_request GET "/technicians/by-state/SP")
status=$(get_status "$response")
check_status 200 "$status" "GET /technicians/by-state/:state"

# Get Technicians by City
log_info "Testing Get Technicians by City..."
response=$(auth_request GET "/technicians/by-city/Campinas")
status=$(get_status "$response")
check_status 200 "$status" "GET /technicians/by-city/:city"

# Get Technician by ID
if [ -n "$TECHNICIAN_ID" ]; then
    log_info "Testing Get Technician by ID..."
    response=$(auth_request GET "/technicians/$TECHNICIAN_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /technicians/:id"
    
    # Update Technician
    log_info "Testing Update Technician..."
    response=$(auth_request PUT "/technicians/$TECHNICIAN_ID" '{
        "name": "Test Technician Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /technicians/:id"
fi

# ========================================
# TICKETS
# ========================================

log_section "TICKETS"

# Create Ticket
log_info "Testing Create Ticket..."
TICKET_DATA='{
    "osNumber": "OS-'$(date +%s)'",
    "errorDescription": "Test ticket error description",
    "priority": "MEDIUM",
    "status": "ABERTO"'

if [ -n "$CLIENT_ID" ]; then
    TICKET_DATA="$TICKET_DATA"',"clientId": '$CLIENT_ID
fi
if [ -n "$CATEGORY_ID" ]; then
    TICKET_DATA="$TICKET_DATA"',"categoryId": '$CATEGORY_ID
fi
TICKET_DATA="$TICKET_DATA"'}'

response=$(auth_request POST "/tickets" "$TICKET_DATA")
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /tickets (HTTP $status)"
    TICKET_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created ticket ID: $TICKET_ID"
else
    log_fail "POST /tickets (HTTP $status)"
    echo "$body"
fi

# List Tickets
log_info "Testing List Tickets..."
response=$(auth_request GET "/tickets")
status=$(get_status "$response")
check_status 200 "$status" "GET /tickets"

# List Tickets with filters
log_info "Testing List Tickets with Status Filter..."
response=$(auth_request GET "/tickets?status=open")
status=$(get_status "$response")
check_status 200 "$status" "GET /tickets?status=open"

# Get Ticket by ID
if [ -n "$TICKET_ID" ]; then
    log_info "Testing Get Ticket by ID..."
    response=$(auth_request GET "/tickets/$TICKET_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /tickets/:id"
    
    # Update Ticket
    log_info "Testing Update Ticket..."
    response=$(auth_request PUT "/tickets/$TICKET_ID" '{
        "title": "Test Ticket Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /tickets/:id"
    
    # Update Status
    log_info "Testing Update Ticket Status..."
    response=$(auth_request PUT "/tickets/$TICKET_ID/status" '{
        "status": "in_progress"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /tickets/:id/status"
    
    # Assign Technician
    if [ -n "$TECHNICIAN_ID" ]; then
        log_info "Testing Assign Technician..."
        response=$(auth_request PUT "/tickets/$TICKET_ID/assign" "{
            \"technician_id\": $TECHNICIAN_ID
        }")
        status=$(get_status "$response")
        check_status 200 "$status" "PUT /tickets/:id/assign"
    fi
fi

# ========================================
# DASHBOARD
# ========================================

log_section "DASHBOARD"

# Get Stats
log_info "Testing Dashboard Stats..."
response=$(auth_request GET "/dashboard/stats")
status=$(get_status "$response")
check_status 200 "$status" "GET /dashboard/stats"

# Get Tickets by Status
log_info "Testing Tickets by Status..."
response=$(auth_request GET "/dashboard/tickets-by-status")
status=$(get_status "$response")
check_status 200 "$status" "GET /dashboard/tickets-by-status"

# Get Technicians by State
log_info "Testing Technicians by State..."
response=$(auth_request GET "/dashboard/technicians-by-state")
status=$(get_status "$response")
check_status 200 "$status" "GET /dashboard/technicians-by-state"

# Get Chart Data
log_info "Testing Chart Data..."
response=$(auth_request GET "/dashboard/chart")
status=$(get_status "$response")
check_status 200 "$status" "GET /dashboard/chart"

# Get Recent Activity
log_info "Testing Recent Activity..."
response=$(auth_request GET "/dashboard/recent-activity")
status=$(get_status "$response")
check_status 200 "$status" "GET /dashboard/recent-activity"

# ========================================
# HIERARCHIES
# ========================================

log_section "HIERARCHIES"

# Create Hierarchy
log_info "Testing Create Hierarchy..."
response=$(auth_request POST "/hierarchies" '{
    "name": "Test Hierarchy '$(date +%s)'",
    "description": "Test hierarchy for API testing"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /hierarchies (HTTP $status)"
    HIERARCHY_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created hierarchy ID: $HIERARCHY_ID"
else
    log_fail "POST /hierarchies (HTTP $status)"
    echo "$body"
fi

# List Hierarchies
log_info "Testing List Hierarchies..."
response=$(auth_request GET "/hierarchies")
status=$(get_status "$response")
check_status 200 "$status" "GET /hierarchies"

# Get Hierarchy by ID
if [ -n "$HIERARCHY_ID" ]; then
    log_info "Testing Get Hierarchy by ID..."
    response=$(auth_request GET "/hierarchies/$HIERARCHY_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /hierarchies/:id"
    
    # Update Hierarchy
    log_info "Testing Update Hierarchy..."
    response=$(auth_request PUT "/hierarchies/$HIERARCHY_ID" '{
        "name": "Test Hierarchy Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /hierarchies/:id"
    
    # Create Node
    log_info "Testing Create Node..."
    response=$(auth_request POST "/hierarchies/$HIERARCHY_ID/nodes" '{
        "name": "Test Node '$(date +%s)'",
        "node_type": "department"
    }')
    status=$(get_status "$response")
    body=$(get_body "$response")
    if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
        log_success "POST /hierarchies/:id/nodes (HTTP $status)"
        NODE_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
        log_info "Created node ID: $NODE_ID"
    else
        log_fail "POST /hierarchies/:id/nodes (HTTP $status)"
    fi
fi

# List Nodes
log_info "Testing List Nodes..."
response=$(auth_request GET "/nodes")
status=$(get_status "$response")
check_status 200 "$status" "GET /nodes"

# Get Node Members (if node exists)
if [ -n "$NODE_ID" ]; then
    log_info "Testing Get Node Members..."
    response=$(auth_request GET "/nodes/$NODE_ID/members")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /nodes/:id/members"
fi

# List Roles
log_info "Testing List Roles..."
response=$(auth_request GET "/roles")
status=$(get_status "$response")
check_status 200 "$status" "GET /roles"

# List Permissions
log_info "Testing List Permissions..."
response=$(auth_request GET "/permissions")
status=$(get_status "$response")
check_status 200 "$status" "GET /permissions"

# ========================================
# ACTIVITY LOGS
# ========================================

log_section "ACTIVITY LOGS"

# Get Activity Logs
log_info "Testing Get Activity Logs..."
response=$(auth_request GET "/activity-logs")
status=$(get_status "$response")
check_status 200 "$status" "GET /activity-logs"

# Get Recent Activity Logs
log_info "Testing Get Recent Activity Logs..."
response=$(auth_request GET "/activity-logs/recent")
status=$(get_status "$response")
check_status 200 "$status" "GET /activity-logs/recent"

# ========================================
# GEO LOCATIONS
# ========================================

log_section "GEOLOCATION"

# Get Technicians Last Locations
log_info "Testing Get Technicians Last Locations..."
response=$(auth_request GET "/geo/technicians/last")
status=$(get_status "$response")
check_status 200 "$status" "GET /geo/technicians/last"

# Get Geo Settings
log_info "Testing Get Geo Settings..."
response=$(auth_request GET "/geo/settings")
status=$(get_status "$response")
check_status 200 "$status" "GET /geo/settings"

# Create Location (if technician exists)
if [ -n "$TECHNICIAN_ID" ]; then
    log_info "Testing Create Location..."
    response=$(auth_request POST "/geo/locations" "{
        \"technician_id\": $TECHNICIAN_ID,
        \"latitude\": -23.5505,
        \"longitude\": -46.6333,
        \"accuracy\": 10.5
    }")
    status=$(get_status "$response")
    if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
        log_success "POST /geo/locations (HTTP $status)"
    else
        log_fail "POST /geo/locations (HTTP $status)"
    fi
    
    # Get Technician History
    log_info "Testing Get Technician History..."
    response=$(auth_request GET "/geo/technicians/$TECHNICIAN_ID/history")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /geo/technicians/:id/history"
fi

# ========================================
# ADMIN ROUTES
# ========================================

log_section "ADMIN ROUTES"

# Security Logs
log_info "Testing Get Security Logs..."
response=$(auth_request GET "/admin/security-logs")
status=$(get_status "$response")
check_status 200 "$status" "GET /admin/security-logs"

# Recent Security Logs
log_info "Testing Get Recent Security Logs..."
response=$(auth_request GET "/admin/security-logs/recent")
status=$(get_status "$response")
check_status 200 "$status" "GET /admin/security-logs/recent"

# Security Stats
log_info "Testing Get Security Stats..."
response=$(auth_request GET "/admin/security-logs/stats")
status=$(get_status "$response")
check_status 200 "$status" "GET /admin/security-logs/stats"

# System Metrics
log_info "Testing Get System Metrics..."
response=$(auth_request GET "/admin/system-metrics")
status=$(get_status "$response")
check_status 200 "$status" "GET /admin/system-metrics"

# ========================================
# FINANCIAL
# ========================================

log_section "FINANCIAL"

# Get Financial Categories
log_info "Testing Get Financial Categories..."
response=$(auth_request GET "/financial/categories")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/categories"

# Get Financial Dashboard
log_info "Testing Get Financial Dashboard..."
response=$(auth_request GET "/financial/dashboard")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/dashboard"

# Get Cash Flow Report
log_info "Testing Get Cash Flow Report..."
START_DATE=$(date -d "-30 days" +%Y-%m-%d 2>/dev/null || date -v-30d +%Y-%m-%d 2>/dev/null || echo "2025-12-01")
END_DATE=$(date +%Y-%m-%d)
response=$(auth_request GET "/financial/reports/cash-flow?startDate=$START_DATE&endDate=$END_DATE")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/reports/cash-flow"

# Get Technician Payments Report
log_info "Testing Get Technician Payments Report..."
response=$(auth_request GET "/financial/reports/technician-payments?startDate=$START_DATE&endDate=$END_DATE")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/reports/technician-payments"

# Create Financial Entry
log_info "Testing Create Financial Entry..."
ENTRY_DATE=$(date +%Y-%m-%d)
response=$(auth_request POST "/financial/entries" '{
    "type": "expense",
    "category": "operational",
    "subcategory": "fuel",
    "description": "Test expense '$(date +%s)'",
    "amount": 150.00,
    "entryDate": "'"$ENTRY_DATE"'"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /financial/entries (HTTP $status)"
    FINANCIAL_ENTRY_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created financial entry ID: $FINANCIAL_ENTRY_ID"
else
    log_fail "POST /financial/entries (HTTP $status)"
    echo "$body"
fi

# List Financial Entries
log_info "Testing List Financial Entries..."
response=$(auth_request GET "/financial/entries")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/entries"

# Get Financial Entry by ID
if [ -n "$FINANCIAL_ENTRY_ID" ]; then
    log_info "Testing Get Financial Entry by ID..."
    response=$(auth_request GET "/financial/entries/$FINANCIAL_ENTRY_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /financial/entries/:id"
    
    # Update Financial Entry
    log_info "Testing Update Financial Entry..."
    response=$(auth_request PUT "/financial/entries/$FINANCIAL_ENTRY_ID" '{
        "description": "Test expense updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /financial/entries/:id"
    
    # Update Entry Status
    log_info "Testing Update Entry Status..."
    response=$(auth_request PATCH "/financial/entries/$FINANCIAL_ENTRY_ID/status" '{
        "status": "paid"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PATCH /financial/entries/:id/status"
fi

# List Batches
log_info "Testing List Batches..."
response=$(auth_request GET "/financial/batches")
status=$(get_status "$response")
check_status 200 "$status" "GET /financial/batches"

# ========================================
# STOCK
# ========================================

log_section "STOCK"

# Create Stock Location
log_info "Testing Create Stock Location..."
TIMESTAMP=$(date +%s)
response=$(auth_request POST "/stock/locations" '{
    "name": "Test Location '"$TIMESTAMP"'",
    "type": "WAREHOUSE",
    "scopeId": "00000000-0000-0000-0000-000000000001"
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /stock/locations (HTTP $status)"
    STOCK_LOCATION_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created stock location ID: $STOCK_LOCATION_ID"
else
    log_fail "POST /stock/locations (HTTP $status)"
    echo "$body"
fi

# List Stock Locations
log_info "Testing List Stock Locations..."
response=$(auth_request GET "/stock/locations")
status=$(get_status "$response")
check_status 200 "$status" "GET /stock/locations"

# Create Stock Item
log_info "Testing Create Stock Item..."
response=$(auth_request POST "/stock/items" '{
    "name": "Test Item '$(date +%s)'",
    "sku": "TEST-'$(date +%s)'",
    "description": "Test stock item",
    "unit": "un",
    "min_quantity": 5
}')
status=$(get_status "$response")
body=$(get_body "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /stock/items (HTTP $status)"
    STOCK_ITEM_ID=$(echo "$body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    log_info "Created stock item ID: $STOCK_ITEM_ID"
else
    log_fail "POST /stock/items (HTTP $status)"
    echo "$body"
fi

# List Stock Items
log_info "Testing List Stock Items..."
response=$(auth_request GET "/stock/items")
status=$(get_status "$response")
check_status 200 "$status" "GET /stock/items"

# Get Stock Item by ID
if [ -n "$STOCK_ITEM_ID" ]; then
    log_info "Testing Get Stock Item by ID..."
    response=$(auth_request GET "/stock/items/$STOCK_ITEM_ID")
    status=$(get_status "$response")
    check_status 200 "$status" "GET /stock/items/:id"
    
    # Update Stock Item
    log_info "Testing Update Stock Item..."
    response=$(auth_request PUT "/stock/items/$STOCK_ITEM_ID" '{
        "name": "Test Item Updated"
    }')
    status=$(get_status "$response")
    check_status 200 "$status" "PUT /stock/items/:id"
fi

# List Movements
log_info "Testing List Stock Movements..."
response=$(auth_request GET "/stock/movements")
status=$(get_status "$response")
check_status 200 "$status" "GET /stock/movements"

# Create Movement (if item and location exist)
if [ -n "$STOCK_ITEM_ID" ] && [ -n "$STOCK_LOCATION_ID" ]; then
    log_info "Testing Create Stock Movement..."
    response=$(auth_request POST "/stock/movements" "{
        \"item_id\": $STOCK_ITEM_ID,
        \"location_id\": $STOCK_LOCATION_ID,
        \"movement_type\": \"in\",
        \"quantity\": 10,
        \"reason\": \"Test movement\"
    }")
    status=$(get_status "$response")
    if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
        log_success "POST /stock/movements (HTTP $status)"
    else
        log_fail "POST /stock/movements (HTTP $status)"
    fi
fi

# List Balances
log_info "Testing List Stock Balances..."
response=$(auth_request GET "/stock/balances")
status=$(get_status "$response")
check_status 200 "$status" "GET /stock/balances"

# ========================================
# EXPORTS
# ========================================

log_section "EXPORTS"

# Export Clients
log_info "Testing Export Clients..."
response=$(auth_request GET "/export/clients")
status=$(get_status "$response")
check_status 200 "$status" "GET /export/clients"

# Export Technicians
log_info "Testing Export Technicians..."
response=$(auth_request GET "/export/technicians")
status=$(get_status "$response")
check_status 200 "$status" "GET /export/technicians"

# Export Tickets
log_info "Testing Export Tickets..."
response=$(auth_request GET "/export/tickets")
status=$(get_status "$response")
check_status 200 "$status" "GET /export/tickets"

# Export Categories
log_info "Testing Export Categories..."
response=$(auth_request GET "/export/categories")
status=$(get_status "$response")
check_status 200 "$status" "GET /export/categories"

# ========================================
# ERROR LOGS
# ========================================

log_section "ERROR LOGS"

# Create Frontend Error
log_info "Testing Create Frontend Error..."
response=$(auth_request POST "/errors/frontend" '{
    "error_type": "javascript_error",
    "message": "Test error from curl",
    "stack_trace": "Error: Test\n    at test.js:1:1",
    "url": "/test",
    "user_agent": "curl/test"
}')
status=$(get_status "$response")
if [ "$status" -eq 201 ] || [ "$status" -eq 200 ]; then
    log_success "POST /errors/frontend (HTTP $status)"
else
    log_fail "POST /errors/frontend (HTTP $status)"
fi

# Get Error Logs (admin)
log_info "Testing Get Error Logs..."
response=$(auth_request GET "/errors")
status=$(get_status "$response")
check_status 200 "$status" "GET /errors"

# Get Error Stats
log_info "Testing Get Error Stats..."
response=$(auth_request GET "/errors/stats")
status=$(get_status "$response")
check_status 200 "$status" "GET /errors/stats"

# ========================================
# CLEANUP (DELETE operations)
# ========================================

log_section "CLEANUP (DELETE operations)"

# Delete Ticket
if [ -n "$TICKET_ID" ]; then
    log_info "Testing Delete Ticket..."
    response=$(auth_request DELETE "/tickets/$TICKET_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /tickets/:id (HTTP $status)"
    else
        log_fail "DELETE /tickets/:id (HTTP $status)"
    fi
fi

# Delete Client
if [ -n "$CLIENT_ID" ]; then
    log_info "Testing Delete Client..."
    response=$(auth_request DELETE "/clients/$CLIENT_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /clients/:id (HTTP $status)"
    else
        log_fail "DELETE /clients/:id (HTTP $status)"
    fi
fi

# Delete Technician
if [ -n "$TECHNICIAN_ID" ]; then
    log_info "Testing Delete Technician..."
    response=$(auth_request DELETE "/technicians/$TECHNICIAN_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /technicians/:id (HTTP $status)"
    else
        log_fail "DELETE /technicians/:id (HTTP $status)"
    fi
fi

# Delete Category
if [ -n "$CATEGORY_ID" ]; then
    log_info "Testing Delete Category..."
    response=$(auth_request DELETE "/categories/$CATEGORY_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /categories/:id (HTTP $status)"
    else
        log_fail "DELETE /categories/:id (HTTP $status)"
    fi
fi

# Delete Stock Item
if [ -n "$STOCK_ITEM_ID" ]; then
    log_info "Testing Delete Stock Item..."
    response=$(auth_request DELETE "/stock/items/$STOCK_ITEM_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /stock/items/:id (HTTP $status)"
    else
        log_fail "DELETE /stock/items/:id (HTTP $status)"
    fi
fi

# Delete Stock Location
if [ -n "$STOCK_LOCATION_ID" ]; then
    log_info "Testing Delete Stock Location..."
    response=$(auth_request DELETE "/stock/locations/$STOCK_LOCATION_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /stock/locations/:id (HTTP $status)"
    else
        log_fail "DELETE /stock/locations/:id (HTTP $status)"
    fi
fi

# Delete Node (if exists)
if [ -n "$NODE_ID" ]; then
    log_info "Testing Delete Node..."
    response=$(auth_request DELETE "/nodes/$NODE_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /nodes/:id (HTTP $status)"
    else
        log_fail "DELETE /nodes/:id (HTTP $status)"
    fi
fi

# Delete Hierarchy
if [ -n "$HIERARCHY_ID" ]; then
    log_info "Testing Delete Hierarchy..."
    response=$(auth_request DELETE "/hierarchies/$HIERARCHY_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /hierarchies/:id (HTTP $status)"
    else
        log_fail "DELETE /hierarchies/:id (HTTP $status)"
    fi
fi

# Delete Financial Entry
if [ -n "$FINANCIAL_ENTRY_ID" ]; then
    log_info "Testing Delete Financial Entry..."
    response=$(auth_request DELETE "/financial/entries/$FINANCIAL_ENTRY_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /financial/entries/:id (HTTP $status)"
    else
        log_fail "DELETE /financial/entries/:id (HTTP $status)"
    fi
fi

# Delete User
if [ -n "$USER_ID" ]; then
    log_info "Testing Delete User..."
    response=$(auth_request DELETE "/users/$USER_ID")
    status=$(get_status "$response")
    if [ "$status" -eq 200 ] || [ "$status" -eq 204 ]; then
        log_success "DELETE /users/:id (HTTP $status)"
    else
        log_fail "DELETE /users/:id (HTTP $status)"
    fi
fi

# ========================================
# SUMMARY
# ========================================

log_section "TEST SUMMARY"

TOTAL=$((PASSED + FAILED + SKIPPED))

echo ""
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
echo -e "Total: $TOTAL"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
