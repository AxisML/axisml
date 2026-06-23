import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { cn } from "@/lib/utils";

// Centered loading spinner used while a page/pane's primary query is in flight.
// Replaces the ~9 ad-hoc `grid place-items-center py-NN` spinner blocks (which
// used inconsistent py-24/20/16/10) with one treatment.
export function PageLoading({ className }: { className?: string }) {
  return (
    <div className={cn("grid place-items-center py-24", className)}>
      <Spinner className="size-7 text-muted-foreground" />
    </div>
  );
}

// Standard detail-page error / empty state. Replaces the four ad-hoc treatments
// (Card+CardContent text, Alert, raw div) with one bordered Empty.
export function DetailError({ message, className }: { message: string; className?: string }) {
  return (
    <Empty className={cn("border", className)}>
      <EmptyHeader>
        <EmptyTitle>{message}</EmptyTitle>
      </EmptyHeader>
    </Empty>
  );
}
