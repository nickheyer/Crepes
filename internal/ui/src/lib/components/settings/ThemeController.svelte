<script>
  import { onMount, afterUpdate } from "svelte";
  import { writable } from "svelte/store";
  import { initTheme, availableThemes } from "$lib/stores/uiStore";

  // EXPORT THEME PROP FOR BINDING
  export let theme = "default";

  // INITIALIZE STORE FOR PERSISTENT STATE
  const themeStore = writable(theme);

  // SYNC THE THEME WITH HTML DATA-THEME ATTRIBUTE
  function updateTheme(newTheme) {
    document.documentElement.setAttribute("data-theme", newTheme);
    localStorage.setItem("theme", newTheme);
    themeStore.set(newTheme);
    theme = newTheme;
    initTheme();
  }

  // LOAD THEME FROM LOCAL STORAGE OR USE DEFAULT
  onMount(() => {
    const savedTheme = localStorage.getItem("theme");
    if (savedTheme) {
      updateTheme(savedTheme);
    } else {
      updateTheme(theme);
    }
  });

  // UPDATE THEME WHEN PROP CHANGES
  afterUpdate(() => {
    const currentValue = document.documentElement.getAttribute("data-theme");
    if (currentValue !== theme) {
      updateTheme(theme);
    }
  });

  // HANDLE THEME CHANGE
  function handleThemeChange(e) {
    const newTheme = e.target.value;
    updateTheme(newTheme);
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
      checked={theme === themeOption.value}
      onchange={handleThemeChange}
    />
  {/each}
</div>
