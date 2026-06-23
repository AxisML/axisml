import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import "dayjs/locale/zh-cn";
import { zhCN } from "./locales/zh";
import { enUS } from "./locales/en";
import type { Lang } from "@/app/store";

// Front-end owns all localization (the backend stays locale-neutral, returning
// only stable machine-readable identifiers — see system_design/platform/frontend.md §5).
// Two catalogs ship today: zh-CN (default) and en-US.
//
// Catalog structure: `locales/{zh,en}.ts` hold the shell/common core; each page
// drops a `locales/features/<feature>.{zh,en}.ts` exporting `default { feature: {...} }`.
// They're glob-merged below so feature catalogs never touch shared files.

dayjs.extend(relativeTime);
dayjs.extend(utc);

export const LANGS = {
  zh: "zh-CN",
  en: "en-US",
} as const;

type CatalogModule = { default: Record<string, unknown> };
const mergeFeatures = (mods: Record<string, unknown>) =>
  Object.assign({}, ...Object.values(mods).map((m) => (m as CatalogModule).default));

const zhFeatures = import.meta.glob("./locales/features/*.zh.ts", { eager: true });
const enFeatures = import.meta.glob("./locales/features/*.en.ts", { eager: true });

const zhResources = { ...zhCN, ...mergeFeatures(zhFeatures) };
const enResources = { ...enUS, ...mergeFeatures(enFeatures) };

void i18n.use(initReactI18next).init({
  resources: {
    "zh-CN": { translation: zhResources },
    "en-US": { translation: enResources },
  },
  lng: localStorage.getItem("axisml.lang") === "en" ? "en-US" : "zh-CN",
  fallbackLng: "zh-CN",
  interpolation: { escapeValue: false },
  returnNull: false,
});

// Keep i18next + dayjs locale in lock-step with the app store's `lang` pref.
export function applyLang(lang: Lang) {
  void i18n.changeLanguage(LANGS[lang]);
  dayjs.locale(lang === "en" ? "en" : "zh-cn");
  document.documentElement.setAttribute("lang", LANGS[lang]);
}

// Initialize dayjs locale to match the persisted preference on first load.
applyLang(localStorage.getItem("axisml.lang") === "en" ? "en" : "zh");

export default i18n;
