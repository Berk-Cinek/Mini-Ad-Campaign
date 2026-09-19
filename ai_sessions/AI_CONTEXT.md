# AI Context

CLAUDE.md is the context file Claude Code read at the start of every session. I wrote the decisions in it. stack, hard rules (budget invariants, soft delete, integer money), domain decisions, status transitions, the API surface, and the rules for how the AI should work with me. I used a separate planning chat to helpphrase them precisely and to challenge them, but the decisions are mine.

The file was then reviewed by AI twice for gaps, contradictions and ambiguous rules. I accepted findings on domain decisions and hard rules and rejected others. I treated it as a living document, updating it as the project developed
rather than writing it once at the start.

The Open Questions section is deliberate. I used it to mark the parts I wanted to decide myself rather than have the AI decide for me. The clearest example is the race-condition mechanism, which I left undecided so Claude Code would have to
propose options with trade-offs. What it proposed, and what I chose, is covered in AI_WORKFLOW.md.