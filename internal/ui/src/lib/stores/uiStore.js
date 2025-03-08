import { writable } from 'svelte/store';

// SIDEBAR STATE
export const isSidebarOpen = writable(false);

// TOAST NOTIFICATIONS
export const toasts = writable([]);

// ADD A TOAST NOTIFICATION
export function addToast(message, type = 'info', duration = 4000) {
  const id = Date.now().toString();

  toasts.update(all => [
    ...all,
    { id, message, type, duration }
  ]);

  return id;
}

// REMOVE A TOAST NOTIFICATION
export function removeToast(id) {
  toasts.update(all => all.filter(t => t.id !== id));
}

// MODAL STATES
export const activeModals = writable(new Set());

// OPEN A MODAL
export function openModal(modalId) {
  activeModals.update(modals => {
    modals.add(modalId);
    return modals;
  });
}

// CLOSE A MODAL
export function closeModal(modalId) {
  activeModals.update(modals => {
    modals.delete(modalId);
    return modals;
  });
}

// THEME MANAGEMENT
export let currentTheme = writable('default');

// APPLY A THEME
export function applyTheme(theme) {
  // UPDATE THE STATE
  currentTheme = theme;

  // APPLY TO DOM
  document.documentElement.setAttribute('data-theme', theme);

  // SAVE TO LOCAL STORAGE
  localStorage.setItem('theme', theme);

  // PUBLISH THEME CHANGE EVENT
  window.dispatchEvent(new CustomEvent('theme-changed', { detail: { theme } }));
}

// GET THEME FROM LOCAL STORAGE OR API
export async function initTheme() {
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

