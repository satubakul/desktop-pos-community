# Agent instructions

## KNL-View and AnchorDB

Run KNL-View commands that access the local AnchorDB daemon outside the sandbox. The sandbox can hide the local Podman container and loopback ports, causing `knl-view anchordb status` to report that nothing is detected even while AnchorDB is running.

Use the command runner with `sandbox_permissions: "require_escalated"` and a short, specific `justification`. Keep the command's working directory set to this project. For example:

```json
{
  "cmd": "knl-view anchordb status",
  "workdir": "/home/natadana/rahman-project/kasir-kita",
  "sandbox_permissions": "require_escalated",
  "justification": "May I check the local AnchorDB daemon outside the sandbox?"
}
```

Apply the same outside-sandbox execution to KNL-View traces and synchronization that need AnchorDB. If escalation is denied, report the limitation and do not interpret an in-sandbox `nothing detected` result as proof that the daemon is stopped. Use KNL-View's documented commands for graph access; do not connect to AnchorDB directly.

<!-- knl-view:start -->
# Knowledge graph tools
- KNL-View is the primary graph tool. For `.knowledges`, source graphs, external module graphs, and project graph questions, read the project's `knl-view` skill and use the relevant KNL-View feature first.
- Graphify is the fallback. Use it only when a specific need remains unresolved by KNL-View or the user explicitly requests Graphify. If KNL-View answers the task, stop; do not run Graphify for duplicate analysis.
- When the user explicitly invokes `/graphify`, honor that request without automatically running both tools.

## Strict Anti-Grep & Mandatory Trace-Before-Edit Protocol (Universal Enforcement)
- **STRICT BAN ON AD-HOC GREP/FIND FOR CODE DISCOVERY**:
  - `knl-view sources trace --target=<path>#<symbol>` or `knl-view sources trace --target=<path>` is the MANDATORY primary tool for navigating codebase architecture, caller hierarchies, and dependencies.
  - Blind ad-hoc `grep`/`find` as a lazy substitute for architectural graph navigation is strictly forbidden.
  - `grep`/`find` is permitted for literal text searches (exact error messages, literal UI strings, or config values), and for cross-verifying call sites when a symbol trace returns `in: []` due to dynamic receiver dispatch.
  - **CRITICAL AUDIT BOUNDARY (DYNAMIC RECEIVER BOUNDARY)**: `in: []` on a symbol trace does NOT prove zero callers; dynamic or variable receiver calls (`obj.Method()`) may be skipped as `local-receiver`. When auditing symbol connectivity, agents MUST cross-verify `in: []` by reading the code and using targeted literal search (`grep -rn <symbol>`) before concluding a symbol is uncalled.
- **MANDATORY THREE-PHASE KNL-VIEW WORKFLOW**:
  1. **Pre-Edit Exploration (MANDATORY)**: Run `knl-view sources trace --target=<path>` (file lineage) or `knl-view sources trace --target=<path>#<symbol>` (call hierarchy) BEFORE touching or modifying files.
  2. **Deliberate Batch Edit**: Apply changes based strictly on semantic graph relationships mapped by KNL-View.
  3. **Post-Edit State Synchronization (MANDATORY)**: Run `knl-view sources sync` immediately after all file edits and test runs are complete.

## KNL-View graph access
- Use the KNL-View feature that fits the task (`graph trace`, `sources trace/sync/collect`, `modules build-graph/trace/list`, `model check/new`, consensus commands, or the viewer). Do not approximate a supported KNL-View operation with ad-hoc graph queries.
- For third-party libraries and dependencies (`node_modules`, `go mod`, `site-packages`, `crates`): NEVER read raw library files directly. Build and query the module graph on-demand: run `knl-view modules build-graph <eco>::<module_name>` (e.g. `npm::express`, `go::github.com/gin-gonic/gin`), then inspect API definitions and call hierarchies via `knl-view modules trace <eco>::<module_name> [symbol]` to save context tokens.
- Route all graph reads and writes through KNL-View's documented CLI, viewer, or application endpoints. Never connect to AnchorDB directly through AnchorWire, its CLI, direct database endpoints, files, SQL, or custom clients/scripts. If KNL-View cannot perform a requested operation, explain the limitation; do not bypass it.
- Keep graph work targeted: specify the relevant node/path, direction, and smallest useful hop count; summarize only relevant nodes and edges; avoid dumping whole graphs or large raw JSON.
- `in: []` on a symbol trace does not mean "no callers" until the function body has been read. The `resolution` field explains which call sites the resolver skipped and why.
- The graph organizes relations, not behavior. Behavior claims are verified by reading the code and running tests, not by querying the graph.
- Follow the KNL-View skill's trace-before-edit and `sources sync`-after-change rules. Treat edge direction as lineage; do not claim causality without separate evidence.
<!-- knl-view:end -->
