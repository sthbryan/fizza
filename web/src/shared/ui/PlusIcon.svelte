<script lang="ts">
  interface Props {
    size?: number;
    stroke?: number;
    class?: string;
  }

  let {
    size = 16,
    stroke = 1.5,
    class: className = "",
  }: Props = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  aria-hidden="true"
  class={className}
>
  <path
    d="M12 5v14M5 12h14"
    stroke="currentColor"
    stroke-width={stroke}
    stroke-linecap="round"
  />
</svg>