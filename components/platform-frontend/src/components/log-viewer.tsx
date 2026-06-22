// Dark terminal log box — the prototype's `.logbox`. Renders plain log text with
// per-line severity tinting (timestamp prefix dimmed; INFO/WARN/ERROR coloured).
// Shared by RunDetail / WorkspaceDetail / ServiceDetail so every log surface looks
// identical. Pass the raw log string (e.g. from a pod-logs endpoint).
function LogLine({ line }: { line: string }) {
  const tsMatch = line.match(/^(\[?\d[\d:.\- TZ]*\]?)\s+(.*)$/);
  const lvl = /\b(ERROR|ERR|FATAL)\b|\[ERR/i.test(line)
    ? "text-red-400"
    : /\b(WARN|WARNING)\b|\[WARN/i.test(line)
      ? "text-amber-300"
      : /\b(INFO)\b|\[INFO/i.test(line)
        ? "text-sky-300"
        : "text-zinc-100";
  if (tsMatch) {
    return (
      <div>
        <span className="mr-2.5 text-zinc-500">{tsMatch[1]}</span>
        <span className={lvl}>{tsMatch[2]}</span>
      </div>
    );
  }
  return <div className={lvl}>{line}</div>;
}

export function LogViewer({ text, empty }: { text?: string | null; empty?: string }) {
  const lines = typeof text === "string" ? text.split("\n").filter((l) => l.length) : [];
  return (
    <div className="max-h-[440px] overflow-auto rounded-md bg-zinc-950 p-4 font-mono text-xs leading-[1.7]">
      {lines.length ? (
        lines.map((l, i) => <LogLine key={i} line={l} />)
      ) : (
        <span className="text-zinc-500">{empty ?? "暂无日志"}</span>
      )}
    </div>
  );
}
