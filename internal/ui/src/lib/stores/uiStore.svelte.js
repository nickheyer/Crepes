// UI STORE USING SVELTE 5 RUNES
export const state = $state({
  isSidebarOpen: false,
  toasts: [],
  activeModals: new Set(),
  currentTheme: 'default'
});

// SIDEBAR TOGGLE
export function toggleSidebar() {
  state.isSidebarOpen = !state.isSidebarOpen;
}

// ADD A TOAST NOTIFICATION
export function addToast(message, type = 'info', duration = 4000) {
  const id = Date.now().toString();
  state.toasts = [
    ...state.toasts,
    { id, message, type, duration }
  ];
  
  // AUTO-REMOVE TOAST AFTER DURATION
  if (duration > 0) {
    setTimeout(() => {
      removeToast(id);
    }, duration);
  }
  
  return id;
}

// REMOVE A TOAST NOTIFICATION
export function removeToast(id) {
  state.toasts = state.toasts.filter(t => t.id !== id);
}

// OPEN A MODAL
export function openModal(modalId) {
  const newModals = new Set(state.activeModals);
  newModals.add(modalId);
  state.activeModals = newModals;
}

// CLOSE A MODAL
export function closeModal(modalId) {
  const newModals = new Set(state.activeModals);
  newModals.delete(modalId);
  state.activeModals = newModals;
}

// APPLY A THEME
export function applyTheme(theme) {
  // UPDATE THE STATE
  state.currentTheme = theme;

  // SAVE TO LOCAL STORAGE
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('theme', theme);
  }

  // PUBLISH THEME CHANGE EVENT
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('theme-changed', { detail: { theme } }));
  }
}

// GET THEME FROM LOCAL STORAGE OR API
export async function initTheme() {
  if (typeof localStorage === 'undefined' || typeof fetch === 'undefined') {
    return;
  }
  
  // ATTEMPT TO LOAD THEME FROM LOCAL STORAGE
  const savedTheme = localStorage.getItem('theme');
  if (savedTheme) {
    // USE SAVED THEME IF AVAILABLE
    applyTheme(savedTheme);
  } else {
    // TRY TO FETCH FROM SETTINGS API
    try {
      const response = await fetch('/api/settings');
      if (response.ok) {
        const data = await response.json();
        if (data.success && data.data.userConfig && data.data.userConfig.theme) {
          applyTheme(data.data.userConfig.theme);
        } else {
          // FALLBACK TO DEFAULT THEME
          applyTheme('default');
        }
      }
    } catch (error) {
      console.error('Failed to load theme from API:', error);
      // FALLBACK TO DEFAULT THEME
      applyTheme('default');
    }
  }
}

// DEFINE AVAILABLE THEMES
export const availableThemes = [
  { value: "default", label: "Default" },
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
  { value: "dracula", label: "Dracula" },
  { value: "cyberpunk", label: "Cyberpunk" },
  { value: "valentine", label: "Valentine" },
  { value: "aqua", label: "Aqua" },
  { value: "night", label: "Night" },
];
