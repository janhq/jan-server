import { useArtifactGalleryStore, type ContentTypeFilter } from "@/stores/artifact-gallery-store";
import { cn } from "@/lib/utils";

const CONTENT_TYPE_FILTERS: Array<{ value: ContentTypeFilter; label: string }> = [
  { value: "all", label: "All" },
  { value: "slides", label: "Presentations" },
  { value: "document", label: "Documents" },
  { value: "research", label: "Research" },
  { value: "code", label: "Code" },
  { value: "image", label: "Images" },
  { value: "html", label: "HTML" },
  { value: "markdown", label: "Markdown" },
];

export function ArtifactFilterTabs() {
  const { contentTypeFilter, setContentTypeFilter } = useArtifactGalleryStore();

  return (
    <div className="border-b px-6">
      <div className="flex gap-1 overflow-x-auto">
        {CONTENT_TYPE_FILTERS.map((filter) => (
          <button
            key={filter.value}
            onClick={() => setContentTypeFilter(filter.value)}
            className={cn(
              "px-4 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap",
              contentTypeFilter === filter.value
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
            )}
          >
            {filter.label}
          </button>
        ))}
      </div>
    </div>
  );
}
