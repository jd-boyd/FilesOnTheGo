# FilesOnTheGo Implementation Plan - Dependency Matrix

## Parallel Execution Strategy

This plan is optimized to keep **4 agents busy concurrently** throughout the implementation.

## Dependency Groups

### Group 1: Foundation (1 step)
**Can start immediately**

| Step | Description | Duration Est. |
|------|-------------|---------------|
| 01 | Project scaffolding and PocketBase setup | 30 min |

### Group 2: Core Services (4 steps in parallel)
**Dependencies: Group 1 complete**
**Recommended: Run all 4 in parallel**

| Step | Description | Dependencies | Duration Est. |
|------|-------------|--------------|---------------|
| 02 | S3 service implementation | Step 01 | 45 min |
| 03 | Database models/collections setup | Step 01 | 30 min |
| 04 | Permission service implementation | Step 01 | 45 min |
| 05 | Basic HTMX UI layout | Step 01 | 30 min |

### Group 3: Business Logic (4 steps in parallel)
**Dependencies: Group 2 complete**
**Recommended: Run all 4 in parallel**

| Step | Description | Dependencies | Duration Est. |
|------|-------------|--------------|---------------|
| 06 | File upload handler | Steps 02, 03, 04 | 60 min |
| 07 | File download handler | Steps 02, 03, 04 | 45 min |
| 08 | Directory management | Steps 03, 04 | 45 min |
| 09 | Share service implementation | Steps 03, 04 | 60 min |

### Group 4: Frontend Components (4 steps in parallel)
**Dependencies: Group 3 complete**
**Recommended: Run all 4 in parallel**

| Step | Description | Dependencies | Duration Est. |
|------|-------------|--------------|---------------|
| 10 | File browser UI component | Steps 05, 06, 07, 08 | 45 min |
| 11 | Upload UI component | Steps 05, 06 | 30 min |
| 12 | Share creation UI | Steps 05, 09 | 45 min |
| 13 | Public share page | Steps 05, 07, 09 | 45 min |

### Group 5: Quality Assurance (3 steps, 2-3 in parallel)
**Dependencies: Groups 1-4 complete**
**Recommended: Run Steps 14 & 15 in parallel, then 16**

| Step | Description | Dependencies | Duration Est. |
|------|-------------|--------------|---------------|
| 14 | Integration tests | All steps 01-13 | 60 min |
| 15 | Security tests | All steps 01-13 | 45 min |
| 16 | Documentation & deployment | Steps 14, 15 | 30 min |

## Execution Timeline (Optimized for 4 Parallel Agents)

```
Time    Agent 1         Agent 2         Agent 3         Agent 4
-----   -------------   -------------   -------------   -------------
Hour 1  Step 01 ──────────────────────────────────────> (All agents wait)

Hour 2  Step 02         Step 03         Step 04         Step 05
        (S3 Service)    (Models)        (Permissions)   (UI Layout)

Hour 3  Step 06         Step 07         Step 08         Step 09
        (Upload)        (Download)      (Directories)   (Shares)

Hour 4  Step 10         Step 11         Step 12         Step 13
        (File Browser)  (Upload UI)     (Share UI)      (Public Page)

Hour 5  Step 14         Step 15         (idle)          (idle)
        (Integration)   (Security)

Hour 6  Step 16 ─────> (idle)          (idle)          (idle)
        (Docs)
```

## Critical Path

The critical path through the project is:
**Step 01 → Step 02 → Step 06 → Step 10 → Step 14 → Step 16**

Total estimated time: **~6 hours** with 4 parallel agents

## Dependency Graph (Visual)

```
                    ┌─────────┐
                    │ Step 01 │
                    │ Project │
                    │  Setup  │
                    └────┬────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌────▼────┐    ┌────▼────┐    ┌────────┐
    │ Step 02 │    │ Step 03 │    │ Step 04 │    │Step 05 │
    │   S3    │    │ Models  │    │  Perms  │    │UI Base │
    └────┬────┘    └────┬────┘    └────┬────┘    └────┬───┘
         │              │              │              │
         └──────┬───────┴──────┬───────┘              │
                │              │                      │
         ┌──────▼─────┐  ┌────▼─────┐  ┌────────┐   │
         │  Step 06   │  │ Step 07  │  │Step 08 │   │
         │   Upload   │  │ Download │  │  Dirs  │   │
         └──────┬─────┘  └────┬─────┘  └────┬───┘   │
                │             │             │        │
                └──────┬──────┴─────┬───────┘        │
                       │            │                │
                  ┌────▼────┐  ┌───▼─────┐          │
                  │ Step 09 │  │         │          │
                  │ Shares  │  │         │          │
                  └────┬────┘  │         │          │
                       │       │         │          │
         ┌─────────────┴───────┴─────────┴──────────┘
         │             │             │             │
    ┌────▼────┐  ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
    │ Step 10 │  │ Step 11 │  │ Step 12 │  │ Step 13 │
    │FileBrwsr│  │UploadUI │  │ ShareUI │  │PublicPg │
    └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘
         │            │            │            │
         └────────────┴────────────┴────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
    │ Step 14 │      │ Step 15 │      │ Step 16 │
    │Integr.  │      │Security │      │  Docs   │
    └─────────┘      └─────────┘      └─────────┘
```

## Parallel Execution Commands

### Group 1
```bash
# Run Step 01 first (single agent required)
```

### Group 2
```bash
# Start all 4 in parallel after Group 1 completes:
# Agent 1: Step 02
# Agent 2: Step 03
# Agent 3: Step 04
# Agent 4: Step 05
```

### Group 3
```bash
# Start all 4 in parallel after Group 2 completes:
# Agent 1: Step 06
# Agent 2: Step 07
# Agent 3: Step 08
# Agent 4: Step 09
```

### Group 4
```bash
# Start all 4 in parallel after Group 3 completes:
# Agent 1: Step 10
# Agent 2: Step 11
# Agent 3: Step 12
# Agent 4: Step 13
```

### Group 5
```bash
# Start Steps 14 & 15 in parallel after Group 4 completes:
# Agent 1: Step 14
# Agent 2: Step 15
# Then Step 16 after both complete
```

## Notes

- **Always complete an entire dependency group before starting the next group**
- Each step includes comprehensive tests (as per CLAUDE.md requirements)
- Security considerations are embedded in each step
- All code must follow the guidelines in CLAUDE.md
- Minimum 80% test coverage required for all packages
