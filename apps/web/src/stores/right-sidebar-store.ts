import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

type SidebarVariant = "sidebar" | "floating";

export type SlideImage = {
  id: string;
  thumb: string;
};

export type ArtifactItem = {
  id: string;
  filename: string;
  contentType: string; // e.g., "slides", "document", "research"
  mimeType: string; // e.g., "application/vnd.openxmlformats-officedocument.presentationml.presentation", "text/markdown"
  size: number;
  downloadUrl: string;
  createdAt?: string;
  slidesImages?: SlideImage[];
  content?: string; // markdown content for research artifacts
};

export type SearchResultItem = {
  type: "link" | "image" | "text" | "artifact";
  title?: string;
  description?: string;
  url?: string;
  downloadUrl?: string;
  filename?: string;
  requiresAuth?: boolean;
  imageUrl?: string;
  icon?: string;
  content?: string;
  artifact?: ArtifactItem;
};

export type StepWithResults = {
  stepName: string;
  stepTitle: string;
  parentTask?: string;
  parentTaskDescription?: string;
  results: SearchResultItem[];
};

interface RightSidebarState {
  isOpen: boolean;
  variant: SidebarVariant;
  allSteps: StepWithResults[];
  currentStepIndex: number;
  artifacts: ArtifactItem[];
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSidebarVariant: (variant: SidebarVariant) => void;
  setAllSteps: (steps: StepWithResults[]) => void;
  setCurrentStep: (stepIndex: number) => void;
  setArtifacts: (artifacts: ArtifactItem[]) => void;
  addArtifact: (artifact: ArtifactItem) => void;
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
      artifacts: [],
      toggleSidebar: () => set((state) => ({ isOpen: !state.isOpen })),
      setSidebarOpen: (open: boolean) => set({ isOpen: open }),
      setSidebarVariant: (variant: SidebarVariant) => set({ variant }),
      setAllSteps: (steps: StepWithResults[]) =>
        set({ allSteps: steps, isOpen: true, currentStepIndex: 0 }),
      setCurrentStep: (stepIndex: number) =>
        set({ currentStepIndex: stepIndex }),
      setArtifacts: (artifacts: ArtifactItem[]) => set({ artifacts }),
      addArtifact: (artifact: ArtifactItem) =>
        set((state) => {
          // Avoid duplicates
          if (state.artifacts.some((a) => a.id === artifact.id)) {
            return state;
          }
          return { artifacts: [...state.artifacts, artifact] };
        }),
      clearSelection: () => set({ allSteps: [], currentStepIndex: 0, artifacts: [] }),
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
