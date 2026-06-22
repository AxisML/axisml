import { type ComponentType } from "react";
import { Tag, Trash2 } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Card-grid item shared by the model and image repositories — their cards are
// structurally identical, differing only in the leading icon, the badge value,
// and the localized labels. Clicking opens the version drawer; the footer carries
// a version count and a delete affordance.
export function AssetCard({
  icon: Icon,
  name,
  desc,
  badge,
  latest,
  updated,
  versionsText,
  onOpen,
  onDelete,
  deleteLabel,
}: {
  icon: ComponentType<{ className?: string }>;
  name: string;
  desc?: string;
  badge?: string;
  latest: string;
  updated: string;
  versionsText: string;
  onOpen: () => void;
  onDelete: () => void;
  deleteLabel: string;
}) {
  return (
    <Card
      className="cursor-pointer gap-0 p-4 transition-shadow hover:border-primary/30 hover:shadow-md"
      onClick={onOpen}
    >
      <div className="flex items-center gap-2.5">
        <div className="grid size-[38px] shrink-0 place-items-center rounded-[9px] border bg-muted">
          <Icon className="size-[20px] text-muted-foreground" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate font-mono text-sm font-semibold text-foreground">{name}</div>
          {desc && <div className="truncate text-xs text-muted-foreground">{desc}</div>}
        </div>
        {badge && <Badge variant="secondary">{badge}</Badge>}
      </div>
      <div className="mt-2.5 flex items-center gap-2 text-xs">
        <span className="inline-flex items-center gap-1.5 rounded-full border bg-muted px-2 py-0.5 font-mono text-[11.5px] text-foreground/80">
          <Tag className="size-3.5 text-muted-foreground" />
          {latest}
        </span>
        <span className="ml-auto text-muted-foreground">{updated}</span>
      </div>
      <Separator className="mt-3.5 mb-2.5" />
      <div className="flex items-center text-xs text-muted-foreground">
        <span>{versionsText}</span>
        <div className="grow" />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-destructive"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              aria-label={deleteLabel}
            >
              <Trash2 />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{deleteLabel}</TooltipContent>
        </Tooltip>
      </div>
    </Card>
  );
}
