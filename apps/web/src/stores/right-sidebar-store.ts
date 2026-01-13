import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

type SidebarVariant = "sidebar" | "floating";

export type SearchResultItem = {
  type: "link" | "image" | "text";
  title?: string;
  description?: string;
  url?: string;
  imageUrl?: string;
  icon?: string;
  content?: string;
};

export type StepWithResults = {
  stepName: string;
  stepTitle: string;
  results: SearchResultItem[];
};

interface RightSidebarState {
  isOpen: boolean;
  variant: SidebarVariant;
  allSteps: StepWithResults[];
  currentStepIndex: number;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSidebarVariant: (variant: SidebarVariant) => void;
  setAllSteps: (steps: StepWithResults[]) => void;
  setCurrentStep: (stepIndex: number) => void;
  clearSelection: () => void;
  nextStep: () => void;
  prevStep: () => void;
}

// Computed getter
export const useCurrentStepResults = () => {
  const allSteps = useRightSidebarStore((state) => state.allSteps);
  const currentStepIndex = useRightSidebarStore(
    (state) => state.currentStepIndex,
  );
  return allSteps[currentStepIndex]?.results || [];
};

export const useRightSidebarStore = create<RightSidebarState>()(
  persist(
    (set) => ({
      isOpen: true,
      variant: "sidebar",
      allSteps: [],
      currentStepIndex: 0,
      toggleSidebar: () => set((state) => ({ isOpen: !state.isOpen })),
      setSidebarOpen: (open: boolean) => set({ isOpen: open }),
      setSidebarVariant: (variant: SidebarVariant) => set({ variant }),
      setAllSteps: (steps: StepWithResults[]) =>
        set({ allSteps: steps, isOpen: true, currentStepIndex: 0 }),
      setCurrentStep: (stepIndex: number) =>
        set({ currentStepIndex: stepIndex }),
      clearSelection: () => set({ allSteps: [], currentStepIndex: 0 }),
      nextStep: () =>
        set((state) => ({
          currentStepIndex:
            state.currentStepIndex < state.allSteps.length - 1
              ? state.currentStepIndex + 1
              : state.currentStepIndex,
        })),
      prevStep: () =>
        set((state) => ({
          currentStepIndex:
            state.currentStepIndex > 0
              ? state.currentStepIndex - 1
              : state.currentStepIndex,
        })),
    }),
    {
      name: "right-sidebar-storage",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);
