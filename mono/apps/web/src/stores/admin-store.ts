import { create } from "zustand";
import { persist } from "zustand/middleware";
import { adminService } from "@/services/admin-service";

interface AdminState {
  isAdmin: boolean | null;
  isLoading: boolean;
  lastChecked: number | null;
  checkAdminStatus: (forceCheck?: boolean) => Promise<boolean>;
  clearAdminStatus: () => void;
}

// Cache admin status for 1 minute (reduced from 5 minutes for security)
const CACHE_DURATION = 60 * 1000;

export const useAdminStore = create<AdminState>()(
  persist(
    (set, get) => ({
      isAdmin: null,
      isLoading: false,
      lastChecked: null,

      checkAdminStatus: async (forceCheck = false) => {
        const { isAdmin, lastChecked, isLoading } = get();

        // If already loading, don't start another check
        if (isLoading) {
          return isAdmin ?? false;
        }

        // If we have a cached value and it's still valid, return it (unless forceCheck)
        if (
          !forceCheck &&
          isAdmin !== null &&
          lastChecked &&
          Date.now() - lastChecked < CACHE_DURATION
        ) {
          return isAdmin;
        }

        // Start loading
        set({ isLoading: true });

        try {
          const result = await adminService.checkIsAdmin();
          set({
            isAdmin: result,
            isLoading: false,
            lastChecked: Date.now(),
          });
          return result;
        } catch (error) {
          console.error("Failed to check admin status:", error);
          set({
            isAdmin: false,
            isLoading: false,
            lastChecked: Date.now(),
          });
          return false;
        }
      },

      clearAdminStatus: () => {
        set({
          isAdmin: null,
          isLoading: false,
          lastChecked: null,
        });
      },
    }),
    {
      name: "admin-storage",
      partialize: (state) => ({
        isAdmin: state.isAdmin,
        lastChecked: state.lastChecked,
      }),
    },
  ),
);
