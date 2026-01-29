import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

type SidebarVariant = "sidebar" | "floating";

interface SidebarState {
  isOpen: boolean;
  variant: SidebarVariant;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSidebarVariant: (variant: SidebarVariant) => void;
}

export const useSidebarStore = create<SidebarState>()(
  persist(
    (set) => ({
      isOpen: true,
      variant: "sidebar",
      toggleSidebar: () => set((state) => ({ isOpen: !state.isOpen })),
      setSidebarOpen: (open: boolean) => set({ isOpen: open }),
      setSidebarVariant: (variant: SidebarVariant) => set({ variant }),
    }),
    {
      name: "sidebar-storage",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);
