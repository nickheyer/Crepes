<script>
    import { createEventDispatcher } from "svelte";
    
    let {
        title = "",
        subtitle = "",
        noPadding = false,
        variant = "default", // 'default', 'elevated', 'border'
        clickable = false,
        children,
        footer
    } = $props();
    
    // GENERATE CLASS FROM PROPS
    const cardClass = [
        "card bg-base-200",
        variant === "elevated" ? "shadow-lg" : "",
        variant === "border" ? "border border-base-300" : "",
        clickable ? "cursor-pointer hover:shadow-lg transition-all duration-200" : "",
    ].join(" ");
    
    const bodyClass = noPadding ? "card-body p-0" : "card-body";
    
    const dispatch = createEventDispatcher();
    
    function handleClick() {
        if (clickable) {
            dispatch("click");
        }
    }
</script>
<div
    class={cardClass}
    onclick={handleClick}
    {...{
        title,
        subtitle,
        noPadding,
        variant,
        clickable,
    }}
>
    {#if title}
        <div class="card-title p-4 border-b border-base-300">
            <h3 class="text-lg font-medium">{title}</h3>
            {#if subtitle}
                <p class="mt-1 text-sm opacity-70">{subtitle}</p>
            {/if}
        </div>
    {/if}
    <div class={bodyClass}>
        {@render children()}
    </div>
    {#if footer}
        <div class="card-actions bg-base-300 px-4 py-3 border-t border-base-300">
            {@render footer()}
        </div>
    {/if}
</div>
