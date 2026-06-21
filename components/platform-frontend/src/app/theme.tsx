import { useEffect, useState, type ReactNode } from "react";
import { App as AntApp, ConfigProvider, theme as antdTheme } from "antd";
import { StyleProvider } from "@ant-design/cssinjs";
import zhCN from "antd/locale/zh_CN";
import enUS from "antd/locale/en_US";
import { useApp, type ThemePref } from "./store";
import { applyLang } from "@/i18n";

// Single AntD theming + locale authority. Reads the app store's `theme` / `lang`
// prefs and projects them onto:
//   • ConfigProvider.algorithm  → light / dark token set
//   • ConfigProvider.token      → red dust brand (#d32029) + semantic status colors
//   • ConfigProvider.locale     → AntD component strings (zh_CN / en_US)
//   • i18next + dayjs           → app copy + date formatting (via applyLang)
// StyleProvider(layer) parks AntD's CSS-in-JS in `@layer antd` so Tailwind
// utilities can override it (see styles/tailwind.css).

function resolveDark(pref: ThemePref): boolean {
  if (pref === "system") {
    return !!window.matchMedia?.("(prefers-color-scheme: dark)").matches;
  }
  return pref === "dark";
}

function useResolvedDark(pref: ThemePref): boolean {
  const [dark, setDark] = useState(() => resolveDark(pref));
  useEffect(() => {
    if (pref !== "system" || !window.matchMedia) {
      setDark(pref === "dark");
      return;
    }
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setDark(mq.matches);
    onChange();
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [pref]);
  return dark;
}

export function AntdProvider({ children }: { children: ReactNode }) {
  const { theme: themePref, lang } = useApp();
  const dark = useResolvedDark(themePref);

  // Keep i18next + dayjs locale in lock-step with the chosen language.
  useEffect(() => {
    applyLang(lang);
  }, [lang]);

  return (
    <StyleProvider layer>
      <ConfigProvider
        locale={lang === "en" ? enUS : zhCN}
        // Match the prototype: the required marker is a red asterisk placed
        // AFTER the label (AntD's default puts it before).
        form={{
          requiredMark: (label, { required }) => (
            <>
              {label}
              {required && <span className="ml-1 text-accent">*</span>}
            </>
          ),
        }}
        theme={{
          algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
          token: {
            colorPrimary: dark ? "#f04a4c" : "#d32029",
            colorInfo: dark ? "#4a90ff" : "#1677ff",
            colorSuccess: dark ? "#3fbd86" : "#22a06b",
            colorWarning: dark ? "#f0b13a" : "#faad14",
            colorError: dark ? "#ff5a5f" : "#cf1322",
            borderRadius: 8,
            // `size="large"` form controls (used in all create/edit drawers)
            // render at the prototype's taller input height.
            controlHeightLG: 42,
            fontFamily:
              '"Ant Sans", "Alibaba PuHuiTi", Inter, -apple-system, "Segoe UI", Arial, sans-serif',
          },
          components: {
            // AntD's Layout ships a dark header/sider by default; map them onto
            // our light surface tokens so the shell is white in light mode and
            // follows the dark algorithm automatically in dark mode.
            Layout: {
              headerBg: "var(--bg)",
              headerColor: "var(--fg)",
              headerHeight: 56,
              headerPadding: "0 16px",
              bodyBg: "var(--surface)",
              siderBg: "var(--bg)",
            },
            Menu: {
              itemBg: "transparent",
            },
          },
        }}
      >
        <AntApp className="h-full">{children}</AntApp>
      </ConfigProvider>
    </StyleProvider>
  );
}
