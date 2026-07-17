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
    size = "sm",
    class: className = "",
    children,
    disabled,
    ...rest
  }: Props = $props();

  const variants: Record<Variant, string> = {
    primary: "bg-white text-black hover:opacity-90",
    secondary:
      "bg-transparent text-neutral-200 border border-neutral-700 hover:border-neutral-500",
    ghost: "bg-transparent text-neutral-400 hover:text-neutral-200",
    danger:
      "bg-transparent text-red-500 border border-red-500 hover:bg-red-500 hover:text-white",
    destructive:
      "bg-transparent text-red-500 border border-red-500 hover:bg-red-500 hover:text-white",
    icon: "bg-transparent text-neutral-500 hover:text-neutral-200",
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
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/30",
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