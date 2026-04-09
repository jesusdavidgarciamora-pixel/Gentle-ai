# SDD Phase — Common Protocol

Boilerplate identical across all SDD phase skills. Sub-agents MUST load this alongside their phase-specific SKILL.md.

Executor boundary: every SDD phase agent is an EXECUTOR, not an orchestrator. Do the phase work yourself. Do NOT launch sub-agents, do NOT call `delegate`/`task`, and do NOT bounce work back unless the phase skill explicitly says to stop and report a blocker.

## A. Skill Loading

1. Check if the orchestrator injected a `## Project Standards (auto-resolved)` block in your launch prompt. If yes, follow those rules — they are pre-digested compact rules from the skill registry. **Do NOT read any SKILL.md files.**
2. If no Project Standards block was provided, check for `SKILL: Load` instructions. If present, load those exact skill files.
3. If neither was provided, search for the skill registry as a fallback:
   a. `mem_search(query: "skill-registry", project: "{project}")` — if found, `mem_get_observation(id)` for full content
   b. Fallback: read `.atl/skill-registry.md` from the project root if it exists
   c. From the registry's **Compact Rules** section, apply rules whose triggers match your current task.
4. If no registry exists, proceed with your phase skill only.

NOTE: the preferred path is (1) — compact rules pre-injected by the orchestrator. Paths (2) and (3) are fallbacks for backwards compatibility. Searching the registry is SKILL LOADING, not delegation. If `## Project Standards` is present, IGNORE any `SKILL: Load` instructions — they are redundant.

## B. Artifact Retrieval (Engram Mode)

**CRITICAL**: `mem_search` returns 300-char PREVIEWS, not full content. You MUST call `mem_get_observation(id)` for EVERY artifact. **Skipping this produces wrong output.**

**Run all searches in parallel** — do NOT search sequentially.

```
mem_search(query: "sdd/{change-name}/{artifact-type}", project: "{project}") → save ID
```

Then **run all retrievals in parallel**:

```
mem_get_observation(id: {saved_id}) → full content (REQUIRED)
```

Do NOT use search previews as source material.

## C. Artifact Persistence

Every phase that produces an artifact MUST persist it. Skipping this BREAKS the pipeline — downstream phases will not find your output.

### Engram mode

```
mem_save(
  title: "sdd/{change-name}/{artifact-type}",
  topic_key: "sdd/{change-name}/{artifact-type}",
  type: "architecture",
  project: "{project}",
  content: "{your full artifact markdown}"
)
```

`topic_key` enables upserts — saving again updates, not duplicates.

### OpenSpec mode

File was already written during the phase's main step. No additional action needed.

### Hybrid mode

Do BOTH: write the file to the filesystem AND call `mem_save` as above.

### None mode

Return result inline only. Do not write any files or call `mem_save`.

## D. Return Envelope

Every phase MUST return a structured envelope to the orchestrator:

- `status`: `success`, `partial`, or `blocked`
- `executive_summary`: 1-3 sentence summary of what was done
- `detailed_report`: (optional) full phase output, or omit if already inline
- `artifacts`: list of artifact keys/paths written
- `next_recommended`: the next SDD phase to run, or "none"
- `risks`: risks discovered, or "None"
- `skill_resolution`: how skills were loaded — `injected` (received Project Standards from orchestrator), `fallback-registry` (self-loaded from registry), `fallback-path` (loaded via SKILL: Load path), or `none` (no skills loaded)

Example:

```markdown
**Status**: success
**Summary**: Proposal created for `{change-name}`. Defined scope, approach, and rollback plan.
**Artifacts**: Engram `sdd/{change-name}/proposal` | `openspec/changes/{change-name}/proposal.md`
**Next**: sdd-spec or sdd-design
**Risks**: None
**Skill Resolution**: injected — 3 skills (react-19, typescript, tailwind-4)
(other values: `fallback-registry`, `fallback-path`, or `none — no registry found`)
```

## E. Code Search Protocol

Phases that investigate or read existing code (`sdd-explore`, `sdd-apply`, `sdd-verify`) MUST follow this protocol instead of ad-hoc Grep+Read.

### Reading the config

```
Read search_strategy from project context:
├── engram: included in sdd-init/{project} observation → search_strategy block
├── openspec: openspec/config.yaml → search_strategy key
└── Not found → default to mode=grep (current behavior, zero change)
```

### Mode behavior

| Mode | Behavior |
|------|----------|
| `grep` | Current behavior — Grep + Read. No RAG tool calls. Default when config is absent. |
| `hybrid` | RAG-first cascade with grep fallback. Requires `search_strategy.rag.mcp_tool`. |

If `mode` is `hybrid` but `rag.mcp_tool` is missing or empty, fall back to `grep` and include a warning in the return envelope's `risks` field.

### Hybrid cascade

```
1. Semantic query → call configured mcp_tool with a natural-language description
   of what you're looking for
   ├── Results sufficient (≥2 relevant snippets)? → use them, done
   └── Results insufficient or empty?
       ↓
2. Exact search → Grep for symbols, patterns, or file paths
   ├── Context clear from matches? → done
   └── Need surrounding context?
       ↓
3. Targeted read → Read with specific line range (NOT full files)
```

Each step is strictly additive — the cascade never produces worse results than grep-only.

### Grep-only path (mode=grep or missing)

Use Grep to find relevant code, then Read with targeted line ranges. This is identical to current behavior. No RAG tool is called.

### When RAG tool is unavailable at runtime

If `mode` is `hybrid` but the MCP tool call fails (tool not found, server down, timeout):
1. Log the failure
2. Fall back to grep behavior for the remainder of this phase
3. Include in return envelope `risks`: "RAG search unavailable, fell back to grep — consider checking MCP server status"

### Reindex hook (sdd-apply only)

After completing artifact persistence in sdd-apply, IF `search_strategy.rag.reindex_tool` is configured:
1. Call `reindex_tool` once with the list of files modified in this batch
2. Fire-and-forget — do NOT wait for completion, do NOT block on the result
3. If the call fails, log it and continue — grep fallback guarantees correctness

Do NOT call reindex per individual file write. One call per batch only.
