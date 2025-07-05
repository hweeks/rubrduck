# Plan Metadata System Design

## Overview

The Plan Metadata System provides a centralized, efficient way to store, retrieve, and manage planning documents across different AI agent modes. All plan data is stored in a single `.metadata` directory with a unified structure that supports the complete AI agent workflow.

## Directory Structure

```
.duckie/
├── .metadata/
│   ├── {plan-id}.json    # Plan metadata (title, mode, status, timestamps, etc.)
│   └── {plan-id}.md      # Plan content (markdown format)
└── [other system files]
```

## Key Design Principles

### 1. Single Source of Truth

- All plan data is stored in `.metadata/` directory
- No mode-specific subdirectories to avoid confusion and duplication
- Unified metadata structure across all modes

### 2. Mode-Based Organization

- Plans are categorized by `mode` field in metadata
- Supported modes: `planning`, `building`, `debugging`, `enhance`
- Context injection filters by mode for relevant plan retrieval

### 3. Workflow Integration

- **Planning Mode**: Creates plans that become input for Building Mode
- **Building Mode**: Executes plans created in Planning Mode
- **Debugging Mode**: References plans for troubleshooting
- **Enhance Mode**: Improves existing plans

## File Format

### Metadata File (`{plan-id}.json`)

```json
{
  "id": "uuid-string",
  "title": "Plan Title",
  "mode": "planning|building|debugging|enhance",
  "description": "Brief description",
  "created": "2025-07-05T10:00:00Z",
  "updated": "2025-07-05T10:00:00Z",
  "version": 1,
  "status": "draft|active|completed|archived",
  "tags": ["tag1", "tag2"],
  "author": "author-name"
}
```

### Content File (`{plan-id}.md`)

- Markdown format for rich content
- Supports headers, lists, code blocks, etc.
- Human-readable and AI-processable

## Workflow Integration

### Planning → Building Workflow

1. **Planning Mode** creates detailed plans:

   ```
   .metadata/
   ├── plan-001.json  # {mode: "planning", status: "active"}
   └── plan-001.md    # Detailed implementation plan
   ```

2. **Building Mode** retrieves active planning plans:
   - Queries `.metadata/` for plans with `mode: "planning"` and `status: "active"`
   - Uses plan content as execution context
   - Can update plan status to `"completed"` when done

### Context Injection

The system automatically injects relevant plan context:

```go
// Get context for building mode
context := manager.GetContext("building", "")
// Returns:
// - Current active building plan (if any)
// - Related planning plans (for execution context)
// - Recent debugging plans (for troubleshooting context)
```

## API Usage

### Creating Plans

```go
plan, err := manager.CreatePlan("planning", "My Plan", "Description", "# Plan Content")
```

### Retrieving Context

```go
context, err := manager.GetContext("building", "")
// Returns PlanContext with related plans for AI injection
```

### Listing Plans by Mode

```go
filter := &PlanFilter{Mode: "planning", Status: "active"}
plans, err := manager.ListPlans(filter)
```

### Searching Plans

```go
results, err := manager.SearchPlans("implementation", &PlanFilter{Mode: "planning"})
```

## Benefits of This Design

### 1. Simplicity

- Single directory structure
- No empty folders or confusion
- Clear separation of metadata and content

### 2. Performance

- Fast queries by mode and status
- Efficient context injection
- Minimal filesystem operations

### 3. Maintainability

- Unified codebase for all modes
- Single storage implementation
- Easy to extend with new modes

### 4. Workflow Support

- Natural progression from planning to building
- Context preservation across modes
- Status tracking for plan lifecycle

## Mode-Specific Behaviors

### Planning Mode

- Creates detailed implementation plans
- Sets status to `"active"` for building consumption
- Can reference existing plans for continuity

### Building Mode

- Retrieves active planning plans for execution
- Updates plan status to `"completed"` when done
- Can create building-specific plans for complex implementations

### Debugging Mode

- References completed plans for troubleshooting
- Can create debugging plans with solutions
- Links back to original planning/building plans

### Enhance Mode

- Improves existing plans
- Creates new versions of plans
- Maintains plan history through versioning

## Future Enhancements

1. **Plan Relationships**: Track dependencies between plans
2. **Plan Templates**: Predefined structures for common plan types
3. **Plan Validation**: Ensure plan structure meets mode requirements
4. **Plan Export**: Export plans to external formats
5. **Plan Collaboration**: Support for multiple authors and reviews
