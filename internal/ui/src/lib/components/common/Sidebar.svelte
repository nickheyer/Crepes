<script>
  import {
    LayoutDashboard,
    BarChart,
    Image,
    Settings,
    Copy,
    HelpCircle,
    User
  } from "lucide-svelte";
  
  // PROPS
  let { currentPage = "dashboard" } = $props();
  
  // MENU ITEMS
  const menuItems = [
    { id: "dashboard", label: "Dashboard", icon: LayoutDashboard, href: "/" },
    { id: "jobs", label: "Jobs", icon: BarChart, href: "/jobs" },
    { id: "assets", label: "Assets Gallery", icon: Image, href: "/assets" },
    { id: "templates", label: "Templates", icon: Copy, href: "/templates" },
    { id: "settings", label: "Settings", icon: Settings, href: "/settings" },
    { id: "help", label: "Help", icon: HelpCircle, href: "/help" },
  ];
</script>
<ul class="menu p-4 w-64 h-full bg-base-300 text-base-content">
  <!-- Logo -->
  <div class="flex items-center justify-center h-16 mb-4">
    <div class="text-xl font-bold flex items-center">
      <div class="bg-primary text-primary-content h-8 w-8 flex items-center justify-center rounded-md mr-2">C</div>
      <span>CREPES</span>
    </div>
  </div>
  
  <!-- Nav Links -->
  {#each menuItems as item}
    <li>
      <a
        href={item.href}
        class="flex items-center gap-3 py-2 {currentPage === item.id ? 'active' : ''}"
        data-sveltekit-preload-data="hover"
        aria-current={currentPage === item.id ? "page" : undefined}
      >
        <item.icon size={18} />
        <span>{item.label}</span>
      </a>
    </li>
  {/each}
  
  <!-- Footer -->
  <div class="mt-auto border-t border-base-content/20 pt-4">
    <div class="flex items-center gap-3 px-2">
      <div class="avatar">
        <div class="bg-primary text-primary-content rounded-full w-8 h-8 flex items-center justify-center">
          <User size={16} />
        </div>
      </div>
      <div>
        <p class="text-sm font-medium">User Settings</p>
        <p class="text-xs opacity-70">Manage your account</p>
      </div>
    </div>
  </div>
</ul>
