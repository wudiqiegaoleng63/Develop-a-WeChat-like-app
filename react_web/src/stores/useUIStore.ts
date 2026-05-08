import { create } from 'zustand'

interface UIState {
  sidebarCollapsed: boolean
  activeTab: 'all' | 'contacts' | 'groups'
  searchQuery: string
  toggleSidebar: () => void
  setActiveTab: (tab: 'all' | 'contacts' | 'groups') => void
  setSearchQuery: (q: string) => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  activeTab: 'all',
  searchQuery: '',
  toggleSidebar: () => set(state => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setActiveTab: (tab) => set({ activeTab: tab }),
  setSearchQuery: (q) => set({ searchQuery: q }),
}))
