import type { ReactNode } from "react";
import { Breadcrumb, Typography, Flex } from "antd";

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
        <Breadcrumb className="mb-3" items={breadcrumb.map((b) => ({ title: b }))} />
      )}
      <Flex align="flex-start" justify="space-between" gap={16} className="mb-5">
        <div className="min-w-0">
          <Typography.Title level={2} className="!mb-1.5 !font-bold">
            {title}
          </Typography.Title>
          {subtitle && (
            <Typography.Paragraph type="secondary" className="!mb-0 max-w-3xl !text-[13px] !leading-relaxed">
              {subtitle}
            </Typography.Paragraph>
          )}
        </div>
        {extra && <div className="shrink-0">{extra}</div>}
      </Flex>
      {children}
    </div>
  );
}
