import { Fragment, type ReactNode } from "react";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

// Standard page chrome shared by every list/detail page: a breadcrumb, a title
// with optional subtitle, and a right-aligned action slot — followed by the page
// body. Replaces the prototype's hand-rolled .breadcrumb / .page-head markup.
export function PageContainer({
  breadcrumb,
  title,
  subtitle,
  extra,
  children,
}: {
  breadcrumb?: string[];
  title: ReactNode;
  subtitle?: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="mx-auto max-w-[1200px] p-6">
      {breadcrumb && breadcrumb.length > 0 && (
        <Breadcrumb className="mb-3">
          <BreadcrumbList>
            {breadcrumb.map((b, i) => (
              <Fragment key={i}>
                {i > 0 && <BreadcrumbSeparator />}
                <BreadcrumbItem>
                  <BreadcrumbPage
                    className={i < breadcrumb.length - 1 ? "text-muted-foreground" : undefined}
                  >
                    {b}
                  </BreadcrumbPage>
                </BreadcrumbItem>
              </Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      )}
      <div className="mb-5 flex items-start justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          {subtitle && (
            <p className="mt-1.5 max-w-3xl text-[13px] leading-relaxed text-muted-foreground">
              {subtitle}
            </p>
          )}
        </div>
        {extra && <div className="shrink-0">{extra}</div>}
      </div>
      {children}
    </div>
  );
}
