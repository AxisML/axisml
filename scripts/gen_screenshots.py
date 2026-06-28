#!/usr/bin/env python3
"""Generate localized UI screenshots of the Platform frontend.

Drives the Vite + React SPA in its in-browser mock mode (VITE_USE_MOCK_API), so it
needs neither the backend nor a cluster. For each shipped language it walks every
route and writes a PNG to docs/screenshots/<locale>/:

    en    -> docs/screenshots/en
    zh-CN -> docs/screenshots/zh-CN

Usage (preferred — the wrapper installs deps for you):
    scripts/gen-screenshots.sh

Or directly, once Playwright (+ chromium) is available in the environment:
    python3 scripts/gen_screenshots.py

Environment knobs:
    SCREENSHOT_BASE_URL    reuse an already-running dev server instead of spawning one
    SCREENSHOT_FULL_PAGE   "false" to capture just the viewport (default: full page)
    SCREENSHOT_SCALE       device scale factor for the captures (default: 2)
"""

from __future__ import annotations

import os
import shutil
import signal
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
FRONTEND_DIR = REPO_ROOT / "axisml-platform" / "frontend"
OUT_ROOT = REPO_ROOT / "docs" / "screenshots"

PORT = 5173
FULL_PAGE = os.environ.get("SCREENSHOT_FULL_PAGE") != "false"
SCALE = float(os.environ.get("SCREENSHOT_SCALE", "2"))
VIEWPORT = {"width": 1440, "height": 900}

# One demo tenant always exists in mock mode (the first /tenants fixture). Pinning
# it keeps every capture deterministic regardless of the auto-select default.
TENANT = "team-vision"

# The two catalogs the frontend ships (src/i18n/index.ts). `lang` is the value the
# app persists under localStorage "axisml.lang"; `out` is the output folder.
LANGUAGES = [
    {"out": "en", "lang": "en"},
    {"out": "zh-CN", "lang": "zh"},
]

# Every route from src/app/router.tsx. Detail paths use the mock fixtures' names
# (src/api/mock/data.ts); the mock router falls back to the first record for any
# unknown name, so these stay valid even if the fixtures are renamed.
ROUTES = [
    {"name": "login", "path": "/login", "auth": False},
    {"name": "dashboard", "path": "/"},
    {"name": "workspaces", "path": "/workspaces"},
    {"name": "workspace-detail", "path": "/workspaces/data-prep"},
    {"name": "experiments", "path": "/experiments"},
    {"name": "experiment-detail", "path": "/experiments/resnet-aug-search"},
    {"name": "experiment-run", "path": "/experiments/resnet-aug-search/runs/resnet-aug-search-2"},
    {"name": "jobs", "path": "/jobs"},
    {"name": "job-detail", "path": "/jobs/eval-recall"},
    {"name": "job-run", "path": "/jobs/eval-recall/runs/eval-recall-2"},
    {"name": "services", "path": "/services"},
    {"name": "service-detail", "path": "/services/bge-embed-svc"},
    {"name": "traffic", "path": "/traffic"},
    {"name": "traffic-detail", "path": "/traffic/bge-embed-weighted"},
    {"name": "models", "path": "/models"},
    {"name": "images", "path": "/images"},
    {"name": "tenants", "path": "/tenants"},
    {"name": "resource-pools", "path": "/resource-pools"},
    {"name": "data-volumes", "path": "/data-volumes"},
]


def log(msg: str) -> None:
    print(f"[screenshots] {msg}", flush=True)


def load_playwright():
    try:
        from playwright.sync_api import sync_playwright  # noqa: PLC0415

        return sync_playwright
    except ImportError as exc:
        raise SystemExit(
            "Playwright is not installed. Run scripts/gen-screenshots.sh (it installs it for you),\n"
            "  or: pip install playwright && python3 -m playwright install chromium"
        ) from exc


def wait_for_server(base_url: str, timeout: float = 90.0) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:  # urlopen handles name resolution (IPv4/IPv6) like the browser does.
            with urllib.request.urlopen(base_url, timeout=2) as resp:
                if resp.status == 200:
                    return
        except Exception:  # noqa: BLE001 — keep polling until the SPA answers
            pass
        time.sleep(0.5)
    raise RuntimeError(f"dev server at {base_url} did not come up within {timeout}s")


def start_dev_server() -> subprocess.Popen:
    """Spawn the Vite dev server in mock mode, in its own process group."""
    log("starting Vite dev server (VITE_USE_MOCK_API=true)…")
    env = {**os.environ, "VITE_USE_MOCK_API": "true"}
    log_path = Path(os.environ.get("TMPDIR", "/tmp")) / "axisml-screenshots-vite.log"
    log_file = open(log_path, "w")  # noqa: SIM115 — kept open for the server's lifetime
    proc = subprocess.Popen(
        ["pnpm", "dev", "--host", "127.0.0.1", "--port", str(PORT), "--strictPort"],
        cwd=str(FRONTEND_DIR),
        env=env,
        stdout=log_file,
        stderr=subprocess.STDOUT,
        start_new_session=True,  # own process group so we can kill the whole tree
    )
    proc._dev_log_path = log_path  # type: ignore[attr-defined]  # surfaced on failure
    return proc


def stop_dev_server(proc: subprocess.Popen | None) -> None:
    if proc is None:
        return
    try:
        os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
    except ProcessLookupError:
        pass


def capture_language(browser, language: dict, base_url: str) -> None:
    out_dir = OUT_ROOT / language["out"]
    if out_dir.exists():
        shutil.rmtree(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    log(f'capturing {len(ROUTES)} routes for "{language["out"]}"…')

    for route in ROUTES:
        authed = route.get("auth", True)
        context = browser.new_context(viewport=VIEWPORT, device_scale_factor=SCALE)
        # Seed localStorage before any app code runs: language, active tenant, and —
        # for protected routes — a mock JWT so the auth gate admits us straight in.
        token = "mock.jwt.token" if authed else ""
        context.add_init_script(
            f"""
            localStorage.setItem("axisml.lang", "{language['lang']}");
            localStorage.setItem("axisml.tenant", "{TENANT}");
            {f'localStorage.setItem("axisml.token", "{token}");' if token else ''}
            """
        )
        page = context.new_page()
        try:
            page.goto(f"{base_url}{route['path']}", wait_until="networkidle", timeout=30_000)
            page.wait_for_timeout(900)  # let charts / status dots settle
            out_file = out_dir / f"{route['name']}.png"
            page.screenshot(path=str(out_file), full_page=FULL_PAGE)
            log(f"  ✓ {language['out']}/{route['name']}.png")
        except Exception as exc:  # noqa: BLE001 — one bad route shouldn't abort the rest
            print(f"[screenshots]  ✗ {language['out']}/{route['name']}: {exc}", flush=True)
        finally:
            context.close()


def main() -> None:
    sync_playwright = load_playwright()

    external_url = os.environ.get("SCREENSHOT_BASE_URL")
    base_url = external_url or f"http://127.0.0.1:{PORT}"

    with sync_playwright() as pw:
        browser = pw.chromium.launch()
        server = None
        try:
            if external_url:
                log(f"reusing dev server at {base_url}")
            else:
                server = start_dev_server()
            try:
                wait_for_server(base_url)
            except RuntimeError:
                if server is not None and server.poll() is not None:
                    log(f"dev server exited early (code {server.returncode})")
                log_path = getattr(server, "_dev_log_path", None)
                if log_path and Path(log_path).exists():
                    log(f"--- dev server log ({log_path}) ---")
                    print(Path(log_path).read_text(), file=sys.stderr)
                raise
            log(f"dev server ready at {base_url}")
            for language in LANGUAGES:
                capture_language(browser, language, base_url)
            rel = OUT_ROOT.relative_to(REPO_ROOT)
            log(f"done — screenshots written under {rel}/{{en,zh-CN}}")
        finally:
            browser.close()
            stop_dev_server(server)


if __name__ == "__main__":
    main()
