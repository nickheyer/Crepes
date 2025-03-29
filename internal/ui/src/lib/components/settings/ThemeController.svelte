<script>
  import { onMount } from "svelte";
  import { initTheme, availableThemes, applyTheme, state as uiState } from "$lib/stores/uiStore.svelte";

  // HANDLE THEME CHANGE
  function handleThemeChange(e) {
    const newTheme = e.target.value;
    console.log(`THEME CHANGING: ${uiState.theme} -> ${newTheme}`);
    if (newTheme !== uiState.theme) {
      uiState.theme = newTheme;
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
      checked={uiState.theme === themeOption.value}
      onchange={handleThemeChange}
    />
  {/each}
</div>
