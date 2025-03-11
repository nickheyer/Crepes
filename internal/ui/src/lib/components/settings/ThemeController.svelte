<script>
  import { onMount } from "svelte";
  import { initTheme, availableThemes, applyTheme, state as uiState } from "$lib/stores/uiStore.svelte";

  // HANDLE THEME CHANGE
  function handleThemeChange(e) {
    const newTheme = e.target.value;
    if (newTheme !== uiState.currentTheme) {
      applyTheme(newTheme);
    }
  }
</script>

<div class="join join-vertical">
  {#each availableThemes as themeOption}
    <input
      type="radio"
      name="theme-buttons"
      class="btn theme-controller join-item"
      aria-label={themeOption.label}
      value={themeOption.value}
      checked={uiState.currentTheme === themeOption.value}
      onchange={handleThemeChange}
    />
  {/each}
</div>
