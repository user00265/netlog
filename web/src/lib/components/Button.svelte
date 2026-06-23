<script lang="ts">
  import type { Snippet } from "svelte";

  type Variant = "primary" | "blue" | "green" | "yellow" | "gray";

  let {
    variant = "primary",
    size = "md",
    type = "button",
    disabled = false,
    onclick,
    class: cls = "",
    children,
  }: {
    variant?: Variant;
    size?: "sm" | "md";
    type?: "button" | "submit";
    disabled?: boolean;
    onclick?: (e: MouseEvent) => void;
    class?: string;
    children: Snippet;
  } = $props();

  const variantClass: Record<Variant, string> = {
    primary: "nl-btn-primary",
    blue: "nl-btn-blue",
    green: "nl-btn-green",
    yellow: "nl-btn-yellow",
    gray: "nl-btn-gray",
  };

  const sizeClass = $derived(size === "sm" ? "px-3 py-1 text-xs" : "");
</script>

<button
  {type}
  {disabled}
  {onclick}
  class={`nl-btn ${variantClass[variant]} ${sizeClass} ${cls}`}
>
  {@render children()}
</button>
