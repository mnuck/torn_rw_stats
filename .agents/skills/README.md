# Agent Skills

This directory is the canonical home for project skills — reusable, named
procedures that AI coding agents can invoke (e.g. via slash commands).

Each skill lives in its own subdirectory containing a `SKILL.md` that
describes when and how to use it, plus any scripts or templates the skill
needs:

```
.agents/skills/
  my-skill/
    SKILL.md
    generate_something.sh
```

Harness-specific discovery paths point here rather than duplicating
content: `.claude/skills` is a symlink to this directory, so Claude Code
finds these skills automatically, and other harnesses can do the same.
