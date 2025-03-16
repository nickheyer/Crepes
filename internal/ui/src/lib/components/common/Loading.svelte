<script>
    let {
      size = "md", // xs, sm, md, lg
      text = "",   // Optional loading text
      fullscreen = false, // Whether to display as fullscreen overlay
      className = "",
    } = $props();
    const sizeClasses = {
      xs: "loading-xs",
      sm: "loading-sm",
      md: "loading-md",
      lg: "loading-lg"
    };
    const spinnerClasses = `loading loading-spinner ${sizeClasses[size] || sizeClasses.md}`;
  </script>
  
  {#if fullscreen}
    <div class="fixed inset-0 bg-base-900 bg-opacity-50 z-501 flex items-center justify-center">
      <div class="bg-base-800 p-6 rounded-lg shadow-lg flex flex-col items-center">
        <div class={spinnerClasses}></div>
        {#if text}
          <p class="mt-3 text-center text-white">{text}</p>
        {/if}
      </div>
    </div>
  {:else}
    <div class={`flex justify-center items-center py-4 ${className}`}>
      <div class={spinnerClasses}></div>
      {#if text}
        <p class="ml-3">{text}</p>
      {/if}
    </div>
  {/if}
