# Autonomous Executor Skill

Execute tasks autonomously using the taskrunner engine.

## When to Use

Use this skill when:
- A task can be executed without complex planning
- You want one-shot execution with automatic error recovery
- The task is straightforward (create file, run command, fix typo)

## When NOT to Use

Do NOT use this skill when:
- The task requires architectural decisions
- Multiple stakeholders need to review the approach
- The change affects critical systems and needs documentation

## Commands

- `/execute "task description"` → Execute a task autonomously
- `/execute-verbose "task description"` → Execute with detailed output

## Integration with Gentleman Mode

When in Gentleman mode and the user asks for something:

1. **Detect complexity** using `autonomous.DetectComplexity(task)`
2. **If simple** → Use `/execute` (taskrunner - one loop)
3. **If complex** → Use `/sdd-new` (full SDD with mini-loops)

Example:
```
User: "fix the typo in readme"
→ Detect: simple
→ Action: /execute "fix typo in readme"

User: "redesign the auth system"  
→ Detect: complex
→ Action: /sdd-new "redesign auth system"
```

## Integration with SDD Orchestrator

When in SDD mode, use this skill for:
- **Explore phase**: Quick codebase exploration
- **Apply phase**: Implementation of straightforward tasks
- **Any phase**: When the phase task is simple enough for one-shot execution

## Usage

```bash
# From within Claude Code
!gentle-ai task "description"

# Or via the skill directly
/execute "create a Python script that fetches weather data"
```

## Output

The executor will:
1. Run the task autonomously
2. Show progress (if verbose)
3. Return a summary report
4. Save lessons learned to Engram (if configured)

## Error Handling

- If the task fails, the executor retries with different approaches
- Errors are logged and saved as lessons
- The user receives a clear report of what was attempted

## Configuration

The executor auto-detects available AI engines:
- Claude Code (priority 1)
- OpenCode (priority 2)
- Gemini CLI (priority 3)
- Codex (priority 4)

To force a specific engine:
```
/execute --engine claude-code "task description"
```
