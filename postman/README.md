# TechERP API - Postman Collection

This folder contains the Postman collection and environment files for testing the TechERP API.

## Files

- `TechERP_API.postman_collection.json` - Complete API collection with all endpoints
- `TechERP_API.postman_environment.json` - Environment variables for Production

## How to Import

1. Open Postman
2. Click on **Import** button (top left)
3. Select both JSON files or drag them into Postman
4. The collection and environment will be imported

## Setup

1. After importing, select the **TechERP - Production** environment from the dropdown (top right)
2. The `baseUrl` is pre-configured to `https://srv1209774.hstgr.cloud/api/v1`

## Authentication

1. Open the **Auth > Sign In** request
2. Update the email/password if needed (default: `admin@techerp.com` / `admin123`)
3. Send the request
4. The token will be automatically saved to the `{{token}}` variable (via test script)
5. All subsequent requests will use this token automatically

## Endpoints Overview

| Category | Endpoints |
|----------|-----------|
| Health | 1 |
| Auth | 4 |
| Users | 8 |
| Technicians | 9 |
| Tickets | 7 |
| Clients | 6 |
| Categories | 5 |
| Dashboard | 5 |
| Export | 4 |
| Terms | 2 |
| Hierarchies | 6 |
| Nodes | 5 |
| Memberships | 2 |
| Roles | 5 |
| Permissions | 1 |
| Access | 4 |
| **Total** | **74** |

## Notes

- Requests with `:id`, `:userId`, etc. require you to replace the variable with an actual value
- Some requests require ADMIN role (Users CRUD, Roles CRUD, etc.)
- The Sign In request has a test script that automatically saves the token to the collection variable
