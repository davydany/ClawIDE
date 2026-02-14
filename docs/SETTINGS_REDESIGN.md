# Settings UI Redesign: AI Agent CLI Configuration

## Overview

The Settings page has been reorganized to provide better separation of concerns and comprehensive documentation for AI Agent CLI configuration. The AI Agent settings are now in their own dedicated block with full help documentation and examples.

## Changes Made

### 1. **General Settings Section** (Simplified)
- Now contains only core ClawIDE settings:
  - Projects Directory
  - Max Sessions
  - Listen Address
- AI Agent settings moved to dedicated section

### 2. **New: AI Agent CLI Configuration Section**
A comprehensive, self-contained block for configuring AI agents with:

#### Header & Description
- Clear section title: "AI Agent CLI Configuration"
- Context: "Configure the AI agent command and options that ClawIDE will auto-launch in new terminal panes."
- Toggle-able "Show Help" button for inline documentation

#### Collapsible Help Section
The "Show Help" button reveals inline documentation covering:
- **What is this?** - Explanation of AI agent auto-launch feature
- **Supported Agents:**
  - `claude` - Claude Code (recommended)
  - `codex` - OpenAI Codex
  - `aider` - Aider
  - `(custom)` - Any command path
- **CLI Arguments** - How to pass additional options
- **Common Options** - Quick reference for popular flags:
  - `--model sonnet|opus|haiku` - Model selection
  - `--effort low|medium|high` - Effort level
  - `--permission-mode plan|acceptEdits|default` - Permission level
  - `-c, --continue` - Resume previous session
  - `--verbose` - Enable verbose output
- **Full Reference** - Link to complete documentation

#### Configuration Fields

1. **AI Agent Command** (Required)
   - Dropdown with preset options
   - Custom command path input
   - Save button with visual feedback

2. **Custom Command Path** (Conditional)
   - Shows only when custom option selected
   - Placeholder examples provided
   - Clear description

3. **Additional CLI Arguments** (Conditional)
   - Shows only when agent is selected (not "shell only")
   - Placeholder examples: `--model opus --effort high --permission-mode acceptEdits`
   - Explains option separation

4. **Shell Only Note** (Conditional)
   - Shows when no agent selected
   - Clarifies that plain shell will be launched

## Documentation Files

### `docs/claude-cli-options.md`
A comprehensive reference document containing:
- **Quick Start** - Basic usage instructions
- **Session & Output Options** - Session management, resumption, output formats
- **Model & Performance** - Model selection, effort levels, fallback options
- **Permissions & Security** - Permission modes and security options
- **Tool & Environment Configuration** - Tool access, environment setup
- **Prompting & System** - Custom and system prompts
- **Advanced Features** - Custom agents, IDE integration, settings
- **Output Formatting** - Structured output, JSON schemas
- **Debugging** - Debug mode, logging, verbose output
- **MCP Servers** - MCP configuration and options
- **Input/Output Control** - Stream modes, replay options
- **API & Budget** - Cost limiting, beta features
- **File Resources** - File download options
- **Examples** - Common use cases and configurations
- **Common ClawIDE Configurations** - Pre-built examples for different workflows

## Usage Guide

### For Users: Accessing Help

1. **In Settings UI:**
   - Navigate to Settings page
   - Find "AI Agent CLI Configuration" section
   - Click "Show Help" button to see inline documentation
   - Review "Common Options" and "Supported Agents"

2. **For Complete Reference:**
   - See `docs/claude-cli-options.md`
   - Contains all available options with descriptions
   - Organized by category
   - Includes examples and use cases

### For Users: Common Configurations

#### High-Effort Development
```
Command: Claude Code (claude)
Arguments: --model opus --effort high --verbose
```

#### Budget-Conscious Quick Work
```
Command: Claude Code (claude)
Arguments: --model sonnet --effort low
```

#### Read-Only Analysis (Safe)
```
Command: Claude Code (claude)
Arguments: --tools Read,Grep,Glob --permission-mode plan
```

#### Full Automation (Trust-Based)
```
Command: Claude Code (claude)
Arguments: --model opus --permission-mode acceptEdits --effort high
```

#### Resume Previous Session
```
Command: Claude Code (claude)
Arguments: --continue
```

## Design Improvements

### Visual Hierarchy
- Dedicated section emphasizes importance of agent configuration
- Clear distinction between general settings and agent settings
- Collapsible help reduces cognitive load while providing access to information

### User Guidance
- Inline help explains concepts without overwhelming
- Dropdown presets make common choices obvious
- Placeholder text shows example arguments
- Progressive disclosure (Show Help button) for advanced users

### Documentation
- Two-tier approach: quick help in UI + comprehensive reference
- Examples for common use cases
- Organized by functional category
- Clear explanations of option purposes

## Files Modified

1. **`web/templates/pages/settings.html`**
   - Separated General and AI Agent CLI sections
   - Added collapsible help with inline documentation
   - Improved form field organization and labeling
   - Added conditional visibility for form elements

2. **`docs/claude-cli-options.md`** (New)
   - Comprehensive reference for all Claude CLI options
   - Organized by functional category
   - Includes examples and common configurations
   - Generated from `claude --help` output

## Future Enhancements

Potential improvements for future iterations:
- Favorite/preset argument combinations
- Validation of Claude CLI availability before save
- Live preview of what command will be executed
- Integration with environment variable display
- Per-project agent configurations
- Recent argument history/autocomplete
