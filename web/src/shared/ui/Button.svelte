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
      "bg-transparent text-accent border border-accent hover:text-white",
    destructive:
      "bg-transparent text-accent border border-accent hover:text-white",
    icon: "bg-transparent text-neutral-500 hover:text-neutral-200",
  };

  const sizes: Record<Size, string> = {
    sm: "min-h-11 px-6 text-label gap-2",
    md: "min-h-11 px-6 text-label gap-2",
    lg: "min-h-11 px-6 text-label gap-2",
  };

</script>

<button
  class={cn(
    "inline-flex cursor-pointer items-center justify-center font-mono font-medium uppercase transition-colors",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/30",
    "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-30",
    variants[variant],
    sizes[size],
    variant === "icon" && "h-11 w-11 px-0",
    className
  )}
  {disabled}
  {...rest}
>
  {@render children?.()}
</button>