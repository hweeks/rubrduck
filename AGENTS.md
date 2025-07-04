# RubrDuck Agent Instructions

## File Operations Best Practices

### Large File Handling

When working with files, be aware of size limitations:

1. **Small files (< 50KB)**: Can be written normally in a single operation
2. **Medium files (50KB - 200KB)**: Will work but may be slow - the system will warn about this
3. **Large files (> 200KB)**: Will be rejected to prevent timeout issues

### Strategies for Large File Updates

Instead of rewriting entire large files, use these approaches:

1. **Incremental Updates**: Update only the specific sections that need changes
2. **Search and Replace**: Use targeted replacements for specific content
3. **Append Operations**: Add new content to the end of files when possible
4. **Section-by-Section**: Break large updates into multiple smaller operations

### Example: Updating NEXT_STEPS.md

Instead of:

```
// DON'T DO THIS - Rewriting entire large file
file_operations: {
  "type": "write",
  "path": "NEXT_STEPS.md",
  "content": "... entire 250KB file content ..."
}
```

Do this:

```
// DO THIS - Update specific sections
file_operations: {
  "type": "write",
  "path": "NEXT_STEPS_updates.md",
  "content": "... just the new or changed sections ..."
}
```

Or use shell commands for targeted updates:

```
shell_execute: {
  "command": "sed -i '' 's/old text/new text/g' NEXT_STEPS.md"
}
```

## General Guidelines

1. Always check file size before attempting large writes
2. Break complex operations into smaller, manageable chunks
3. Use appropriate tools for the task (shell commands for find/replace, file operations for new content)
4. Provide progress updates for long-running operations
5. Consider the timeout constraints when planning operations
