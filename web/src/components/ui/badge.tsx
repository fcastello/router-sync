import { cn } from "@/lib/utils";
import { HTMLAttributes } from "react";

export function Badge({
  className,
  variant = "default",
  ...props
}: HTMLAttributes<HTMLSpanElement> & {
  variant?: "default" | "success" | "muted" | "warn" | "secondary";
}) {
  const styles = {
    default: "bg-primary/10 text-primary",
    success: "bg-green-100 text-success",
    muted: "bg-muted text-muted-foreground",
    warn: "bg-amber-100 text-amber-800",
    secondary: "bg-secondary text-secondary-foreground",
  };
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        styles[variant],
        className,
      )}
      {...props}
    />
  );
}
