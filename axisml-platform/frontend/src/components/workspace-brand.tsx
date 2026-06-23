import type { SVGProps } from "react";

// Tool brand marks (official colours), mirroring the prototype's `svg.brand`
// symbols. Shared by the workspace list cards, the create-drawer image picker,
// and the detail header's launch buttons so the Jupyter / VS Code / PyTorch
// glyphs stay identical everywhere.

type MarkProps = SVGProps<SVGSVGElement>;

export function JupyterMark({ className, ...props }: MarkProps) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false" {...props}>
      <path
        fill="#F37726"
        d="M7.157 22.201A1.784 1.784 0 0 1 5.374 24a1.784 1.784 0 0 1-1.784-1.799 1.784 1.784 0 0 1 1.784-1.799 1.784 1.784 0 0 1 1.783 1.799zM20.582 1.427a1.415 1.415 0 0 1-1.415 1.428 1.415 1.415 0 0 1-1.416-1.428A1.415 1.415 0 0 1 19.167 0a1.415 1.415 0 0 1 1.415 1.427zM4.992 3.336A1.781 1.781 0 0 1 3.21 5.135 1.781 1.781 0 0 1 1.427 3.336 1.781 1.781 0 0 1 3.21 1.537a1.781 1.781 0 0 1 1.782 1.799zM12 18.694c-3.945 0-7.394-1.417-9.191-3.506a9.799 9.799 0 0 0 18.382 0c-1.797 2.089-5.246 3.506-9.191 3.506zM12 5.306c3.945 0 7.394 1.417 9.191 3.506a9.799 9.799 0 0 0-18.382 0C4.606 6.723 8.055 5.306 12 5.306z"
      />
    </svg>
  );
}

export function VscodeMark({ className, ...props }: MarkProps) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false" {...props}>
      <path
        fill="#007ACC"
        d="M23.15 2.587L18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z"
      />
    </svg>
  );
}

export function PytorchMark({ className, ...props }: MarkProps) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false" {...props}>
      <path
        fill="#EE4C2C"
        d="M12.005 0L4.952 7.053a9.865 9.865 0 0 0 0 13.945 9.866 9.866 0 0 0 13.946 0c3.515-3.515 3.515-9.21 0-12.724l-1.508 1.508c2.682 2.682 2.682 7.026 0 9.708a6.865 6.865 0 0 1-9.71 0 6.865 6.865 0 0 1 0-9.708l4.317-4.34.008.008V0zm3.291 4.388a1.184 1.184 0 1 0 0 2.368 1.184 1.184 0 0 0 0-2.368z"
      />
    </svg>
  );
}

export type WsBrand = "jupyter" | "vscode" | "pytorch";

export function wsBrand(image: string): WsBrand {
  const s = (image || "").toLowerCase();
  if (s.includes("vscode") || s.includes("code-server")) return "vscode";
  if (s.includes("pytorch") || s.includes("torch")) return "pytorch";
  return "jupyter";
}

export function WorkspaceMark({ brand, ...props }: { brand: WsBrand } & MarkProps) {
  const Mark = brand === "vscode" ? VscodeMark : brand === "pytorch" ? PytorchMark : JupyterMark;
  return <Mark {...props} />;
}
