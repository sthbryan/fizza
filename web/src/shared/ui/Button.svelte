<script lang="ts">
  import type { HTMLButtonAttributes } from "svelte/elements";
  import { cn } from "@/lib/cn";

  type Variant = "primary" | "secondary" | "ghost" | "danger" | "destructive" | "icon";
  type Size = "sm" | "md" | "lg";

  interface Props extends HTMLButtonAttributes {
    variant?: Variant;
    size?: Size;
    class?: string;
  }

  let {
    variant = "secondary",
    size = "md",
    class: className = "",
    children,
    disabled,
    ...rest
  }: Props = $props();

  const variants: Record<Variant, string> = {
    primary:
      "bg-[var(--color-text-display)] text-[var(--color-bg)] hover:opacity-90",
    secondary:
      "bg-transparent text-[var(--color-text)] border border-[var(--color-border-visible)] hover:border-[var(--color-text-muted)]",
    ghost:
      "bg-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text)]",
    danger:
      "bg-transparent text-[var(--color-danger)] border border-[var(--color-danger)] hover:bg-[var(--color-danger)] hover:text-[var(--color-text-display)]",
    destructive:
      "bg-transparent text-[var(--color-danger)] border border-[var(--color-danger)] hover:bg-[var(--color-danger)] hover:text-[var(--color-text-display)]",
    icon:
      "bg-transparent text-[var(--color-text-muted)] hover:text-[var(--color-text)]",
  };

  const sizes: Record<Size, string> = {
    sm: "h-9 px-4 text-[11px] tracking-[0.08em] gap-2",
    md: "h-11 px-5 text-xs gap-2.5",
    lg: "h-12 px-6 text-sm gap-3",
  };

  const radius = $derived(
    variant === "icon"
      ? "rounded-md"
      : variant === "primary" ||
          variant === "secondary" ||
          variant === "danger" ||
          variant === "destructive"
        ? "rounded-full"
        : "rounded-none"
  );
</script>

<button
  class={cn(
    "inline-flex cursor-pointer items-center justify-center font-mono font-medium uppercase transition-colors",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-text-display)]/30",
    "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-30",
    variants[variant],
    sizes[size],
    radius,
    variant === "icon" && "h-10 w-10 px-0 rounded-md",
    className
  )}
  {disabled}
  {...rest}
>
  {@render children?.()}
</button>