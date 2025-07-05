# RubrDuck Prompts System

This package provides a flexible prompt management system for RubrDuck, allowing both default prompts and custom user-defined prompts.

## Overview

The prompts system supports:

- **Default prompts** for each mode (planning, building, debugging, enhance)
- **Custom prompts** that can override defaults
- **Tooling preamble** that explains available tools to the AI
- **Template variables** for dynamic prompt customization

## Default Prompts

Default prompts are stored in `internal/prompts/defaults/` as YAML files:

- `planning.yaml` - Project planning and architecture design
- `building.yaml` - Code implementation and development
- `debugging.yaml` - Debugging and problem-solving
- `enhance.yaml` - Code quality improvement and refactoring
- `tooling_preamble.yaml` - Explains available tools (file_operations, shell_execute, git_operations)

## Custom Prompts

Users can create custom prompts by:

1. Setting the `prompts.custom_dir` in their config file:

   ```yaml
   prompts:
     custom_dir: ~/.rubrduck/prompts
   ```

2. Creating YAML files in that directory with the same names as default prompts to override them

## YAML Template Structure

```yaml
name: "Mode Name"
description: "Description of this mode"
system_prompt: |
  Your system prompt here...
  Can be multiple lines...
tooling_preamble: |
  Optional custom tooling preamble...
  (If not provided, default tooling preamble is used)
variables:
  key1: "default value 1"
  key2: "default value 2"
```

## Template Variables

Prompts can include template variables using Go template syntax:

```yaml
system_prompt: |
  Hello {{.Name}}!
  Working on project: {{.ProjectName}}
variables:
  ProjectName: "MyProject"
```

Variables can be:

- Defined in the template's `variables` section (defaults)
- Passed when calling `GetPrompt(mode, variables)`

## Usage in Code

```go
// Create prompt manager
pm, err := prompts.NewPromptManager(customDir)

// Get a prompt
prompt, err := pm.GetPrompt("planning", nil)

// Get a prompt with variables
vars := map[string]string{"ProjectName": "RubrDuck"}
prompt, err := pm.GetPrompt("building", vars)

// List available modes
modes := pm.ListModes()
```

## Adding New Modes

To add a new mode:

1. Create a YAML file in the defaults directory or custom directory
2. Follow the template structure above
3. The mode name is the filename without `.yaml` extension

## Best Practices

1. Keep prompts focused on specific tasks
2. Include clear methodologies and output formats
3. Use the tooling preamble to explain available tools
4. Test custom prompts thoroughly before deploying
5. Use template variables for project-specific customization
