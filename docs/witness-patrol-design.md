# Witness Patrol: Theory of Operation

## Overview

The Witness is the per-rig worker monitor. It watches polecats, nudges them toward
completion, verifies clean state before cleanup, and escalates stuck workers.

**Key principle: Claude-driven execution.** The mol-witness-patrol molecule is a
playbook that Claude reads and executes. There is no Go "runtime" that auto-executes
steps. Claude provides the intelligence; gt/bd commands provide the primitives.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        THE AGENT (Claude)                       │
│                                                                 │
│   Reads molecule steps → Executes commands → Closes atoms       │
│   Uses TodoWrite for complex atoms (optional)                   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                     PRIMITIVES (gt, bd CLI)                     │
│                                                                 │
│   gt mail, gt nudge, gt session, bd close, bd show, etc.        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      COORDINATION (Mail)                        │
│                                                                 │
│   Polecats → POLECAT_DONE → Witness inbox                       │
│   Witness → "You're stuck" → Polecat (via gt nudge)             │
│   Witness → Escalation → Mayor inbox                            │
└─────────────────────────────────────────────────────────────────┘
```

## The Patrol Cycle

Each patrol cycle follows the mol-witness-patrol molecule:

### 1. inbox-check
Check mail for lifecycle events:
- **POLECAT_DONE**: Polecat finished work, ready for cleanup
- **Help requests**: Polecat asking for assistance
- **Escalations**: Issues requiring attention

```bash
gt mail inbox
gt mail read <id>
```

### 2. survey-workers
For each polecat in the rig:

```bash
gt polecat list <rig>
```

For each polecat:
1. **Capture**: `tmux capture-pane -t gt-<rig>-<name> -p | tail -50`
2. **Assess**: Claude reads output, determines state (working/idle/error/done)
3. **Load history**: Read nudge count from handoff bead
4. **Decide**: Apply escalation matrix (see below)
5. **Execute**: Take action (none, nudge, escalate, cleanup)

### 3. save-state
Persist state to handoff bead for next cycle:
- Nudge counts per polecat
- Last nudge timestamps
- Pending actions

### 4. burn-or-loop
- If context low: sleep briefly, loop back to inbox-check
- If context high: exit (daemon respawns fresh Witness)

## Nudge Escalation Matrix

The Witness applies escalating pressure to idle polecats:

| Idle Time | Nudge Count | Action |
|-----------|-------------|--------|
| <10min | any | none |
| 10-15min | 0 | Gentle: "How's progress?" |
| 15-20min | 1 | Direct: "Please wrap up. What's blocking?" |
| 20+min | 2 | Final: "Will escalate in 5min if no response." |
| any | 3 | Escalate to Mayor |

**Key insight**: Only Claude can assess whether a polecat is truly stuck.
Looking at tmux output requires understanding context:
- "I'm stuck on this error" → needs help
- "Running tests..." → actively working
- Sitting at prompt with no activity → maybe stuck

## State Persistence

The Witness handoff bead tracks:

```yaml
# In handoff bead description
nudges:
  toast:
    count: 2
    last: "2025-12-24T10:30:00Z"
  ace:
    count: 0
    last: null
pending_cleanup:
  - nux  # received POLECAT_DONE, queued for verification
```

This survives across patrol cycles and context burns.

## Polecat Cleanup Flow

When a polecat signals completion:

1. Polecat runs `gt done` or sends POLECAT_DONE mail
2. Witness receives mail in inbox-check
3. Witness runs pre-kill verification:
   ```bash
   cd polecats/<name>
   git status              # Must be clean
   git log origin/main..   # Check for unpushed
   bd show <issue>         # Verify closed
   ```
4. If clean: kill session, remove worktree, delete branch
5. If dirty: send nudge asking polecat to fix state

## What We DON'T Need

- **Go patrol runtime**: Claude executes the playbook
- **Polling for WaitsFor**: Mail tells us when things are ready
- **Automated health checks**: Claude reads tmux, assesses
- **Go nudge logic**: Claude applies the matrix

## What We DO Need

- **mol-witness-patrol**: The playbook (exists)
- **Handoff bead**: State persistence (gt-poxd)
- **CLI primitives**: gt mail, gt nudge, gt session (exist)
- **Molecule tracking**: bd close for step completion (exists)

## Related Issues

- gt-poxd: Create handoff beads for Witness and Refinery roles
- gt-y481: Patrol parity - Witness and Refinery match Deacon sophistication
- gt-tnow: Implement Christmas Ornament pattern for mol-witness-patrol
