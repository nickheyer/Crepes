<script>
  import { createEventDispatcher } from "svelte";
  
  // DEFINE PROPS USING $PROPS RUNE
  let {
      variant = 'primary',
      size = 'md',
      disabled = false,
      fullWidth = false,
      loading = false,
      type = 'button',
      onclick = null,
      children
  } = $props();
  
  // MAP VARIANTS TO DAISYUI CLASSES
  const variantClasses = {
    primary: 'btn-primary',
    secondary: 'btn-secondary',
    success: 'btn-success',
    danger: 'btn-error',
    warning: 'btn-warning',
    outline: 'btn-outline',
    ghost: 'btn-ghost'
  };
  
  // MAP SIZES TO DAISYUI CLASSES
  const sizeClasses = {
    xs: 'btn-xs',
    sm: 'btn-sm',
    md: 'btn-md',
    lg: 'btn-lg',
    xl: 'btn-xl'
  };
  
  // GENERATE CLASS FROM PROPS
  const buttonClasses = [
    'btn',
    variantClasses[variant] || variantClasses.primary,
    sizeClasses[size] || sizeClasses.md,
    fullWidth ? 'w-full' : '',
    loading ? 'loading' : ''
  ].join(' ');
  
  const dispatch = createEventDispatcher();
</script>
<button
  {type}
  class={buttonClasses}
  {disabled}
  onclick={onclick || (() => dispatch('click'))}
>
  {@render children()}
</button>
