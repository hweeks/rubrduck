# RubrDuck Agent: Large Context Production Planning

## Overview

This document outlines the strategy for handling large contexts in production environments, moving beyond simple token limit management to intelligent context processing.

## Current State Analysis

### Token Limit Issues

- **Problem**: Models have hard token limits (e.g., Claude 3.5 Sonnet: 200k tokens)
- **Current Approach**: Conservative limits (context: 50k, completion: 10k)
- **Limitation**: Arbitrary truncation loses important context

### Context Components Breakdown

```
System Prompt: ~2-3k tokens
Tooling Preamble: ~1-2k tokens
Plan Context: ~5-10k tokens
User Input: ~1-5k tokens
Response Buffer: ~10k tokens
Total: ~19-30k tokens (well within limits)
```

## Production Context Strategy

### 1. Intelligent Context Chunking

#### Hierarchical File Analysis

```go
type ContextChunk struct {
    Priority    int     // 1-10, higher = more important
    Relevance   float64 // 0.0-1.0, semantic relevance score
    Size        int     // token count
    Dependencies []string // files this depends on
    LastModified time.Time
}

type ChunkedContext struct {
    Essential    []ContextChunk // Always included
    Relevant     []ContextChunk // High relevance to current task
    Background   []ContextChunk // Lower priority context
    Excluded     []ContextChunk // Below threshold
}
```

#### Chunking Strategy

- **Essential**: Configuration files, main entry points, current working files
- **Relevant**: Files with high semantic similarity to current task
- **Background**: Supporting files, utilities, tests
- **Excluded**: Generated files, large binaries, irrelevant directories

### 2. Relevance-Based Context Selection

#### Semantic Analysis Pipeline

```go
func calculateRelevance(query string, filePath string, content string) float64 {
    // 1. Path relevance (directory structure matching)
    pathScore := calculatePathRelevance(query, filePath)

    // 2. Content relevance (semantic similarity)
    contentScore := calculateContentRelevance(query, content)

    // 3. Recency relevance (recently modified files)
    recencyScore := calculateRecencyScore(filePath)

    // 4. Dependency relevance (imports, references)
    dependencyScore := calculateDependencyRelevance(filePath, projectGraph)

    return weightedAverage(pathScore, contentScore, recencyScore, dependencyScore)
}
```

#### Implementation Steps

1. **Query Analysis**: Extract key terms, file patterns, function names
2. **File Scoring**: Calculate relevance for each project file
3. **Threshold Filtering**: Include files above relevance threshold
4. **Size Balancing**: Ensure total context fits within limits

### 3. Hierarchical Context Management

#### Context Levels

```
Level 1: Current File (always included)
Level 2: Direct Dependencies (imports, includes)
Level 3: Related Files (same module/package)
Level 4: Supporting Files (tests, configs, docs)
Level 5: Background Context (project structure, conventions)
```

#### Dynamic Context Injection

```go
type ContextManager struct {
    baseContext    []ContextChunk
    dynamicContext []ContextChunk
    maxTokens      int
    currentTask    string
}

func (cm *ContextManager) GetContext() string {
    // Start with essential context
    context := cm.baseContext

    // Add task-relevant context
    relevant := cm.getRelevantChunks(cm.currentTask)
    context = append(context, relevant...)

    // Ensure we stay within limits
    return cm.truncateToLimit(context)
}
```

### 4. Smart File Handling

#### File Type Classification

```go
type FileType int

const (
    FileTypeSource FileType = iota
    FileTypeConfig
    FileTypeTest
    FileTypeDocumentation
    FileTypeGenerated
    FileTypeBinary
    FileTypeVendor
)

func classifyFile(filePath string) FileType {
    // Implementation based on:
    // - File extension
    // - Directory location
    // - Content analysis
    // - Git ignore patterns
}
```

#### File Processing Rules

- **Source Files**: Full content with syntax highlighting
- **Config Files**: Include in context, highlight relevant sections
- **Test Files**: Include only if directly related to current task
- **Documentation**: Extract relevant sections, not full files
- **Generated Files**: Exclude unless explicitly requested
- **Binary Files**: Exclude, provide metadata only

### 5. Dynamic Context Adaptation

#### Context Evolution

```go
type ContextState struct {
    CurrentFiles    []string
    RecentChanges   []string
    ActiveImports   []string
    TaskHistory     []string
    UserPreferences map[string]interface{}
}

func (cs *ContextState) UpdateContext(newTask string) {
    // Analyze task requirements
    requirements := analyzeTask(newTask)

    // Update context based on requirements
    cs.CurrentFiles = requirements.requiredFiles
    cs.ActiveImports = requirements.imports

    // Maintain context continuity
    cs.TaskHistory = append(cs.TaskHistory, newTask)
}
```

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2)

1. **Context Chunking System**

   - Implement `ContextChunk` and `ChunkedContext` structures
   - Create file classification system
   - Build basic relevance scoring

2. **Token Management**
   - Implement token counting for different file types
   - Create context size estimation
   - Build truncation strategies

### Phase 2: Intelligence (Week 3-4)

1. **Semantic Analysis**

   - Implement content relevance scoring
   - Build dependency analysis
   - Create path relevance algorithms

2. **Context Selection**
   - Implement threshold-based filtering
   - Create priority-based selection
   - Build size balancing algorithms

### Phase 3: Integration (Week 5-6)

1. **Agent Integration**

   - Modify agent to use new context system
   - Update prompt generation
   - Implement context state management

2. **Performance Optimization**
   - Add caching for relevance calculations
   - Implement incremental context updates
   - Optimize token counting

### Phase 4: Production (Week 7-8)

1. **Testing & Validation**

   - Create comprehensive test suite
   - Performance benchmarking
   - User experience testing

2. **Documentation & Deployment**
   - Update user documentation
   - Create configuration guides
   - Deploy with monitoring

## Technical Specifications

### Configuration Options

```yaml
context:
  max_tokens: 50000
  relevance_threshold: 0.3
  include_tests: false
  include_generated: false
  cache_relevance: true
  dynamic_loading: true

  chunking:
    max_file_size: 10000
    min_relevance: 0.1
    priority_boost:
      current_file: 2.0
      recent_changes: 1.5
      dependencies: 1.2
```

### API Changes

```go
// New context manager interface
type ContextManager interface {
    GetContext(task string) (string, error)
    UpdateContext(task string) error
    GetRelevantFiles(query string) ([]string, error)
    EstimateTokens(files []string) (int, error)
}

// Enhanced agent interface
type Agent interface {
    Execute(task string, context ContextManager) error
    GetContextSummary() string
    UpdatePreferences(prefs map[string]interface{}) error
}
```

## Risk Assessment

### Technical Risks

1. **Performance Impact**: Relevance calculation overhead
   - **Mitigation**: Caching, incremental updates, background processing
2. **Accuracy Loss**: Incorrect relevance scoring
   - **Mitigation**: Multiple scoring algorithms, user feedback, continuous improvement
3. **Context Fragmentation**: Important information split across chunks
   - **Mitigation**: Dependency tracking, context continuity, summary generation

### User Experience Risks

1. **Inconsistent Behavior**: Different context for similar tasks
   - **Mitigation**: Deterministic algorithms, user preferences, context history
2. **Slow Response**: Context processing delays
   - **Mitigation**: Async processing, progressive loading, smart caching

## Success Metrics

### Performance Metrics

- **Context Processing Time**: < 100ms for typical projects
- **Token Efficiency**: > 80% relevance score for included context
- **Memory Usage**: < 50MB for context management

### Quality Metrics

- **Task Success Rate**: > 95% for tasks with sufficient context
- **User Satisfaction**: > 4.5/5 for context relevance
- **Error Reduction**: < 5% context-related errors

### Operational Metrics

- **Cache Hit Rate**: > 80% for relevance calculations
- **Context Hit Rate**: > 90% for essential files
- **Processing Overhead**: < 10% of total execution time

## Conclusion

This planning document outlines a comprehensive approach to handling large contexts in production. The key insight is moving from simple token management to intelligent context processing that maintains relevance while staying within model limits.

The implementation prioritizes:

1. **Intelligence**: Semantic understanding of context relevance
2. **Efficiency**: Smart caching and incremental updates
3. **Flexibility**: Configurable thresholds and preferences
4. **Reliability**: Robust error handling and fallback strategies

This approach will enable RubrDuck to handle projects of any size while maintaining high-quality, relevant context for the AI agent.
