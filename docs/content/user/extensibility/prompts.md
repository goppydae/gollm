---
title: Prompt Templates
weight: 20
description: Reusable editor templates invoked with /prompt:name
categories: [extensions]
tags: [prompts]
---

Prompt templates are reusable text snippets that expand directly into the TUI input editor. Unlike skills (which are sent to the agent immediately), prompt templates let you pre-fill the editor so you can review, edit, or complete the text before sending.

---

## How Prompt Templates Work

When you type `/prompt:<name>` and press Enter, the template content is loaded into the editor input. You can then modify it, add context, attach files with `@`, and send it normally. This is useful for long, structured prompts you use frequently.

---

## Prompt Template Directories

`gollm` searches these locations (in order):

| Path | Scope |
|---|---|
| `~/.gollm/prompts/` | Global — available in all projects |
| `.gollm/prompts/` (project root) | Project-specific templates |

---

## Template File Format

A prompt template is any `.md` file in a prompts directory. The filename (without extension) is the template name.

```
.gollm/prompts/bug-report.md
```

Invoke with:
```
/prompt:bug-report
```

### Minimal Template (no frontmatter)

The entire file content becomes the template text:

```markdown
Describe the bug you found:

**Steps to reproduce:**
1.
2.
3.

**Expected behavior:**

**Actual behavior:**

**Environment:**
- OS:
- glm version:
- Model:
```

### Template with Frontmatter

Add optional YAML frontmatter for metadata:

```markdown
---
description: Generate a structured bug report
argument-hint: <component-name>
---

Describe the bug you found in the $1 component:

**Steps to reproduce:**
1.
2.
3.

**Expected behavior:**

**Actual behavior:**
```

**Frontmatter fields:**

| Field | Description |
|---|---|
| `description` | Short description shown in the `/prompt:` picker |
| `argument-hint` | Hint shown in autocomplete describing expected arguments |

---

## Argument Substitution

Templates support positional argument placeholders: `$1`, `$2`, etc.

When you invoke a template via the slash command handler (not the interactive TUI), arguments after the template name are substituted. To mitigate prompt injection, `gollm` automatically wraps these arguments in `<untrusted_input>` tags. In the TUI, the template expands as-is and you fill in the values manually.

---

## Practical Examples

### PR Description Template

`.gollm/prompts/pr-description.md`

```markdown
---
description: Generate a pull request description
---

Write a pull request description for the following changes.

**Format:**
## Summary
<What does this PR do? Why?>

## Changes
<Bullet list of specific changes>

## Testing
<How was this tested?>

## Notes
<Anything reviewers should pay attention to>

The diff is:
```

Invoke:
```
/prompt:pr-description
```
Then paste or attach the diff before sending.

---

### Architecture Decision Record

`.gollm/prompts/adr.md`

```markdown
---
description: Draft an Architecture Decision Record (ADR)
argument-hint: <decision-title>
---

Draft an Architecture Decision Record (ADR) for: **$1**

Use this structure:

# ADR: $1

## Status
Proposed

## Context
<What is the issue motivating this decision?>

## Decision
<What was decided?>

## Consequences
### Positive
-

### Negative
-

### Neutral
-

## Alternatives Considered
<What other approaches were evaluated and why were they rejected?>
```

Invoke:
```
/prompt:adr Use JSONL for session storage
```

---

### Global Commit Message Template

`~/.gollm/prompts/commit.md`

```markdown
---
description: Generate a conventional commit message
---

Generate a commit message following the Conventional Commits specification for the following diff or description of changes.

Format:
` ` `
<type>(<scope>): <short description>

<body: what changed and why, wrapped at 72 chars>

<footer: breaking changes, issue references>
` ` `

Types: feat, fix, docs, style, refactor, perf, test, chore

Changes:
```

Invoke:
```
/prompt:commit
```

---

### Code Explanation for PR Comments

`.gollm/prompts/explain-for-review.md`

```markdown
---
description: Explain a code block suitable for a PR comment
---

Explain the following code in a way that's suitable for a GitHub PR review comment. Be concise (2-4 sentences max), assume the reader is a senior engineer, and highlight any non-obvious design decisions.

Code:
```

---

## Tips

- **Prompt templates are for your input.** They expand into the editor, not directly to the agent. This gives you a chance to customize before sending.
- **Use `$1`, `$2` placeholders** for dynamic parts you'll always fill in differently. Leave static boilerplate as literal text.
- **Combine with `@` file attachments.** Type `/prompt:code-review` then add `@src/myfile.go` before pressing Enter to attach a file.
- **Project-specific overrides.** A template in `.gollm/prompts/` with the same name as a global template takes priority for that project.
- **Organize with subdirectories.** Templates are discovered recursively, so you can group them:
  ```
  .gollm/prompts/
    code/
      refactor.md
      review.md
    docs/
      readme.md
      adr.md
  ```
  Invoke as `/prompt:refactor`, `/prompt:adr`, etc. (name is the filename, not the full path).
