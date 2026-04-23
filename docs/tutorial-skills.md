# Tutorial: Creating Skills

Skills are Markdown files that provide `gollm` with specialized, reusable instructions for specific tasks. When a skill is invoked, its content is sent as a user message to the agent along with any arguments you provide.

---

## How Skills Work

When `gollm` starts, it scans the skill directories and adds a list of available skills to the system prompt. The agent knows which skills exist and their descriptions. You can explicitly invoke a skill with `/skill:<name>` from the TUI, or the agent may suggest using one automatically.

When you invoke a skill, its Markdown content is wrapped in a `<skill>` tag and sent to the agent:

```
<skill name="refactor" location="/path/to/refactor/SKILL.md">
References are relative to /path/to/refactor/.

...skill content here...
</skill>

your additional arguments here
```

---

## Skill Discovery Directories

`gollm` searches for skills in these locations (in order):

| Path | Scope |
|---|---|
| `~/.gollm/skills/` | Global — available in all projects |
| `.gollm/skills/` (project root) | Project-specific skills |

Skills with the same name in a project directory override global ones.

---

## Skill File Formats

### Simple: Single `.md` file

Create a `.md` file directly in a skills directory. The filename (without extension) becomes the skill name.

```
.gollm/skills/refactor.md
```

Invoke with:
```
/skill:refactor improve error handling
```

### Structured: Directory with `SKILL.md`

Create a directory containing a `SKILL.md` file. The **directory name** becomes the skill name. This format lets you include supporting files (examples, templates) alongside the skill.

```
.gollm/skills/
  code-review/
    SKILL.md
    checklist.md
    examples/
      before.go
      after.go
```

Invoke with:
```
/skill:code-review
```

> **Note:** When a `SKILL.md` is found in a directory, subdirectories are not scanned further. This lets you bundle reference files with your skill.

---

## Frontmatter (Optional)

Both formats support optional YAML frontmatter to provide metadata:

```markdown
---
name: refactor
description: Refactor Go code to use idiomatic patterns and interfaces
---

You are an expert Go developer. When asked to refactor code:

1. Identify opportunities to use interfaces for testability
2. Replace repetitive code with helper functions
3. Add godoc comments to all exported symbols
4. Ensure error handling follows Go conventions (wrap with %w)

Always explain the reasoning behind each change before making it.
```

**Frontmatter fields:**

| Field | Description |
|---|---|
| `name` | Override the skill name (defaults to filename/directory name) |
| `description` | A short description shown to the agent in the system prompt |

---

## Practical Examples

### Example 1: Code Review Skill

```
.gollm/skills/code-review.md
```

```markdown
---
name: code-review
description: Perform a thorough code review with actionable feedback
---

Review the provided code and evaluate it against these criteria:

**Correctness**
- Does the logic match the intended behavior?
- Are edge cases handled?
- Are there potential nil pointer dereferences or index out-of-bounds issues?

**Maintainability**
- Is the code readable and self-documenting?
- Are functions focused on a single responsibility?
- Is there appropriate error handling?

**Performance**
- Are there obvious inefficiencies (e.g. unnecessary allocations, N+1 queries)?

Format your response as:
## Summary
<one paragraph>

## Issues
<numbered list of specific issues with file:line references>

## Suggestions
<numbered list of improvements>
```

Invoke:
```
/skill:code-review
```
Or attach a file reference:
```
/skill:code-review @[internal/agent/loop.go]
```

---

### Example 2: Structured Skill with Supporting Files

```
.gollm/skills/
  db-migration/
    SKILL.md
    schema-example.sql
```

```markdown
---
name: db-migration
description: Generate SQL migration files following our project conventions
---

Generate a database migration for the requested schema change.

Our migration file conventions:
- Files are named: `YYYYMMDD_HHMMSS_description.sql`
- Each file has an `-- +migrate Up` and `-- +migrate Down` section
- All tables use `BIGINT` primary keys with `AUTO_INCREMENT`
- Always include `created_at` and `updated_at` TIMESTAMP columns

See the example schema at the path listed in this skill's location directory: `schema-example.sql`
```

---

### Example 3: Global Utility Skill

```
~/.gollm/skills/explain.md
```

```markdown
---
name: explain
description: Explain code clearly for a non-expert audience
---

Explain the following code in plain English. Assume the reader is a competent programmer but unfamiliar with this codebase.

Structure your explanation as:
1. **Purpose** — What does this code do in one sentence?
2. **How it works** — Step-by-step walkthrough of the logic
3. **Key concepts** — Any domain-specific terms or patterns used
4. **Gotchas** — Anything surprising or non-obvious
```

---

## Tips

- **Keep skills focused.** One skill = one task type. Compose them with arguments rather than making a single skill do everything.
- **Use relative file references** — when your skill body references files, note they resolve relative to the skill's directory. The agent is told the skill's location so it can use the `read` tool on supporting files.
- **Test your skill** by invoking it with `/skill:<name>` in the TUI. The exact text sent to the agent is shown in the chat history.
- **Override skills per-project** — place a skill with the same name in `.gollm/skills/` to override the global version for a specific project.
