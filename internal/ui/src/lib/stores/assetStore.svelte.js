import { addToast } from './uiStore.svelte';
import { fetchAssets, fetchAssetDetails, deleteAsset, regenerateThumbnail } from '$lib/utils/api';

export const state = $state({
  assets: [],
  selectedAsset: null,
  assetsLoading: false,
  assetViewerOpen: false,
  assetFilters: {
    type: '',
    jobId: '',
    search: '',
    dateRange: {
      from: null,
      to: null
    },
    sortBy: 'date',
    sortDirection: 'desc'
  },
  assetCounts: {
    total: 0,
    image: 0,
    video: 0,
    audio: 0,
    document: 0
  }
});

const filteredAssetsDer = $derived(() => {
  // ENSURE ASSETS IS ALWAYS AN ARRAY
  let assetArray = Array.isArray(state.assets) ? state.assets : [];
  let result = [...assetArray];
  const filters = state.assetFilters;
  
  // APPLY TYPE FILTER
  if (filters.type) {
    result = result.filter(asset => asset.type === filters.type);
  }
  
  // APPLY JOB FILTER
  if (filters.jobId) {
    result = result.filter(asset => asset.jobId === filters.jobId);
  }
  
  // APPLY TEXT SEARCH
  if (filters.search) {
    const searchLower = filters.search.toLowerCase();
    result = result.filter(asset => 
      (asset.title && asset.title.toLowerCase().includes(searchLower)) ||
      (asset.description && asset.description.toLowerCase().includes(searchLower)) ||
      (asset.url && asset.url.toLowerCase().includes(searchLower))
    );
  }
  
  // APPLY DATE RANGE FILTER
  if (filters.dateRange.from) {
    const fromDate = new Date(filters.dateRange.from);
    result = result.filter(asset => 
      asset.date && new Date(asset.date) >= fromDate
    );
  }
  if (filters.dateRange.to) {
    const toDate = new Date(filters.dateRange.to);
    result = result.filter(asset => 
      asset.date && new Date(asset.date) <= toDate
    );
  }
  
  // APPLY SORTING
  result.sort((a, b) => {
    const direction = filters.sortDirection === 'asc' ? 1 : -1;
    switch (filters.sortBy) {
      case 'date':
        return direction * ((new Date(a.date || 0)) - (new Date(b.date || 0)));
      case 'title':
        return direction * ((a.title || '').localeCompare(b.title || ''));
      case 'type':
        return direction * ((a.type || '').localeCompare(b.type || ''));
      case 'size':
        return direction * ((a.size || 0) - (b.size || 0));
      default:
        return 0;
    }
  });
  
  return result;
});

export const filteredAssets = () => filteredAssetsDer();

const assetsByTypeDer = $derived(() => {
  const groups = {};
  // ENSURE ASSETS IS ALWAYS AN ARRAY
  const assetArray = Array.isArray(state.assets) ? state.assets : [];

  assetArray.forEach(asset => {
    const type = asset.type || 'unknown';
    if (!groups[type]) {
      groups[type] = [];
    }
    groups[type].push(asset);
  });
  return groups;
});

export const assetsByType = () => assetsByTypeDer();

// LOAD ASSETS FROM API
export async function loadAssets(filters = {}) {
  state.assetsLoading = true;
  try {
    const data = await fetchAssets(filters);
    
    // HANDLE NEW API RESPONSE FORMAT
    if (data && data.assets) {
      state.assets = Array.isArray(data.assets) ? data.assets : [];
      
      // UPDATE ASSET COUNTS
      if (data.counts) {
        state.assetCounts = data.counts;
      }
    } else {
      // FALLBACK FOR OLD FORMAT
      state.assets = Array.isArray(data) ? data : [];
    }
    
    return data;
  } catch (error) {
    // SET EMPTY ARRAY ON ERROR
    state.assets = [];
    return { assets: [], counts: {} };
  } finally {
    state.assetsLoading = false;
  }
}

// LOAD ASSET DETAILS
export async function loadAssetDetails(assetId) {
  try {
    const asset = await fetchAssetDetails(assetId);
    state.selectedAsset = asset;
    return asset;
  } catch (error) {
    throw error;
  }
}

// DELETE AN ASSET
export async function removeAsset(assetId) {
  try {
    await deleteAsset(assetId);
    // UPDATE ASSETS STORE SAFELY
    const updatedAssets = Array.isArray(state.assets)
      ? state.assets.filter(asset => asset.id !== assetId)
      : [];
    state.assets = updatedAssets;

    // UPDATE COUNTS
    const asset = state.assets.find(a => a.id === assetId);
    if (asset && state.assetCounts[asset.type]) {
      state.assetCounts[asset.type]--;
    }
    state.assetCounts.total--;
  } catch (error) {
    throw error;
  }
}

// REGENERATE THUMBNAIL FOR AN ASSET
export async function regenerateAssetThumbnail(assetId) {
  try {
    const result = await regenerateThumbnail(assetId);
    // UPDATE ASSET IN STORE SAFELY
    state.assets = Array.isArray(state.assets)
      ? state.assets.map(asset =>
        asset.id === assetId
          ? { ...asset, thumbnailPath: result.thumbnailPath }
          : asset
      )
      : [];

    // UPDATE SELECTED ASSET IF IT'S THE SAME ONE
    if (state.selectedAsset && state.selectedAsset.id === assetId) {
      state.selectedAsset = {
        ...state.selectedAsset,
        thumbnailPath: result.thumbnailPath
      };
    }
    return result.thumbnailPath;
  } catch (error) {
    throw error;
  }
}

// UPDATE FILTERS
export function updateFilters(newFilters) {
  state.assetFilters = {
    ...state.assetFilters,
    ...newFilters
  };
}

// RESET FILTERS
export function resetFilters() {
  state.assetFilters = {
    type: '',
    jobId: '',
    search: '',
    dateRange: {
      from: null,
      to: null
    },
    sortBy: 'date',
    sortDirection: 'desc'
  };
}

// OPEN ASSET VIEWER WITH A SPECIFIC ASSET
export function viewAsset(asset) {
  state.selectedAsset = asset;
  state.assetViewerOpen = true;
}

// CLOSE ASSET VIEWER
export function closeAssetViewer() {
  state.assetViewerOpen = false;
}
