<script lang="ts">
  import type { HTMLButtonAttributes } from "svelte/elements";
  import { cn } from "@/lib/cn";

  type Variant = "primary" | "secondary" | "ghost" | "danger" | "icon";
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
      "bg-[var(--color-accent)] text-white hover:brightness-110 active:brightness-95",
    secondary:
      "bg-[var(--color-bg-soft)] text-[var(--color-text)] border border-[var(--color-border)] hover:bg-[var(--color-bg-hover)]",
    ghost:
      "bg-transparent text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text)]",
    danger:
      "bg-transparent text-[var(--color-danger)] border border-[var(--color-danger)]/25 hover:bg-[var(--color-danger)]/10",
    icon: "bg-transparent text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text)]",
  };

  const sizes: Record<Size, string> = {
    sm: "h-9 px-3.5 text-sm rounded-xl gap-1.5",
    md: "h-10 px-4 text-sm rounded-xl gap-2",
    lg: "h-11 px-5 text-base rounded-2xl gap-2",
  };
</script>

<button
  class={cn(
    "inline-flex cursor-pointer items-center justify-center font-medium tracking-tight transition",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)]/40",
    "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-40",
    variants[variant],
    sizes[size],
    variant === "icon" && "h-10 w-10 px-0",
    className
  )}
  {disabled}
  {...rest}
>
  {@render children?.()}
</button>
