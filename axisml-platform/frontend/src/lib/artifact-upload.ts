import { USE_MOCK } from "@/api/mock";
import { errorText } from "@/lib/errors";

// Web-upload handshake shared by the model & image "web upload" tabs: initiate
// the version → transfer the bytes to the returned upload target → finalize with
// the content digest (completeModel / completeImage). Previously the tab only
// initiated (or did nothing), so uploads were never finalized.

// sha256 of a File via Web Crypto (secure context: https / localhost).
async function sha256(file: File): Promise<string> {
  const buf = await file.arrayBuffer();
  const hash = await crypto.subtle.digest("SHA-256", buf);
  const hex = Array.from(new Uint8Array(hash))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  return `sha256:${hex}`;
}

// PUT the file bytes to the upload target returned by initiate (the direct-to-
// storage / presigned-URL path). Credential-based targets would add auth here;
// skipped under VITE_USE_MOCK_API since the mock has no object store.
async function putBytes(uri: string, file: File): Promise<void> {
  const res = await fetch(uri, {
    method: "PUT",
    body: file,
    headers: { "Content-Type": file.type || "application/octet-stream" },
  });
  if (!res.ok) throw new Error(`upload failed (${res.status})`);
}

type InitiateResult = Promise<{ data?: { uri?: string } | undefined; error?: unknown }>;

// Orchestrate initiate → transfer → complete. Callers pass the (kind-specific)
// initiate/complete SDK closures so this stays model/image agnostic.
export async function runWebUpload(
  file: File,
  initiate: () => InitiateResult,
  complete: (digest: string) => Promise<{ error?: unknown }>,
): Promise<void> {
  const init = await initiate();
  if (init.error) throw new Error(errorText(init.error));
  const digest = await sha256(file);
  // Mock has no object store — the SDK-level initiate/complete are what the mock
  // answers; only the real byte transfer is skipped.
  if (!USE_MOCK && init.data?.uri) await putBytes(init.data.uri, file);
  const done = await complete(digest);
  if (done.error) throw new Error(errorText(done.error));
}

// Human-readable file size (for the dropzone selected-file label).
export function fmtFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let n = bytes / 1024;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(1)} ${units[i]}`;
}
