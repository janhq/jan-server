import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { analytics } from "@/lib/analytics";

const trackModeSelected = async (
  mode: string,
  previousMode: string,
  enabled: boolean,
) => {
  if (!enabled) return; // Only track when enabling a mode
  const { useAuth } = await import("@/stores/auth-store");
  const isAuthenticated = useAuth.getState().isAuthenticated;
  analytics.capture("mode_selected", {
    mode,
    previous_mode: previousMode,
    user_status: analytics.getUserStatus(isAuthenticated),
  });
};

// Helper to determine current active mode
const getCurrentMode = (state: {
  searchEnabled: boolean;
  deepResearchEnabled: boolean;
  reasoningEnabled: boolean;
  imageGenerationEnabled: boolean;
  agentModeEnabled: boolean;
}): string => {
  if (state.agentModeEnabled) return "agent";
  if (state.deepResearchEnabled) return "deep_research";
  if (state.searchEnabled) return "search";
  if (state.reasoningEnabled) return "reasoning";
  if (state.imageGenerationEnabled) return "create_image";
  return "normal";
};

interface CapabilitiesState {
  searchEnabled: boolean;
  deepResearchEnabled: boolean;
  browserEnabled: boolean;
  reasoningEnabled: boolean;
  imageGenerationEnabled: boolean;
  agentModeEnabled: boolean;
  agentModeAvailable: boolean;
  setSearchEnabled: (enabled: boolean) => void;
  setDeepResearchEnabled: (enabled: boolean) => void;
  setBrowserEnabled: (enabled: boolean) => void;
  setReasoningEnabled: (enabled: boolean) => void;
  setImageGenerationEnabled: (enabled: boolean) => void;
  setAgentModeEnabled: (enabled: boolean) => void;
  setAgentModeAvailable: (available: boolean) => void;
  toggleSearch: () => void;
  toggleDeepResearch: () => void;
  toggleBrowser: () => void;
  toggleReasoning: () => void;
  toggleImageGeneration: () => void;
  toggleAgentMode: () => void;
  hydrate: (preferences: Partial<Preferences>) => void;
}

export const useCapabilities = create<CapabilitiesState>()(
  persist(
    (set) => ({
      searchEnabled: false,
      browserEnabled: false,
      deepResearchEnabled: false,
      reasoningEnabled: false,
      imageGenerationEnabled: false,
      agentModeEnabled: false,
      agentModeAvailable: true,
      setSearchEnabled: (enabled: boolean) => {
        set({ searchEnabled: enabled });
        updatePreferencesInBackground({ enable_search: enabled });
      },
      setDeepResearchEnabled: (enabled: boolean) => {
        set({ deepResearchEnabled: enabled });
        updatePreferencesInBackground({ enable_deep_research: enabled });
      },
      setBrowserEnabled: (enabled: boolean) => {
        set({ browserEnabled: enabled });
        updatePreferencesInBackground({ enable_browser: enabled });
      },
      setReasoningEnabled: (enabled: boolean) => {
        set({ reasoningEnabled: enabled });
        updatePreferencesInBackground({ enable_thinking: enabled });
      },
      setImageGenerationEnabled: (enabled: boolean) => {
        set({ imageGenerationEnabled: enabled });
      },
      setAgentModeEnabled: (enabled: boolean) => {
        set({ agentModeEnabled: enabled });
        updatePreferencesInBackground({ enable_agent_mode: enabled });
      },
      setAgentModeAvailable: (available: boolean) => {
        set({ agentModeAvailable: available });
      },
      toggleSearch: () =>
        set((state) => {
          const newValue = !state.searchEnabled;
          const previousMode = getCurrentMode(state);
          updatePreferencesInBackground({
            enable_search: newValue,
            enable_image_generation: newValue
              ? false
              : state.imageGenerationEnabled,
            enable_agent_mode: newValue ? false : state.agentModeEnabled,
          });
          trackModeSelected("search", previousMode, newValue);
          return {
            searchEnabled: newValue,
            agentModeEnabled: newValue ? false : state.agentModeEnabled,
          };
        }),
      toggleDeepResearch: () =>
        set((state) => {
          const newValue = !state.deepResearchEnabled;
          const previousMode = getCurrentMode(state);
          updatePreferencesInBackground({
            enable_deep_research: newValue,
            enable_image_generation: newValue
              ? false
              : state.imageGenerationEnabled,
            enable_agent_mode: newValue ? false : state.agentModeEnabled,
          });
          trackModeSelected("deep_research", previousMode, newValue);
          return {
            deepResearchEnabled: newValue,
            agentModeEnabled: newValue ? false : state.agentModeEnabled,
          };
        }),
      toggleBrowser: () =>
        set((state) => {
          const newValue = !state.browserEnabled;
          const previousMode = getCurrentMode(state);
          updatePreferencesInBackground({
            enable_browser: newValue,
            enable_image_generation: newValue
              ? false
              : state.imageGenerationEnabled,
            enable_agent_mode: newValue ? false : state.agentModeEnabled,
          });
          trackModeSelected("browser", previousMode, newValue);
          return {
            browserEnabled: newValue,
            agentModeEnabled: newValue ? false : state.agentModeEnabled,
          };
        }),
      toggleReasoning: () =>
        set((state) => {
          const newValue = !state.reasoningEnabled;
          const previousMode = getCurrentMode(state);
          updatePreferencesInBackground({
            enable_thinking: newValue,
            enable_image_generation: newValue
              ? false
              : state.imageGenerationEnabled,
            enable_agent_mode: newValue ? false : state.agentModeEnabled,
          });
          trackModeSelected("reasoning", previousMode, newValue);
          return {
            reasoningEnabled: newValue,
            agentModeEnabled: newValue ? false : state.agentModeEnabled,
          };
        }),
      toggleImageGeneration: () =>
        set((state) => {
          const newValue = !state.imageGenerationEnabled;
          const previousMode = getCurrentMode(state);
          updatePreferencesInBackground({
            enable_image_generation: newValue,
            enable_agent_mode: newValue ? false : state.agentModeEnabled,
          });
          trackModeSelected("create_image", previousMode, newValue);
          return {
            imageGenerationEnabled: newValue,
            agentModeEnabled: newValue ? false : state.agentModeEnabled,
          };
        }),
      toggleAgentMode: () =>
        set((state) => {
          if (!state.agentModeAvailable) {
            return state;
          }
          const newValue = !state.agentModeEnabled;
          const previousMode = getCurrentMode(state);
          if (newValue) {
            // Disable all other modes when enabling agent mode
            updatePreferencesInBackground({
              enable_agent_mode: newValue,
              enable_search: false,
              enable_deep_research: false,
              enable_browser: false,
              enable_thinking: false,
              enable_image_generation: false,
            });
            trackModeSelected("agent", previousMode, newValue);
            return {
              agentModeEnabled: newValue,
              searchEnabled: false,
              deepResearchEnabled: false,
              browserEnabled: false,
              reasoningEnabled: false,
              imageGenerationEnabled: false,
            };
          }
          updatePreferencesInBackground({ enable_agent_mode: newValue });
          return { agentModeEnabled: newValue };
        }),
      hydrate: (preferences: Partial<Preferences>) =>
        set({
          searchEnabled: preferences.enable_search ?? false,
          browserEnabled: preferences.enable_browser ?? false,
          deepResearchEnabled: preferences.enable_deep_research ?? false,
          reasoningEnabled: preferences.enable_thinking ?? false,
          imageGenerationEnabled: preferences.enable_image_generation ?? false,
          agentModeEnabled: preferences.enable_agent_mode ?? false,
        }),
    }),
    {
      name: "capabilities-storage",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        searchEnabled: state.searchEnabled,
        deepResearchEnabled: state.deepResearchEnabled,
        browserEnabled: state.browserEnabled,
        reasoningEnabled: state.reasoningEnabled,
        imageGenerationEnabled: state.imageGenerationEnabled,
        agentModeEnabled: state.agentModeEnabled,
      }),
    },
  ),
);

let pendingPreferences: Partial<Preferences> = {};
let debounceTimer: ReturnType<typeof setTimeout> | null = null;

// Helper function to update preferences in the background
async function updatePreferencesInBackground(
  preferences: Partial<Preferences>,
) {
  // Merge new preferences into pending
  pendingPreferences = { ...pendingPreferences, ...preferences };

  // Clear existing timer
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }

  // Set new timer
  debounceTimer = setTimeout(async () => {
    try {
      const { useProfile } = await import("./profile-store");
      const preferencesToUpdate = { ...pendingPreferences };
      pendingPreferences = {}; // Reset pending preferences

      await useProfile
        .getState()
        .updatePreferences({ preferences: preferencesToUpdate });
    } catch (error) {
      console.error("Failed to update preferences:", error);
    } finally {
      debounceTimer = null;
    }
  }, 100);
}
