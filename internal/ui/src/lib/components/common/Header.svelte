<script>
  // PROPS
  let {
    title = "",
    subtitle = "",
    backLink = "",
    backLabel = "Back",
    actions = () => {}
  } = $props();
</script>

<div class="mb-6">
  <div class="flex justify-between items-start mb-2">
    <div class="flex items-center">
      {#if backLink}
        <a href={backLink} class="mr-2 btn btn-ghost btn-sm p-1" aria-label={backLabel}>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" />
          </svg>
        </a>
      {/if}
      <h1 class="text-2xl font-bold">{title}</h1>
    </div>
    
    <div class="flex space-x-2">
      {@render actions()}
    </div>
  </div>
  
  {#if subtitle}
    <p class="text-base-content text-opacity-70">{subtitle}</p>
  {/if}
</div>