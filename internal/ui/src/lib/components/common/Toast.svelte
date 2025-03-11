<script>
    import { fade, fly } from "svelte/transition";
    import {
        AlertCircle,
        CheckCircle,
        XCircle,
        Info,
        X
    } from "lucide-svelte";
    import { removeToast } from "$lib/stores/uiStore.svelte";
    
    // Props
    let {
        type = "info", // 'info', 'success', 'warning', 'error'
        message = "",
        id = "",
        duration = 4000,
    } = $props();
    
    // Auto dismiss
    let timer;
    $effect(() => {
        if (duration > 0) {
            timer = setTimeout(() => {
                dismiss();
            }, duration);
        }
        return () => {
            if (timer) clearTimeout(timer);
        };
    });
    
    function dismiss() {
        removeToast(id);
    }
    
    // Get appropriate icon and color based on type
    const typeConfig = {
        info: {
            icon: Info,
            alertClass: "alert-info",
        },
        success: {
            icon: CheckCircle,
            alertClass: "alert-success",
        },
        warning: {
            icon: AlertCircle,
            alertClass: "alert-warning",
        },
        error: {
            icon: XCircle,
            alertClass: "alert-error",
        },
    };
    
    const config = typeConfig[type] || typeConfig.info;
</script>

<div
    in:fly={{ x: 20, duration: 300 }}
    out:fade={{ duration: 200 }}
    class={`alert ${config.alertClass} min-w-[300px] max-w-md shadow-lg`}
>
    <div class="flex w-full justify-between">
        <div class="flex items-center">
            <config.icon size={16} />
            <span class="ml-2">{message}</span>
        </div>
        <button
            class="btn btn-ghost btn-xs"
            onclick={dismiss}
        >
            <X size={16} />
        </button>
    </div>
</div>
