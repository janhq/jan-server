import { useRouter } from "@tanstack/react-router";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";
import {
  Monitor,
  FileText,
  Gamepad2,
  Zap,
  Sparkles,
  ClipboardList,
  PlusCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useConversations } from "@/stores/conversation-store";
import { useModels } from "@/stores/models-store";

interface ArtifactCategoryPickerProps {
  open: boolean;
  onClose: () => void;
}

const ARTIFACT_CATEGORIES = [
  {
    id: "apps-websites",
    label: "Apps and websites",
    icon: Monitor,
    prompt: "I want to create an interactive app or website. Help me build something with HTML, CSS, and JavaScript.",
  },
  {
    id: "documents-templates",
    label: "Documents and templates",
    icon: FileText,
    prompt: "I want to create a document or template. Help me write and format professional content.",
  },
  {
    id: "games",
    label: "Games",
    icon: Gamepad2,
    prompt: "I want to create a game. Help me build an interactive browser-based game.",
  },
  {
    id: "productivity-tools",
    label: "Productivity tools",
    icon: Zap,
    prompt: "I want to create a productivity tool. Help me build something useful for daily tasks.",
  },
  {
    id: "creative-projects",
    label: "Creative projects",
    icon: Sparkles,
    prompt: "I want to create something creative. Help me with an artistic or creative project.",
  },
  {
    id: "quiz-survey",
    label: "Quiz or survey",
    icon: ClipboardList,
    prompt: "I want to create a quiz or survey. Help me build an interactive questionnaire.",
  },
  {
    id: "start-from-scratch",
    label: "Start from scratch",
    icon: PlusCircle,
    prompt: "",
  },
];

export function ArtifactCategoryPicker({ open, onClose }: ArtifactCategoryPickerProps) {
  const router = useRouter();
  const createConversation = useConversations((state) => state.createConversation);
  const selectedModel = useModels((state) => state.selectedModel);

  const handleCategorySelect = async (category: typeof ARTIFACT_CATEGORIES[number]) => {
    try {
      // Create a new conversation with required metadata
      const conversation = await createConversation({
        title: category.id === "start-from-scratch" ? "New Artifact" : category.label,
        metadata: {
          model_id: selectedModel?.id ?? "default",
          model_provider: selectedModel?.owned_by ?? "default",
        },
      });

      onClose();

      // Navigate to the conversation
      // If there's a prompt, we could potentially pass it as state or URL param
      // For now, just navigate to the conversation
      router.navigate({
        to: "/threads/$conversationId",
        params: { conversationId: conversation.id },
      });

      // Note: The prompt could be sent as an initial message using the chat input
      // This would require additional integration with the chat system
    } catch (error) {
      console.error("Failed to create conversation:", error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="text-center text-lg">
            Let's get cooking! Pick an artifact category or start building your idea from scratch.
          </DialogTitle>
        </DialogHeader>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mt-4">
          {ARTIFACT_CATEGORIES.map((category) => {
            const Icon = category.icon;
            return (
              <button
                key={category.id}
                onClick={() => handleCategorySelect(category)}
                className={cn(
                  "flex flex-col items-start p-4 rounded-xl border",
                  "bg-card hover:bg-accent hover:border-primary/50",
                  "transition-all duration-200 text-left",
                  "min-h-[100px]"
                )}
              >
                <span className="text-sm font-medium mb-auto">{category.label}</span>
                <Icon className="size-5 text-muted-foreground mt-2" />
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
