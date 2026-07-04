import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

// Shared "load more" control for server-paginated lists. Renders nothing when
// there is no next page, so callers can drop it in unconditionally.
export function LoadMore({
  hasMore,
  loading,
  onClick,
}: {
  hasMore: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  const { t } = useTranslation();
  if (!hasMore) return null;
  return (
    <div className="mt-4 flex justify-center">
      <Button variant="outline" size="sm" onClick={onClick} disabled={loading}>
        {loading && <Spinner data-icon="inline-start" />}
        {t("common.loadMore")}
      </Button>
    </div>
  );
}
