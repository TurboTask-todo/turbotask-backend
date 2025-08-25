# Subtasks by Project ID API Documentation

## Overview

This API endpoint allows you to retrieve all subtasks associated with a specific project. It provides comprehensive filtering, sorting, and pagination capabilities for efficient data management.

## Endpoint

```
GET /api/v1/todo/projects/{project_id}/subtasks
```

## Authentication

- **Required**: Bearer Token (JWT)
- **Header**: `Authorization: Bearer <your_jwt_token>`

## URL Parameters

| Parameter    | Type | Required | Description                          |
| ------------ | ---- | -------- | ------------------------------------ |
| `project_id` | UUID | Yes      | The unique identifier of the project |

## Query Parameters

| Parameter          | Type    | Required | Default      | Description                                        |
| ------------------ | ------- | -------- | ------------ | -------------------------------------------------- |
| `include_archived` | boolean | No       | `false`      | Include archived subtasks and todos                |
| `status`           | string  | No       | -            | Filter by subtask status                           |
| `priority`         | string  | No       | -            | Filter by subtask priority                         |
| `search`           | string  | No       | -            | Search in subtask name, description, or todo title |
| `sort_by`          | string  | No       | `created_at` | Sort field                                         |
| `sort_order`       | string  | No       | `DESC`       | Sort order (ASC or DESC)                           |
| `page`             | integer | No       | `1`          | Page number (starts from 1)                        |
| `limit`            | integer | No       | `20`         | Items per page (max: 100)                          |

### Valid Status Values

- `todo`
- `not_started`
- `in_progress`
- `testing`
- `review`
- `pending`
- `completed`
- `done`
- `on_hold`
- `backlog`
- `blocked`
- `cancelled`

### Valid Priority Values

- `low`
- `medium`
- `high`
- `urgent`

### Valid Sort Fields

- `name` - Sort by subtask name
- `status` - Sort by subtask status
- `priority` - Sort by subtask priority
- `due_date` - Sort by due date
- `todo_title` - Sort by parent todo title
- `sort_order` - Sort by manual sort order
- `created_at` - Sort by creation date

## Response Format

### Success Response (200)

```json
{
  "success": true,
  "message": "Project subtasks retrieved successfully",
  "data": {
    "subtasks": [
      {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "todo_id": "987fcdeb-51a2-43d1-9c4b-123456789abc",
        "name": "Implement user authentication",
        "description": "Add JWT-based authentication system",
        "status": "in_progress",
        "priority": "high",
        "estimated_time": 480,
        "actual_time": 120,
        "due_date": "2024-02-15T23:59:59Z",
        "completed_at": null,
        "sort_order": 1,
        "is_archived": false,
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-20T14:30:00Z",
        "todo_title": "Authentication System",
        "todo_description": "Complete authentication system for the application",
        "todo_status": "in_progress",
        "todo_priority": "high"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 45,
      "total_pages": 3,
      "has_next_page": true,
      "has_prev_page": false,
      "next_page": 2,
      "prev_page": null
    },
    "filters": {
      "include_archived": false,
      "status": "in_progress",
      "priority": "high",
      "search": "auth"
    },
    "sorting": {
      "sort_by": "priority",
      "sort_order": "DESC"
    }
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

### Error Responses

#### 400 Bad Request

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid status: invalid_status. Valid statuses are: [todo not_started in_progress ...]"
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### 401 Unauthorized

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authorization header required"
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### 404 Not Found

```json
{
  "success": false,
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "project not found"
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### 500 Internal Server Error

```json
{
  "success": false,
  "error": {
    "code": "SUBTASK_RETRIEVAL_FAILED",
    "message": "failed to get subtasks: database connection error"
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

## Example Requests

### Basic Request

```bash
curl -X GET "http://localhost:8080/api/v1/todo/projects/123e4567-e89b-12d3-a456-426614174000/subtasks" \
  -H "Authorization: Bearer your_jwt_token"
```

### With Filtering and Sorting

```bash
curl -X GET "http://localhost:8080/api/v1/todo/projects/123e4567-e89b-12d3-a456-426614174000/subtasks?status=in_progress&priority=high&sort_by=due_date&sort_order=ASC&page=1&limit=10" \
  -H "Authorization: Bearer your_jwt_token"
```

### With Search

```bash
curl -X GET "http://localhost:8080/api/v1/todo/projects/123e4567-e89b-12d3-a456-426614174000/subtasks?search=authentication&include_archived=true" \
  -H "Authorization: Bearer your_jwt_token"
```

## Features

### 🔍 **Advanced Filtering**

- Filter by subtask status, priority
- Include/exclude archived items
- Full-text search across subtask names, descriptions, and parent todo titles

### 📊 **Flexible Sorting**

- Sort by any field (name, status, priority, due date, etc.)
- Ascending or descending order
- Default sort by creation date (newest first)

### 📄 **Pagination**

- Configurable page size (1-100 items)
- Complete pagination metadata
- Efficient offset-based pagination

### 🎯 **Rich Data**

- Subtask details with all fields
- Parent todo information included
- Timestamps for tracking

### ⚡ **Performance**

- Optimized database queries
- Single query for data and count
- Proper indexing support

## Use Cases

1. **Project Management Dashboard**: Display all subtasks for a project with filtering
2. **Progress Tracking**: Monitor subtask completion across a project
3. **Resource Planning**: Analyze workload by priority and status
4. **Reporting**: Generate project reports with subtask details
5. **Team Collaboration**: View all subtasks assigned to a project

## Technical Implementation

### Database Query Optimization

- Uses JOIN to fetch todo information efficiently
- Implements proper filtering at database level
- Supports partial indexes for performance

### Status Transition Validation

- Enforces valid status transitions
- Maintains data integrity
- Professional workflow management

### Rate Limiting

- Standard authentication middleware
- Protects against abuse
- Maintains system stability

## Notes

- All timestamps are in UTC format
- UUIDs are used for all ID fields
- Empty result sets return valid pagination metadata
- Search is case-insensitive and uses ILIKE for partial matching
- Archived items are excluded by default for better UX
- Maximum limit of 100 items per page to prevent performance issues
