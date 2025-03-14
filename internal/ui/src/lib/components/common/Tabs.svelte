<script>
  // PROPS
  let { 
    tabs = [],  // Array of tab objects: {id, label, icon}
    activeTab = null,  // Currently active tab id
    size = "default", // "default", "sm", or "lg"
    fullWidth = false, // Whether tabs should take full width
    variant = "default", // "default", "pill", or "underline"
    onChange = () => {},
    children
  } = $props();
  
  // INITIALIZE ACTIVE TAB IF NOT SET
  $effect(() => {
    if (!activeTab && tabs.length > 0) {
      activeTab = tabs[0].id;
    }
  });
  
  // HANDLE TAB CLICK
  function handleTabClick(tabId) {
    activeTab = tabId;
    onChange({ tabId });
  }
  
  // SIZE CLASSES
  let sizeClasses = $derived(() => {
    const classes = {
      default: "py-2 px-4",
      sm: "py-1 px-3 text-sm",
      lg: "py-3 px-5 text-lg"
    };
    return classes[size] || classes.default;
  });
  
  // VARIANT CLASSES
  let variantClasses = $derived(() => {
    const classes = {
      default: "hover:bg-base-700 rounded-t-lg border-transparent border-b-2",
      pill: "hover:bg-base-700 rounded-full",
      underline: "hover:bg-transparent border-transparent border-b-2"
    };
    return classes[variant] || classes.default;
  });
  
  // ACTIVE TAB CLASSES
  let activeClasses = $derived(() => {
    const classes = {
      default: "bg-base-700 border-primary-500 border-b-2",
      pill: "bg-primary-600 text-white",
      underline: "border-primary-500 border-b-2"
    };
    return classes[variant] || classes.default;
  });
</script>

<div class="tabs-container">
  <div class="border-b border-base-700 mb-4">
    <div class={`flex ${fullWidth ? 'w-full' : ''} -mb-px`}>
      {#each tabs as tab}
        <button
          class={`${sizeClasses} ${variantClasses} font-medium transition-colors flex items-center focus-visible:ring-2 focus-visible:ring-primary-500
                ${activeTab === tab.id ? activeClasses : ''} 
                ${fullWidth ? 'flex-1 justify-center' : ''}`}
          onclick={() => handleTabClick(tab.id)}
          aria-selected={activeTab === tab.id}
          role="tab"
        >
          {#if tab.icon}
            <tab.icon class="h-4 w-4 mr-2"></tab.icon>
          {/if}
          {tab.label}
        </button>
      {/each}
    </div>
  </div>

  <div class="tab-content">
    {#if typeof children === 'function'}
      {@render children()}
    {:else}
      {children}
    {/if}
  </div>
</div>