#!/usr/bin/env -S node --experimental-strip-types

import { appendFileSync, existsSync, readFileSync, writeFileSync } from "node:fs";
import { spawn, spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

type Lane = "agent" | "agent-browser";
type Provider = "anthropic" | "openai";

type ToolCall = {
  id: string;
  command: string;
  timeoutSeconds: number;
};

type ToolExecutionResult = {
  id: string;
  isError: boolean;
  content: string;
};

type Args = {
  lane: Lane;
  provider?: Provider;
  model?: string;
  groups?: number[];
  profile?: string;
  maxTokens: number;
  temperature: number;
  maxTurns: number;
  maxIdleTurns: number;
  timeoutSeconds: number;
  turnDelayMs: number;
  reportFile?: string;
  skipInit: boolean;
  noPromptCaching: boolean;
  finalize: boolean;
};

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const SCRIPT_DIR = dirname(SCRIPT_PATH);
const BENCH_DIR = resolve(SCRIPT_DIR, "..");
const REPO_ROOT = resolve(BENCH_DIR, "..", "..");
const RESULTS_DIR = resolve(BENCH_DIR, "results");
const CURRENT_AGENT_PTR = resolve(RESULTS_DIR, "current_agent_report.txt");
const CURRENT_AGENT_BROWSER_PTR = resolve(RESULTS_DIR, "current_agent_browser_report.txt");
const AGENT_COMMANDS_LOG = resolve(RESULTS_DIR, "agent_commands.ndjson");
const MAX_TOOL_OUTPUT_CHARS = 2400;
const COMPACT_AFTER_ANTHROPIC_MESSAGES = 10;
const KEEP_RECENT_ANTHROPIC_MESSAGES = 6;
const COMPACT_AFTER_OPENAI_ITEMS = 16;
const KEEP_RECENT_OPENAI_ITEMS = 10;
const MAX_SUMMARY_STEPS = 12;
const MAX_SUMMARY_TEXT_CHARS = 220;
const AGENT_TASKS_FILE = resolve(BENCH_DIR, "AGENT_TASKS.md");
const AGENT_BROWSER_TASKS_FILE = resolve(BENCH_DIR, "AGENT_BROWSER_TASKS.md");

function usage(exitCode = 1): never {
  const stream = exitCode === 0 ? console.log : console.error;
  stream(`Usage:
  node --experimental-strip-types ./tests/benchmark/scripts/run-api-benchmark.ts --lane agent [options]
  node --experimental-strip-types ./tests/benchmark/scripts/run-api-benchmark.ts --lane agent-browser [options]

Options:
  --provider anthropic|openai
  --model MODEL
  --groups 0,1,2,3
  --profile common10
  --max-tokens N
  --temperature N
  --max-turns N
  --max-idle-turns N
  --timeout-seconds N
  --turn-delay-ms N
  --report-file PATH
  --skip-init
  --no-prompt-caching
  --finalize
`);
  process.exit(exitCode);
}

function parseArgs(argv: string[]): Args {
  let lane: Lane | undefined;
  let provider: Provider | undefined;
  let model: string | undefined;
  let groups: number[] | undefined;
  let profile: string | undefined;
  let maxTokens = 4096;
  let temperature = 0;
  let maxTurns = 300;
  let maxIdleTurns = 25;
  let timeoutSeconds = 120;
  let turnDelayMs = 1500;
  let reportFile: string | undefined;
  let skipInit = false;
  let noPromptCaching = false;
  let finalize = false;

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case "--lane":
        lane = argv[++i] as Lane;
        break;
      case "--provider":
        provider = argv[++i] as Provider;
        break;
      case "--model":
        model = argv[++i];
        break;
      case "--groups":
        groups = argv[++i]
          .split(",")
          .map((value) => Number(value.trim()))
          .filter((value) => Number.isInteger(value));
        break;
      case "--profile":
        profile = argv[++i];
        break;
      case "--max-tokens":
        maxTokens = Number(argv[++i]);
        break;
      case "--temperature":
        temperature = Number(argv[++i]);
        break;
      case "--max-turns":
        maxTurns = Number(argv[++i]);
        break;
      case "--max-idle-turns":
        maxIdleTurns = Number(argv[++i]);
        break;
      case "--timeout-seconds":
        timeoutSeconds = Number(argv[++i]);
        break;
      case "--turn-delay-ms":
        turnDelayMs = Number(argv[++i]);
        break;
      case "--report-file":
        reportFile = argv[++i];
        break;
      case "--skip-init":
        skipInit = true;
        break;
      case "--no-prompt-caching":
        noPromptCaching = true;
        break;
      case "--finalize":
        finalize = true;
        break;
      case "-h":
      case "--help":
      case "help":
        usage(0);
        break;
      default:
        throw new Error(`Unknown argument: ${arg}`);
    }
  }

  if (lane !== "agent" && lane !== "agent-browser") {
    throw new Error("--lane must be 'agent' or 'agent-browser'");
  }

  return {
    lane,
    provider,
    model,
    groups,
    profile,
    maxTokens,
    temperature,
    maxTurns,
    maxIdleTurns,
    timeoutSeconds,
    turnDelayMs,
    reportFile,
    skipInit,
    noPromptCaching,
    finalize,
  };
}

function resolveGroups(args: Args): number[] | undefined {
  if (args.groups && args.groups.length > 0) {
    return [...new Set(args.groups)].sort((a, b) => a - b);
  }
  if (!args.profile) {
    return undefined;
  }
  switch (args.profile) {
    case "common10":
      return [0, 1, 2, 3];
    default:
      throw new Error(`Unknown profile: ${args.profile}`);
  }
}

function formatGroups(groups?: number[]): string {
  if (!groups || groups.length === 0) {
    return "all";
  }
  return groups.join(",");
}

function extractGroupSections(markdownPath: string, groups: number[]): string {
  const lines = readText(markdownPath).split("\n");
  const wanted = new Set(groups);
  const collected: string[] = [];
  let capture = false;

  for (const line of lines) {
    const match = line.match(/^## Group (\d+):/);
    if (match) {
      const group = Number(match[1]);
      capture = wanted.has(group);
    }
    if (capture) {
      collected.push(line);
    }
  }

  return collected.join("\n").trim();
}

function extractSingleGroupSection(markdownPath: string, group: number): string {
  const lines = readText(markdownPath).split("\n");
  const collected: string[] = [];
  let capture = false;

  for (const line of lines) {
    const match = line.match(/^## Group (\d+):/);
    if (match) {
      const currentGroup = Number(match[1]);
      if (capture && currentGroup !== group) {
        break;
      }
      capture = currentGroup === group;
    }
    if (capture) {
      collected.push(line);
    }
  }

  return collected.join("\n").trim();
}

function readText(path: string): string {
  return readFileSync(path, "utf8").trim();
}

function readJsonFile(path: string): any {
  return JSON.parse(readFileSync(path, "utf8"));
}

function resolveReportPath(lane: Lane): string {
  const ptr = lane === "agent" ? CURRENT_AGENT_PTR : CURRENT_AGENT_BROWSER_PTR;
  return readText(ptr);
}

function runChecked(cmd: string[], env?: NodeJS.ProcessEnv): void {
  const result = spawnSync(cmd[0], cmd.slice(1), {
    cwd: REPO_ROOT,
    env: env ?? process.env,
    stdio: "inherit",
  });
  if (result.status !== 0) {
    throw new Error(`command failed: ${cmd.join(" ")}`);
  }
}

function resolveProvider(args: Args): Provider {
  if (args.provider) {
    return args.provider;
  }
  const hasOpenAI = (process.env.OPENAI_API_KEY ?? "").trim().length > 0;
  const hasAnthropic = (process.env.ANTHROPIC_API_KEY ?? "").trim().length > 0;
  const configured = [hasOpenAI, hasAnthropic].filter(Boolean).length;
  if (configured > 1) {
    throw new Error("Multiple providers appear configured. Pass --provider openai|anthropic explicitly.");
  }
  if (hasOpenAI) {
    return "openai";
  }
  if (hasAnthropic) {
    return "anthropic";
  }
  throw new Error("Set OPENAI_API_KEY or ANTHROPIC_API_KEY, or pass --provider");
}

function resolveModel(provider: Provider, explicit?: string): string {
  if (explicit) {
    return explicit;
  }
  if (provider === "openai") {
    return process.env.OPENAI_MODEL?.trim() || "gpt-5";
  }
  return process.env.ANTHROPIC_MODEL?.trim() || "claude-haiku-4-5-20251001";
}

async function listAnthropicModels(apiKey: string): Promise<string[]> {
  const response = await fetch("https://api.anthropic.com/v1/models", {
    method: "GET",
    headers: {
      "x-api-key": apiKey,
      "anthropic-version": "2023-06-01",
    },
  });

  const payload = await response.json();
  if (!response.ok) {
    throw new Error(`Anthropic models API error ${response.status}: ${JSON.stringify(payload)}`);
  }

  return Array.isArray(payload.data)
    ? payload.data
        .map((item: any) => String(item.id ?? "").trim())
        .filter((id: string) => id.length > 0)
    : [];
}

async function resolveAnthropicModel(explicit?: string): Promise<string> {
  if (explicit) {
    return explicit;
  }

  const envModel = process.env.ANTHROPIC_MODEL?.trim();
  if (envModel) {
    return envModel;
  }

  const apiKey = (process.env.ANTHROPIC_API_KEY ?? "").trim();
  if (!apiKey) {
    return "claude-haiku-4-5-20251001";
  }

  try {
    const available = await listAnthropicModels(apiKey);
    const preferred = [
      "claude-haiku-4-5-20251001",
      "claude-3-5-haiku-20241022",
      "claude-3-5-haiku-latest",
      "claude-3-haiku-20240307",
      "claude-sonnet-4-6",
      "claude-sonnet-4-20250514",
      "claude-sonnet-4-5-20250929",
      "claude-3-7-sonnet-20250219",
      "claude-3-7-sonnet-latest",
      "claude-3-5-sonnet-20241022",
      "claude-3-5-sonnet-latest",
      "claude-3-5-sonnet-20240620",
      "claude-opus-4-7",
      "claude-opus-4-6",
      "claude-opus-4-5-20251101",
      "claude-opus-4-1-20250805",
    ];

    for (const candidate of preferred) {
      if (available.includes(candidate)) {
        return candidate;
      }
    }

    if (available.length > 0) {
      return available[0];
    }
  } catch {
    // Fall back to a documented default if model discovery fails.
  }

  return "claude-haiku-4-5-20251001";
}

function runnerSource(provider: Provider): string {
  if (provider === "openai") {
    return "openai-responses";
  }
  return "anthropic-messages";
}

function initializeLane(args: Args, provider: Provider, model: string): string {
  const env = { ...process.env, BENCHMARK_MODEL: model, BENCHMARK_RUNNER: runnerSource(provider) };
  if (args.lane === "agent") {
    runChecked([resolve(BENCH_DIR, "scripts", "run-optimization.sh")], env);
  } else {
    runChecked([resolve(BENCH_DIR, "scripts", "run-agent-browser-benchmark.sh")], env);
  }
  return resolveReportPath(args.lane);
}

class PersistentShell {
  private proc = spawn("/bin/bash", [], {
    cwd: BENCH_DIR,
    stdio: ["pipe", "pipe", "pipe"],
  });

  constructor() {
    this.proc.stderr?.pipe(process.stderr);
  }

  async run(command: string, timeoutSeconds: number): Promise<{ output: string; exitCode: number }> {
    const marker = `__BENCH_DONE__${Math.random().toString(16).slice(2)}`;
    const wrapped = `${command}\nprintf '\\n${marker}:%s\\n' "$?"\n`;
    const stdout = this.proc.stdout;
    const stdin = this.proc.stdin;
    if (!stdout || !stdin) {
      throw new Error("persistent shell unavailable");
    }

    stdin.write(wrapped);
    const deadline = Date.now() + timeoutSeconds * 1000;
    let buffer = "";

    while (Date.now() < deadline) {
      const chunk = await new Promise<string>((resolveChunk, rejectChunk) => {
        const onData = (data: Buffer) => {
          cleanup();
          resolveChunk(data.toString("utf8"));
        };
        const onExit = () => {
          cleanup();
          rejectChunk(new Error("persistent shell exited unexpectedly"));
        };
        const cleanup = () => {
          clearTimeout(timer);
          stdout.off("data", onData);
          this.proc.off("exit", onExit);
        };
        const timer = setTimeout(() => {
          cleanup();
          resolveChunk("");
        }, 100);
        stdout.once("data", onData);
        this.proc.once("exit", onExit);
      });

      if (chunk) {
        buffer += chunk;
        const match = buffer.match(new RegExp(`\\n${marker}:(\\d+)\\r?\\n`));
        if (match) {
          const output = buffer.slice(0, match.index).replace(/^\n/, "");
          return { output, exitCode: Number(match[1]) };
        }
      }
    }

    this.close(true);
    throw new Error(`shell command timed out after ${timeoutSeconds}s: ${command}`);
  }

  close(force = false): void {
    if (this.proc.exitCode !== null) {
      return;
    }
    if (force) {
      this.proc.kill("SIGKILL");
      return;
    }
    this.proc.stdin?.end("exit\n");
  }
}

abstract class ApiRunner {
  requestCount = 0;
  inputTokens = 0;
  outputTokens = 0;
  cacheCreationInputTokens = 0;
  cacheReadInputTokens = 0;
  provider: Provider;
  source: string;
  apiKey: string;
  model: string;
  maxTokens: number;
  temperature: number;
  promptCaching: boolean;

  constructor(provider: Provider, source: string, apiKey: string, model: string, maxTokens: number, temperature: number, promptCaching: boolean) {
    this.provider = provider;
    this.source = source;
    this.apiKey = apiKey;
    this.model = model;
    this.maxTokens = maxTokens;
    this.temperature = temperature;
    this.promptCaching = promptCaching;
  }

  abstract toolDefinitions(): unknown[];
  abstract initialConversation(userPrompt: string): unknown[];
  abstract send(systemPrompt: string, conversation: unknown[]): Promise<any>;
  abstract extractToolCalls(response: any, defaultTimeout: number): ToolCall[];
  abstract appendToolResults(conversation: unknown[], response: any, results: ToolExecutionResult[]): void;
  abstract extractFinalText(response: any): string;
}

class AnthropicRunner extends ApiRunner {
  constructor(apiKey: string, model: string, maxTokens: number, temperature: number, promptCaching: boolean) {
    super("anthropic", "anthropic-messages", apiKey, model, maxTokens, temperature, promptCaching);
  }

  toolDefinitions(): unknown[] {
    return [
      {
        name: "run_command",
        description:
          "Run a shell command in a persistent bash session rooted at tests/benchmark. Environment changes such as cd/export persist across calls.",
        input_schema: {
          type: "object",
          properties: {
            command: { type: "string" },
            timeout_seconds: { type: "integer", minimum: 1, maximum: 600 },
          },
          required: ["command"],
        },
      },
    ];
  }

  initialConversation(userPrompt: string): unknown[] {
    return [{ role: "user", content: userPrompt }];
  }

  async send(systemPrompt: string, conversation: unknown[]): Promise<any> {
    const body: Record<string, unknown> = {
      model: this.model,
      max_tokens: this.maxTokens,
      temperature: this.temperature,
      system: systemPrompt,
      tools: this.toolDefinitions(),
      messages: conversation,
    };
    if (this.promptCaching) {
      body.cache_control = { type: "ephemeral" };
    }

    const response = await fetchWithRetry("https://api.anthropic.com/v1/messages", {
      method: "POST",
      headers: {
        "content-type": "application/json",
        "x-api-key": this.apiKey,
        "anthropic-version": "2023-06-01",
      },
      body: JSON.stringify(body),
    });

    const payload = await response.json();
    if (!response.ok) {
      throw new Error(`Anthropic API error ${response.status}: ${JSON.stringify(payload)}`);
    }

    const usage = payload.usage ?? {};
    this.requestCount += 1;
    this.inputTokens += Number(usage.input_tokens ?? 0);
    this.outputTokens += Number(usage.output_tokens ?? 0);
    this.cacheCreationInputTokens += Number(usage.cache_creation_input_tokens ?? 0);
    this.cacheReadInputTokens += Number(usage.cache_read_input_tokens ?? 0);
    return payload;
  }

  extractToolCalls(response: any, defaultTimeout: number): ToolCall[] {
    return (response.content ?? [])
      .filter((item: any) => item.type === "tool_use")
      .map((item: any) => ({
        id: String(item.id),
        command: String(item.input?.command ?? "").trim(),
        timeoutSeconds: Number(item.input?.timeout_seconds ?? defaultTimeout),
      }))
      .filter((call: ToolCall) => call.command.length > 0);
  }

  appendToolResults(conversation: unknown[], response: any, results: ToolExecutionResult[]): void {
    conversation.push({ role: "assistant", content: response.content });
    if (results.length > 0) {
      conversation.push({
        role: "user",
        content: results.map((result) => ({
          type: "tool_result",
          tool_use_id: result.id,
          is_error: result.isError,
          content: result.content,
        })),
      });
    }
  }

  extractFinalText(response: any): string {
    return (response.content ?? [])
      .filter((item: any) => item.type === "text" && item.text)
      .map((item: any) => String(item.text))
      .join("\n")
      .trim();
  }
}

class OpenAIRunner extends ApiRunner {
  private baseUrl: string;

  constructor(provider: "openai", source: string, baseUrl: string, apiKey: string, model: string, maxTokens: number, temperature: number, promptCaching: boolean) {
    super(provider, source, apiKey, model, maxTokens, temperature, promptCaching);
    this.baseUrl = baseUrl;
  }

  toolDefinitions(): unknown[] {
    return [
      {
        type: "function",
        name: "run_command",
        description:
          "Run a shell command in a persistent bash session rooted at tests/benchmark. Environment changes such as cd/export persist across calls.",
        parameters: {
          type: "object",
          properties: {
            command: { type: "string" },
            timeout_seconds: { type: "integer", minimum: 1, maximum: 600 },
          },
          required: ["command"],
        },
      },
    ];
  }

  initialConversation(userPrompt: string): unknown[] {
    return [{ role: "user", content: [{ type: "input_text", text: userPrompt }] }];
  }

  async send(systemPrompt: string, conversation: unknown[]): Promise<any> {
    const body: Record<string, unknown> = {
      model: this.model,
      instructions: systemPrompt,
      tools: this.toolDefinitions(),
      input: conversation,
      max_output_tokens: this.maxTokens,
    };
    if (this.promptCaching) {
      body.prompt_cache_retention = "24h";
    }

    const response = await fetchWithRetry(this.baseUrl, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        authorization: `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify(body),
    });

    const payload = await response.json();
    if (!response.ok) {
      throw new Error(`OpenAI API error ${response.status}: ${JSON.stringify(payload)}`);
    }

    const usage = payload.usage ?? {};
    const totalInput = Number(usage.input_tokens ?? 0);
    const cached = Number(usage.input_tokens_details?.cached_tokens ?? 0);
    this.requestCount += 1;
    this.inputTokens += Math.max(0, totalInput - cached);
    this.outputTokens += Number(usage.output_tokens ?? 0);
    this.cacheReadInputTokens += cached;
    return payload;
  }

  extractToolCalls(response: any, defaultTimeout: number): ToolCall[] {
    return (response.output ?? [])
      .filter((item: any) => item.type === "function_call")
      .map((item: any) => {
        const parsed = item.arguments ? JSON.parse(item.arguments) : {};
        return {
          id: String(item.call_id ?? item.id),
          command: String(parsed.command ?? "").trim(),
          timeoutSeconds: Number(parsed.timeout_seconds ?? defaultTimeout),
        };
      })
      .filter((call: ToolCall) => call.command.length > 0);
  }

  appendToolResults(conversation: unknown[], response: any, results: ToolExecutionResult[]): void {
    conversation.push(...(response.output ?? []));
    for (const result of results) {
      conversation.push({
        type: "function_call_output",
        call_id: result.id,
        output: result.content,
      });
    }
  }

  extractFinalText(response: any): string {
    if (typeof response.output_text === "string" && response.output_text.trim()) {
      return response.output_text.trim();
    }
    return (response.output ?? [])
      .flatMap((item: any) => (item.type === "message" ? item.content ?? [] : []))
      .filter((part: any) => (part.type === "output_text" || part.type === "text") && part.text)
      .map((part: any) => String(part.text))
      .join("\n")
      .trim();
  }
}

function trimToolOutput(text: string): string {
  if (text.length <= MAX_TOOL_OUTPUT_CHARS) {
    return text;
  }
  return `${text.slice(0, MAX_TOOL_OUTPUT_CHARS)}\n\n[output truncated: ${text.length - MAX_TOOL_OUTPUT_CHARS} more chars]`;
}

function parseRetryAfterMs(value: string | null): number {
  if (!value) {
    return 0;
  }
  const seconds = Number(value);
  if (Number.isFinite(seconds) && seconds > 0) {
    return Math.ceil(seconds * 1000);
  }
  const dateMs = Date.parse(value);
  if (!Number.isNaN(dateMs)) {
    return Math.max(0, dateMs - Date.now());
  }
  return 0;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolvePromise) => setTimeout(resolvePromise, ms));
}

async function fetchWithRetry(url: string, init: RequestInit, maxRetries = 3): Promise<Response> {
  let attempt = 0;
  while (true) {
    const response = await fetch(url, init);
    if (response.status !== 429 || attempt >= maxRetries) {
      return response;
    }

    const retryAfterMs = parseRetryAfterMs(response.headers.get("retry-after"));
    const backoffMs = retryAfterMs > 0 ? retryAfterMs : Math.min(15000, 2000 * (attempt + 1));
    await sleep(backoffMs);
    attempt += 1;
  }
}

async function executeToolCalls(shell: PersistentShell, calls: ToolCall[]): Promise<ToolExecutionResult[]> {
  const results: ToolExecutionResult[] = [];
  for (const call of calls) {
    try {
      const { output, exitCode } = await shell.run(call.command, call.timeoutSeconds);
      const result = {
        id: call.id,
        isError: exitCode !== 0,
        content: `$ ${call.command}\n[exit_code=${exitCode}]\n${trimToolOutput(output)}`,
      };
      results.push(result);
      appendCommandLog(call.command, exitCode, trimToolOutput(output));
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      const result = {
        id: call.id,
        isError: true,
        content: `$ ${call.command}\n[runner_error]\n${message}`,
      };
      results.push(result);
      appendCommandLog(call.command, -1, message);
    }
  }
  return results;
}

function appendCommandLog(command: string, exitCode: number, output: string): void {
  const entry = {
    timestamp: new Date().toISOString(),
    command,
    exit_code: exitCode,
    output,
  };
  appendFileSync(AGENT_COMMANDS_LOG, `${JSON.stringify(entry)}\n`, "utf8");
}

function recordUsage(reportFile: string, runner: ApiRunner): void {
  runChecked([
    resolve(BENCH_DIR, "scripts", "record-run-usage.sh"),
    "--report-file",
    reportFile,
    "--provider",
    runner.provider,
    "--source",
    runner.source,
    "--request-count",
    String(runner.requestCount),
    "--input-tokens",
    String(runner.inputTokens),
    "--output-tokens",
    String(runner.outputTokens),
    "--cache-creation-input-tokens",
    String(runner.cacheCreationInputTokens),
    "--cache-read-input-tokens",
    String(runner.cacheReadInputTokens),
  ]);
}

function shorten(text: string, maxChars = MAX_SUMMARY_TEXT_CHARS): string {
  const normalized = text.replace(/\s+/g, " ").trim();
  if (normalized.length <= maxChars) {
    return normalized;
  }
  return `${normalized.slice(0, maxChars - 3)}...`;
}

function buildProgressSummary(reportFile: string): string {
  if (!existsSync(reportFile)) {
    return "No benchmark report file exists yet.";
  }

  const report = JSON.parse(readText(reportFile));
  const totals = report.totals ?? {};
  const steps = Array.isArray(report.steps) ? report.steps : [];
  const recentSteps = steps.slice(-MAX_SUMMARY_STEPS).map((step: any) => {
    const verification = step.verification?.status ? ` / verify=${step.verification.status}` : "";
    const answer = step.answer || step.notes || "";
    return `- ${step.id}: status=${step.status}${verification}; ${shorten(String(answer))}`;
  });

  return [
    "Benchmark progress summary from the external report.",
    `- answered=${totals.steps_answered ?? 0}`,
    `- execution_failed=${totals.steps_failed ?? 0}`,
    `- execution_skipped=${totals.steps_skipped ?? 0}`,
    `- verified_passed=${totals.steps_verified_passed ?? 0}`,
    `- verified_failed=${totals.steps_verified_failed ?? 0}`,
    `- verified_skipped=${totals.steps_verified_skipped ?? 0}`,
    `- pending_verification=${totals.steps_pending_verification ?? 0}`,
    recentSteps.length > 0 ? "Recent recorded steps:" : "No steps recorded yet.",
    ...recentSteps,
    "Use the report file and benchmark wrappers as the source of truth.",
    "Do not re-read large documentation files unless you are blocked on a specific syntax detail.",
  ].join("\n");
}

function readProgress(reportFile: string): { answered: number; failed: number; skipped: number; verifiedPassed: number } {
  if (!existsSync(reportFile)) {
    return { answered: 0, failed: 0, skipped: 0, verifiedPassed: 0 };
  }
  const report = readJsonFile(reportFile);
  const totals = report.totals ?? {};
  return {
    answered: Number(totals.steps_answered ?? 0),
    failed: Number(totals.steps_failed ?? 0),
    skipped: Number(totals.steps_skipped ?? 0),
    verifiedPassed: Number(totals.steps_verified_passed ?? 0),
  };
}

function compactAnthropicConversation(conversation: unknown[], reportFile: string): void {
  if (conversation.length <= COMPACT_AFTER_ANTHROPIC_MESSAGES) {
    return;
  }
  const head = conversation[0];
  const recent = conversation.slice(-KEEP_RECENT_ANTHROPIC_MESSAGES);
  conversation.splice(
    0,
    conversation.length,
    head,
    {
      role: "user",
      content: buildProgressSummary(reportFile),
    },
    ...recent,
  );
}

function compactOpenAIConversation(conversation: unknown[], reportFile: string): void {
  if (conversation.length <= COMPACT_AFTER_OPENAI_ITEMS) {
    return;
  }
  const head = conversation[0];
  const recent = conversation.slice(-KEEP_RECENT_OPENAI_ITEMS);
  conversation.splice(
    0,
    conversation.length,
    head,
    {
      role: "user",
      content: [{ type: "input_text", text: buildProgressSummary(reportFile) }],
    },
    ...recent,
  );
}

function compactConversation(provider: Provider, conversation: unknown[], reportFile: string): void {
  if (provider === "anthropic") {
    compactAnthropicConversation(conversation, reportFile);
    return;
  }
  compactOpenAIConversation(conversation, reportFile);
}

function laneSubsetInstructions(groups?: number[]): string {
  if (!groups || groups.length === 0) {
    return "Execute the full benchmark task set.";
  }

  return [
    `Execute only these benchmark groups: ${groups.join(", ")}.`,
    "Do not attempt groups outside this subset.",
    "Treat all other groups as out of scope for this run rather than as failures.",
    "For the selected groups, execute every step in the group unless blocked.",
  ].join("\n");
}

function laneTaskSourceInstructions(lane: Lane, groups?: number[]): string {
  const fullTaskFile = lane === "agent" ? AGENT_TASKS_FILE : AGENT_BROWSER_TASKS_FILE;
  if (!groups || groups.length === 0) {
    return [
      `Read the full task file at ${fullTaskFile}.`,
      "Execute the full benchmark task set.",
    ].join("\n");
  }

  if (lane === "agent") {
    const subset = extractGroupSections(AGENT_TASKS_FILE, groups);
    return [
      `Use only this selected task subset from ${AGENT_TASKS_FILE}:`,
      "",
      subset,
    ].join("\n");
  }

  const sections: string[] = [];
  for (const group of groups) {
    if (group === 0) {
      sections.push([
        "## Group 0: Setup & Diagnosis (agent-browser lane)",
        "",
        "### 0.1 Open fixtures home",
        "Run `./scripts/ab open http://fixtures/`.",
        "",
        "**Verify**: Open succeeds and the home page loads.",
        "",
        "### 0.2 Snapshot interactive refs",
        "Run `./scripts/ab snapshot -i -c` on the home page.",
        "",
        "**Verify**: Interactive refs are returned without error.",
        "",
        "### 0.3 Session persists across commands",
        "Use the same wrapper session across multiple commands and confirm browser state persists.",
        "",
        "**Verify**: A follow-up action can use the existing page/session state.",
      ].join("\n"));
      continue;
    }

    const section = extractSingleGroupSection(AGENT_TASKS_FILE, group);
    if (section) {
      sections.push(section);
    }
  }

  return [
    `Use only this selected task subset for agent-browser: Group 0 from ${AGENT_BROWSER_TASKS_FILE}, Groups 1+ from ${AGENT_TASKS_FILE}.`,
    "",
    sections.join("\n\n"),
  ].join("\n");
}

type LanePromptConfig = {
  label: string;
  skillInstruction: string;
  wrapper: string;
  recordType: string;
  workflowSummary: string[];
  adapterNotes: string[];
  bootstrapCommands: string[];
};

function lanePromptConfig(lane: Lane): LanePromptConfig {
  if (lane === "agent") {
    return {
      label: "PinchTab agent",
      skillInstruction: `Read ${resolve(REPO_ROOT, "skills", "pinchtab", "SKILL.md")} exactly once before acting.`,
      wrapper: "./scripts/pt",
      recordType: "agent",
      workflowSummary: [
        "use only ./scripts/pt for browser actions",
        "keep one shared tab via PINCHTAB_TAB",
        "use snap -i -c for actionable refs",
        "use text or text --full for reading content",
        "refresh refs after any navigation or DOM change",
      ],
      adapterNotes: [
        'a navigation-expected click can return "Error 409: unexpected page navigation ..."; treat that as likely success and verify with a fresh snapshot/text read',
        "do not assume every command returns JSON; only parse JSON when the command actually returned JSON",
        'the download endpoint returns JSON with base64 content in "data", not a local file path',
        'this environment uses BSD/macOS userland tools; avoid GNU-only flags such as "head -n -1"',
      ],
      bootstrapCommands: [
        "./scripts/pt health",
        "./scripts/pt tab",
        "export PINCHTAB_TAB=$(./scripts/pt nav http://fixtures/)",
        "./scripts/pt snap -i -c",
      ],
    };
  }

  return {
    label: "agent-browser",
    skillInstruction: "Load the official agent-browser skill exactly once with `./scripts/ab skills get agent-browser --full` before acting.",
    wrapper: "./scripts/ab",
    recordType: "agent-browser",
    workflowSummary: [
      "use only ./scripts/ab for browser actions",
      "keep the shared benchmark session across commands",
      "use snapshot -i -c for actionable refs",
      "use fresh refs after any navigation or DOM change",
      "reuse the current browser state instead of reopening pages unless needed",
    ],
    adapterNotes: [
      "after navigation-triggering clicks, verify the resulting page with a fresh snapshot/text read instead of trusting the click response alone",
      "do not assume every command returns JSON; parse only when the command actually returned JSON",
      "for downloads, inspect returned content instead of assuming a file already exists locally",
      'this environment uses BSD/macOS userland tools; avoid GNU-only flags such as "head -n -1"',
    ],
    bootstrapCommands: [
      "./scripts/ab skills get agent-browser --full",
      "./scripts/ab open http://fixtures/",
      "./scripts/ab snapshot -i -c",
    ],
  };
}

function laneUserPrompt(lane: Lane, reportFile: string, groups?: number[]): string {
  const config = lanePromptConfig(lane);
  const firstGroup = groups && groups.length > 0 ? groups[0] : 0;
  const bootstrap = config.bootstrapCommands.map((command, index) => `${index + 1}. ${command}`).join("\n");

  return `
Work in this repo: ${REPO_ROOT}

Benchmark lane: ${config.label} execution.

Requirements:
- Follow a linear execution flow: skill once, selected groups, execute, record, verify, continue.
- Do not read README, browse directories, or inspect unrelated files unless a command path is missing.
- ${config.skillInstruction}
- Tool wrapper:
  - ${config.wrapper}
- Workflow summary:
  - ${config.workflowSummary.join("\n  - ")}
- Adapter notes:
  - ${config.adapterNotes.join("\n  - ")}
- Task scope:
  ${laneTaskSourceInstructions(lane, groups).replace(/\n/g, "\n  ")}
- For each completed step:
  1. record the observed answer with:
  - ./scripts/record-step.sh --report-file ${reportFile} --type ${config.recordType} <group> <step> answer "<what you saw>" "notes"
  2. immediately verify it with:
  - ./scripts/verify-step.sh --report-file ${reportFile} --type ${config.recordType} <group> <step> <pass|fail|skip> "verification notes"
- If a step cannot be completed, record fail or skip in the same report.
- Do not leave answered steps pending verification.
- Keep commands concise. Prefer rg/sed/cat only when you must inspect a specific file.
- After the skill step, begin actual benchmark execution immediately.
- Start with this bootstrap command sequence before attempting the selected steps:
${bootstrap}
- After the bootstrap, immediately execute Group ${firstGroup} step 1.
- Subset selection:
  - ${laneSubsetInstructions(groups).replace(/\n/g, "\n  - ")}
- Finish when all selected steps are executed or when you are blocked.

Your final answer should briefly summarize completion status and the main blockers, if any.
`.trim();
}

function systemPrompt(): string {
  return `
You are a precise benchmark execution agent. Use tools to inspect the repo and run the benchmark lane exactly as instructed.

Rules:
- Never fabricate command output or task results.
- Use the shell tool for all file reads and command execution.
- Do not use destructive commands such as rm -rf, git reset, or checkout.
- After recording an answer, verify it immediately against the task oracle.
- Prefer factual command output over long reasoning.
`.trim();
}

function createRunner(provider: Provider, model: string, args: Args): ApiRunner {
  if (provider === "openai") {
    const apiKey = (process.env.OPENAI_API_KEY ?? "").trim();
    if (!apiKey) {
      throw new Error("OPENAI_API_KEY is required for provider=openai");
    }
    return new OpenAIRunner(
      "openai",
      "openai-responses",
      "https://api.openai.com/v1/responses",
      apiKey,
      model,
      args.maxTokens,
      args.temperature,
      !args.noPromptCaching,
    );
  }

  const apiKey = (process.env.ANTHROPIC_API_KEY ?? "").trim();
  if (!apiKey) {
    throw new Error("ANTHROPIC_API_KEY is required for provider=anthropic");
  }
  return new AnthropicRunner(apiKey, model, args.maxTokens, args.temperature, !args.noPromptCaching);
}

async function main(): Promise<number> {
  try {
    const args = parseArgs(process.argv.slice(2));
    const provider = resolveProvider(args);
    const model = provider === "anthropic" ? await resolveAnthropicModel(args.model) : resolveModel(provider, args.model);
    const groups = resolveGroups(args);
    const reportFile = args.reportFile || (args.skipInit ? resolveReportPath(args.lane) : initializeLane(args, provider, model));
    writeFileSync(AGENT_COMMANDS_LOG, "", "utf8");

    console.log(`[benchmark-runner] provider=${provider} model=${model} lane=${args.lane} groups=${formatGroups(groups)} report=${reportFile}`);

    const runner = createRunner(provider, model, args);
    const shell = new PersistentShell();
    const conversation = runner.initialConversation(laneUserPrompt(args.lane, reportFile, groups));

    let finalText = "";
    let exitCode = 0;
    let idleTurns = 0;
    let lastAnswered = readProgress(reportFile).answered;

    try {
      for (let turn = 1; turn <= args.maxTurns; turn += 1) {
        if (turn > 1 && args.turnDelayMs > 0) {
          await sleep(args.turnDelayMs);
        }
        const response = await runner.send(systemPrompt(), conversation);
        const toolCalls = runner.extractToolCalls(response, args.timeoutSeconds);
        if (toolCalls.length > 0) {
          const results = await executeToolCalls(shell, toolCalls);
          runner.appendToolResults(conversation, response, results);
          compactConversation(runner.provider, conversation, reportFile);
          const progress = readProgress(reportFile);
          if (progress.answered > lastAnswered) {
            idleTurns = 0;
            lastAnswered = progress.answered;
          } else {
            idleTurns += 1;
          }
          if (idleTurns >= args.maxIdleTurns) {
            finalText = `Stopped after ${idleTurns} consecutive turns without recording a benchmark step. Check ${AGENT_COMMANDS_LOG} for the command trace.`;
            exitCode = 3;
            break;
          }
          continue;
        }

        finalText = runner.extractFinalText(response);
        break;
      }

      if (!finalText) {
        finalText = `Stopped after reaching max turns (${args.maxTurns}).`;
        exitCode = 2;
      }
    } finally {
      shell.close();
      if (existsSync(reportFile)) {
        recordUsage(reportFile, runner);
      }
      if (args.finalize && existsSync(reportFile)) {
        runChecked([resolve(BENCH_DIR, "scripts", "finalize-report.sh"), reportFile]);
      }
    }

    if (finalText) {
      console.log(finalText);
    }
    console.log(
      `\n[run-usage] provider=${runner.provider} requests=${runner.requestCount} input=${runner.inputTokens} cache_create=${runner.cacheCreationInputTokens} cache_read=${runner.cacheReadInputTokens} output=${runner.outputTokens} total=${runner.inputTokens + runner.cacheCreationInputTokens + runner.cacheReadInputTokens + runner.outputTokens}`,
    );
    return exitCode;
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    return 1;
  }
}

process.exitCode = await main();
