# tokenprune: Architecture and Engineering Master Plan
> Real-Time Semantic Context Compression and Streaming Reverse Proxy for Long-Horizon AI Agents
> Target: High-Performance Single-Binary Go Proxy (< 20MB RAM, < 10ms overhead) with Embedded React 19 Telemetry Studio

---

## 1. Problem Statement and Motivation

### 1.1 Context Window Bloat in Autonomous Agents
Autonomous coding and research agents (such as Claude Code, Aider, and custom agent loops) generate massive conversation histories across multi-step tasks. In typical sessions, context grows from 8k tokens to over 100k tokens within 15 to 30 turns.

This expansion creates three primary bottlenecks:
1. High Time to First Token (TTFT): Contexts above 80k tokens introduce 4 to 12 seconds of prefill latency per turn before the model produces its first token.
2. Compounding API Expenses: Re-transmitting historical turns with each request causes cost to scale quadratically with session length.
3. Attention Degradation: As context increases, attention mechanisms exhibit degraded retrieval accuracy for facts located in the middle of prompts.

### 1.2 Naive Summarization Failures
Standard conversational summarizers condense prompts by asking an LLM to rewrite history. This introduces severe flaws in agentic execution:
- Syntactic Destruction: Parameter names, file paths, line numbers, and JSON schema keys are often altered or dropped.
- State Drift: Older file contents are blurred into generic descriptions, preventing agents from generating accurate code diffs.
- Extra Latency and Cost: Running a full model call to summarize past turns adds latency and cost to the critical path.

### 1.3 The tokenprune Solution
tokenprune is a lightweight reverse proxy written in Go that intercepts requests between agent harnesses and LLM endpoints. It implements a three-tier pruning pipeline that operates locally on CPU in under 10 milliseconds:
- Tier 1 (Syntactic Stripping): Lossless minification of embedded JSON payloads, elimination of markdown table padding, removal of HTML comments, and normalization of repetitive whitespace while protecting code blocks.
- Tier 2 (Ephemeral Tool Output Eviction): Tracks tool outputs over time. Stale, superseded file reads and command dumps are replaced with deterministic semantic receipts that record target names, line counts, byte sizes, and exit codes. Error traces and active file context are preserved.
- Tier 3 (State Delta Compression): Older dialogue turns are consolidated into structured state transition records. System prompt invariants and recent working memory remain untouched.
- Adaptive Budget Controller: Dynamically regulates pruning intensity to guarantee that outbound prompts adhere to user-configured token ceilings or target compression ratios.

---

## 2. High-Level System Architecture

```
       +--------------------------------------------------------+
       |               AI Agent Application / CLI               |
       |         (Claude Code, Aider, OpenCode, AutoGen)        |
       +--------------------------------------------------------+
                                   |
                                   | HTTP / SSE Request (e.g. 100k Tokens)
                                   v
       +--------------------------------------------------------+
       |                   tokenprune Proxy                     |
       |          (Single Go Binary, Sub-10ms Overhead)         |
       +--------------------------------------------------------+
                                   |
                     +-------------+-------------+
                     |                           |
                     v                           v
     +-------------------------------+   +-------------------------------+
     |  Syntactic Stripper (Tier 1)  |   |  Tool Eviction Engine (Tier 2)|
     | - Minifies JSON payloads      |   | - Detects superseded reads    |
     | - Normalizes whitespace       |   | - Generates verified receipts |
     | - Cleans HTML/Markdown bloat  |   | - Preserves error traces      |
     +-------------------------------+   +-------------------------------+
                     |                           |
                     +-------------+-------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |           State Delta Compressor (Tier 3)              |
       |  - Consolidates turns older than N steps               |
       |  - Anchors system invariants and active work items     |
       |  - Enforces max context token budget                   |
       +--------------------------------------------------------+
                                   |
                                   | Forward Pruned Payload (30k Tokens)
                                   v
       +--------------------------------------------------------+
       |              Upstream LLM Provider                     |
       |        (Anthropic, OpenAI, Gemini, Ollama)             |
       +--------------------------------------------------------+
                                   |
                                   | Streaming SSE Tokens
                                   v
       +--------------------------------------------------------+
       |               React 19 Telemetry Studio                |
       |  - Real-time savings fuel gauge                        |
       |  - Turn waterfall chart and side-by-side diff viewer   |
       |  - Real-time SSE updates and interactive sandbox       |
       +--------------------------------------------------------+
```

---

## 3. Mathematical Compression Formulation

For an inbound prompt $P$ with token count $T_{\text{in}}$ and pruned prompt $P'$ with token count $T_{\text{out}}$:

$$\text{Compression Ratio (CR)} = \frac{T_{\text{in}} - T_{\text{out}}}{T_{\text{in}}} \times 100\%$$

$$\text{Cost Savings (USD)} = (T_{\text{in}} - T_{\text{out}}) \times \text{PricePerInputToken}(M)$$

$$\text{Latency Reduction Factor} = \frac{\text{TTFT}(T_{\text{in}})}{\text{TTFT}(T_{\text{out}})} \approx \frac{T_{\text{in}}}{T_{\text{out}}}$$

tokenprune maintains a strict fidelity condition:
$$\text{TaskAccuracy}(P') \ge 0.99 \times \text{TaskAccuracy}(P)$$

---

## 4. Repository Structure

```
tokenprune/
├── .github/
│   └── workflows/
│       ├── test.yml
│       └── release.yml
├── cmd/
│   └── tokenprune/
│       └── main.go                 # CLI entry point (serve, bench, prune, report, version)
├── internal/
│   ├── config/                     # YAML configuration loader and environment expansion
│   ├── proxy/                      # Streaming HTTP and SSE reverse proxy
│   │   ├── server.go
│   │   ├── openai_adapter.go
│   │   └── anthropic_adapter.go
│   ├── tokenizer/                  # Native BPE and subword token counters with LRU cache
│   │   ├── counter.go
│   │   └── cache.go
│   ├── pruner/                     # Multi-tier pruning pipeline
│   │   ├── pipeline.go
│   │   ├── tier1_syntactic.go      # JSON minification, whitespace and markup cleanup
│   │   ├── tier2_eviction.go       # Tool call lifecycle and semantic receipt emission
│   │   ├── tier3_state_delta.go    # Conversation consolidation and state tracking
│   │   └── budget.go               # Adaptive budget regulator
│   ├── store/                      # SQLite persistence (pure Go, zero CGO)
│   │   ├── db.go
│   │   └── telemetry.go            # Request records, metrics, and SSE broadcast hub
│   └── api/                        # Telemetry and management REST handlers
│       ├── routes.go
│       └── handler.go
├── web/                            # React 19 Telemetry Studio
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.tsx
│       ├── components/
│       │   ├── SavingsGauge.tsx    # Radial fuel gauge showing token savings percentage
│       │   ├── TurnWaterfall.tsx   # Per-turn context growth versus pruned curve
│       │   ├── DiffViewer.tsx      # Side-by-side raw versus pruned prompt diff
│       │   ├── LiveFeed.tsx        # Real-time event log
│       │   └── Sandbox.tsx         # Interactive live compression tester
│       └── pages/
│           └── Dashboard.tsx
├── examples/
│   ├── sample_bloated_prompt.json
│   └── tokenprune.yaml
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── README.md
```

---

## 5. Implementation Roadmap

### Phase 1: Go Module Setup and Configuration
- Initialize Go module with modern, minimal dependencies.
- Implement `internal/config` supporting YAML parsing, default fallbacks, and environment expansion.

### Phase 2: Token Counting Engine
- Implement high-speed local token counter in `internal/tokenizer` supporting OpenAI and Anthropic formats.
- Integrate thread-safe FNV-1a hash caching to ensure sub-millisecond repeated counting.

### Phase 3: Three-Tier Pruning Pipeline
- Tier 1: Implement syntactic minification for JSON arguments, whitespace collapsing, and HTML comment stripping without altering code blocks.
- Tier 2: Implement tool call tracking. Replace superseded successful read outputs with semantic receipts while preserving error outputs and message IDs.
- Tier 3: Implement state delta consolidation for turns older than $N$ steps, anchoring system instructions and active context.
- Budget Controller: Build dynamic pruning regulator to satisfy token ceilings.

### Phase 4: Persistence and Telemetry
- Configure SQLite storage using `modernc.org/sqlite` for pure Go, zero-CGO operation across platforms.
- Implement metrics aggregation (cumulative tokens, USD savings, latency reductions).
- Build in-memory SSE event bus for real-time dashboard updates.

### Phase 5: Streaming Reverse Proxy
- Implement reverse proxy supporting OpenAI chat completions and Anthropic messages endpoints.
- Intercept incoming request payloads, prune context, update headers, and forward to upstream.
- Stream SSE response chunks directly to clients while recording TTFT and completion metrics.

### Phase 6: Telemetry REST API and CLI Suite
- Create REST endpoints for metrics, session inspection, and live sandbox testing.
- Implement CLI subcommands: `serve`, `bench`, `prune`, `report`, and `version`.

### Phase 7: Embedded React 19 Telemetry Studio
- Build frontend with React 19, Tailwind CSS, and Lucide icons.
- Implement fuel gauge, waterfall chart, side-by-side diff viewer, live session feed, and interactive sandbox.
- Embed production frontend assets into the Go binary via `//go:embed`.

### Phase 8: Verification, Test Coverage, and Documentation
- Write comprehensive unit tests for all tiers, proxy routing, token counting, and configuration.
- Write benchmark tests validating memory usage (< 20MB) and processing latency (< 10ms).
- Author humanized README, architecture docs, and working example files.
- Commit and publish to GitHub with proper releases and tags.
