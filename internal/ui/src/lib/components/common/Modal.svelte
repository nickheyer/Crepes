<script>
  import { fade, fly } from 'svelte/transition';
  import { X } from "lucide-svelte";
  import Button from './Button.svelte';
  
  let {
    title = '',
    size = 'md', // 'sm', 'md', 'lg', 'xl', 'full'
    showClose = true,
    closeOnOverlayClick = true,
    showFooter = true,
    primaryAction = 'Save',
    secondaryAction = 'Cancel',
    primaryVariant = 'primary',
    disabled = false,
    loading = false,
    onclose = () => {},
    onprimaryAction = () => {},
    onsecondaryAction = () => {},
    isOpen = true,
    children,
    footer
  } = $props();
  
  // Get size class based on size prop
  const sizeClasses = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
    full: 'max-w-full mx-4',
  };
  
  let modalSizeClass = sizeClasses[size] || sizeClasses.md;
  
  // Event functions
  function closeModal() {
    onclose();
  }
  
  function handleOverlayClick() {
    if (closeOnOverlayClick) {
      closeModal();
    }
  }
  
  function handlePrimaryAction() {
    onprimaryAction();
  }
  
  function handleSecondaryAction() {
    onsecondaryAction();
  }
  
  function handleKeydown(event) {
    if (event.key === 'Escape') {
      closeModal();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
<div class="modal modal-open z-50" style="background: rgba(0,0,0,0.7);">
  <div class="modal-box {modalSizeClass} bg-base-200">
    {#if title || showClose}
      <div class="flex justify-between items-center mb-4">
        {#if title}
          <h3 class="font-bold text-lg">{title}</h3>
        {:else}
          <div></div>
        {/if}
        
        {#if showClose}
          <button
            class="btn btn-sm btn-circle btn-ghost"
            onclick={closeModal}
          >
            <X class="h-5 w-5" />
          </button>
        {/if}
      </div>
    {/if}
    
    <!-- Modal body -->
    <div>
      {#if typeof children === 'function'}
        {@render children()}
      {:else}
        {children}
      {/if}
    </div>
    
    {#if showFooter}
      <div class="modal-action mt-6">
        {#if typeof footer === 'function'}
          {@render footer()}
        {:else}
          {footer}
        {/if}
        
        <button class="btn btn-outline" onclick={handleSecondaryAction}>
          {secondaryAction}
        </button>
        <button 
          class="btn btn-{primaryVariant}" 
          disabled={disabled || loading} 
          onclick={handlePrimaryAction}
        >
          {#if loading}
            <span class="loading loading-spinner loading-xs mr-2"></span>
          {/if}
          {primaryAction}
        </button>
      </div>
    {/if}
  </div>
  
  <!-- Background overlay click handler -->
  {#if closeOnOverlayClick}
    <div
      class="modal-backdrop cursor-pointer"
      onclick={handleOverlayClick}
      onkeydown={() => {}}
      role="button"
      aria-label="Close modal"
      tabindex="0"
    ></div>
  {/if}
</div>
{/if}
