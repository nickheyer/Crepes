<script>
    import { fade } from "svelte/transition";
    import { 
        Briefcase, 
        Play, 
        Image as ImageIcon, 
        Database, 
        BarChart
    } from "lucide-svelte";
    
    // PROPS
    let {
        title = "",
        value = 0,
        icon = "",
        color = "primary",
        trend = null,
        href = "",
    } = $props();
    
    // DETERMINE ICON COMPONENT
    function getIconComponent() {
        switch (icon) {
            case "briefcase":
                return Briefcase;
            case "play":
                return Play;
            case "photo":
                return ImageIcon;
            case "database":
                return Database;
            default:
                return BarChart;
        }
    }
    
    const IconComponent = getIconComponent();
    
    // DETERMINE COLOR CLASS
    function getColorClass() {
        switch (color) {
            case "primary":
                return "bg-primary text-primary-content";
            case "success":
                return "bg-success text-success-content";
            case "warning":
                return "bg-warning text-warning-content";
            case "danger":
                return "bg-error text-error-content";
            default:
                return "bg-primary text-primary-content";
        }
    }
    
    // FORMAT TREND WITH SIGN
    function formatTrend(trend) {
        if (trend === null) return null;
        const sign = trend >= 0 ? "+" : "";
        return `${sign}${trend}%`;
    }
    
    // GET TREND COLOR
    function getTrendColor(trend) {
        if (trend === null) return "";
        return trend >= 0 ? "text-success" : "text-error";
    }
</script>

<a {href} class="block transition-transform hover:scale-102">
    <div class="stat bg-base-200 shadow-xl hover:shadow-2xl transition-all duration-300 rounded-xl" in:fade={{ duration: 200 }}>
        <div class="stat-figure">
            <div class={`p-3 rounded-lg ${getColorClass()}`}>
                <IconComponent size={24} />
            </div>
        </div>
        
        <div class="stat-title font-medium text-base-content opacity-70">{title}</div>
        
        <div class="stat-value text-3xl font-bold">{value}</div>
        
        {#if trend !== null}
            <div class={`stat-desc ${getTrendColor(trend)} flex items-center text-sm font-medium mt-1`}>
                <span>{formatTrend(trend)}</span>
                <span class="ml-1">
                    {#if trend >= 0}
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z" clip-rule="evenodd" />
                        </svg>
                    {:else}
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                        </svg>
                    {/if}
                </span>
            </div>
        {/if}
    </div>
</a>
