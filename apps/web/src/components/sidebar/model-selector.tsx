import { useEffect, useState } from "react";
import { Check, ChevronsUpDown, Box } from "lucide-react";
import {
  DropDrawer,
  DropDrawerContent,
  DropDrawerItem,
  DropDrawerTrigger,
} from "@janhq/interfaces/dropdrawer";
import { Button } from "@janhq/interfaces/button";
import { Jan } from "@janhq/interfaces/svgs/jan";
import { useModels } from "@/stores/models-store";
import { useProfile } from "@/stores/profile-store";
import { useAnimationStore } from "@/stores/animation-store";
import { useAuth } from "@/stores/auth-store";
import { analytics } from "@/lib/analytics";
import { cn } from "@/lib/utils";

export function ModelSelector() {
  const [open, setOpen] = useState(false);
  const [isReady, setIsReady] = useState(false);
  const models = useModels((state) => state.models);
  const fetchPreferences = useProfile((state) => state.fetchPreferences);
  const updatePreferences = useProfile((state) => state.updatePreferences);
  const getModels = useModels((state) => state.getModels);
  const selectedModel = useModels((state) => state.selectedModel);
  const setSelectedModel = useModels((state) => state.setSelectedModel);
  const loading = useModels((state) => state.loading);

  const modelSelectorAnimated = useAnimationStore(
    (state) => state.modelSelectorAnimated,
  );
  const setModelSelectorAnimated = useAnimationStore(
    (state) => state.setModelSelectorAnimated,
  );
  const [shouldAnimate] = useState(() => !modelSelectorAnimated);

  useEffect(() => {
    const initialize = async () => {
      await getModels();
      try {
        const preferences = await fetchPreferences();
        const selectedModelId = preferences?.preferences.selected_model;
        // Get fresh models from store (not stale closure value)
        const freshModels = useModels.getState().models;

        if (selectedModelId) {
          const model = freshModels.find((m) => m.id === selectedModelId);
          if (model) {
            setSelectedModel(model);
          } else if (freshModels.length > 0) {
            // Saved model no longer exists, fall back to first model
            setSelectedModel(freshModels[0]);
          }
        } else if (freshModels.length > 0) {
          // No saved preference, auto-select the first model (which is sorted by order)
          setSelectedModel(freshModels[0]);
        }
      } catch (error) {
        console.error("Failed to fetch preferences:", error);
        // On preference fetch error, still try to select first model
        const freshModels = useModels.getState().models;
        if (freshModels.length > 0) {
          setSelectedModel(freshModels[0]);
        }
      }
      setIsReady(true);
      if (!modelSelectorAnimated) {
        setModelSelectorAnimated();
      }
    };
    initialize();
  }, [
    fetchPreferences,
    getModels,
    setSelectedModel,
    modelSelectorAnimated,
    setModelSelectorAnimated,
  ]);

  const isAuthenticated = useAuth((state) => state.isAuthenticated);

  const handleSelectModel = (model: Model) => {
    const previousModel = selectedModel;

    if (previousModel?.id !== model.id) {
      analytics.capture("model_selected", {
        model: model.id,
        previous_model: previousModel?.id || null,
        user_status: analytics.getUserStatus(isAuthenticated),
      });
    }

    setSelectedModel(model);
    updatePreferences({
      preferences: {
        selected_model: model.id,
      },
    });
    setOpen(false);
  };

  // Don't render until initialization is complete to prevent flashing
  if (!isReady) return null;

  return (
    <DropDrawer open={open} onOpenChange={setOpen}>
      <DropDrawerTrigger asChild>
        <div className="relative">
          <Button
            variant="outline"
            className={cn(
              "justify-between rounded-full",
              shouldAnimate && "animate-in fade-in duration-200",
            )}
          >
            <Jan className="size-4 shrink-0" />
            <span
              className={
                selectedModel?.model_display_name
                  ? "truncate"
                  : "truncate text-muted-foreground"
              }
            >
              {selectedModel?.model_display_name || "Select a model"}
            </span>
            <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
          </Button>
        </div>
      </DropDrawerTrigger>
      <DropDrawerContent align="start" className="p-2 md:w-70">
        {!loading && (
          <>
            {models.length === 0 ? (
              <div className="px-4 py-6 text-center text-sm text-muted-foreground">
                No models available
              </div>
            ) : (
              <>
                {models.map((model) => (
                  <DropDrawerItem
                    key={model.id}
                    onClick={() => handleSelectModel(model)}
                    icon={
                      selectedModel?.id === model.id ? (
                        <Check className="size-4" />
                      ) : undefined
                    }
                  >
                    <div className="flex items-start gap-3">
                      <Box className="size-4 shrink-0 text-muted-foreground mt-1" />
                      <div className="flex flex-col items-start gap-0.5">
                        <span className="font-medium">
                          {model.model_display_name}
                        </span>
                        {model.owned_by && (
                          <span className="text-xs text-muted-foreground">
                            {model.owned_by}
                          </span>
                        )}
                      </div>
                    </div>
                  </DropDrawerItem>
                ))}
              </>
            )}
          </>
        )}
      </DropDrawerContent>
    </DropDrawer>
  );
}
