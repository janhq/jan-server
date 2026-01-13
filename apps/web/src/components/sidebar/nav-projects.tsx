import { useState, useRef, useEffect, useCallback } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import { MoreHorizontal, Loader, Trash2 } from "lucide-react";

import {
  SidebarGroup,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  useSidebar,
} from "@/components/sidebar/sidebar";
import {
  DropDrawer,
  DropDrawerContent,
  DropDrawerItem,
  DropDrawerLabel,
  DropDrawerTrigger,
} from "@janhq/interfaces/dropdrawer";
import { useIsMobile } from "@janhq/interfaces/hooks/use-mobile";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";
import { Button } from "@janhq/interfaces/button";

import { useProjects } from "@/stores/projects-store";
import {
  AnimatedGroupLabel,
  AnimatedProjectItem,
} from "@/components/sidebar/items";
import { QUERY_LIMIT } from "@/constants";

export function NavProjects({ startIndex = 3 }: { startIndex?: number }) {
  const { isMobile: isSidebarMobile } = useSidebar();
  const isMobileDevice = useIsMobile();
  const navigate = useNavigate();
  const params = useParams({ strict: false }) as { projectId?: string };
  const projects = useProjects((state) => state.projects);
  const projectsCursor = useProjects((state) => state.projectsCursor);
  const loadMoreProjects = useProjects((state) => state.loadMoreProjects);
  const deleteProject = useProjects((state) => state.deleteProject);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [itemToDelete, setItemToDelete] = useState<Project | null>(null);
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const bottomSentinelRef = useRef<HTMLDivElement>(null);

  const visibleProjects = projects.slice(0, QUERY_LIMIT.PROJECTS_SIDEBAR);
  const remainingProjects = projects.slice(QUERY_LIMIT.PROJECTS_SIDEBAR);
  const hasMoreProjects = projects.length > QUERY_LIMIT.PROJECTS_SIDEBAR;

  const handleDeleteClick = (item: Project) => {
    setItemToDelete(item);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!itemToDelete) return;

    try {
      await deleteProject(itemToDelete.id);
      setDeleteDialogOpen(false);
      setItemToDelete(null);
      if (params.projectId === itemToDelete.id) {
        navigate({ to: "/" });
      }
    } catch (error) {
      console.error("Failed to delete project:", error);
    }
  };

  const handleProjectClick = (projectId: string) => {
    navigate({ to: "/projects/$projectId", params: { projectId } });
    setDropdownOpen(false);
  };

  const handleLoadMore = useCallback(async () => {
    if (loadingMore || !projectsCursor) return;
    setLoadingMore(true);
    try {
      await loadMoreProjects();
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, projectsCursor, loadMoreProjects]);

  useEffect(() => {
    if (!dropdownOpen) return;
    const sentinel = bottomSentinelRef.current;
    const scrollContainer = scrollContainerRef.current;
    if (!sentinel || !scrollContainer) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          handleLoadMore();
        }
      },
      { root: scrollContainer, threshold: 0.1, rootMargin: "50px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [dropdownOpen, handleLoadMore]);

  if (projects.length === 0) {
    return null;
  }

  return (
    <>
      <SidebarGroup className="group-data-[collapsible=icon]:hidden">
        <AnimatedGroupLabel index={startIndex}>Projects</AnimatedGroupLabel>
        <SidebarMenu>
          {visibleProjects.map((item, idx) => (
            <AnimatedProjectItem
              key={item.id}
              item={item}
              isActive={params.projectId === item.id}
              isMobile={isSidebarMobile}
              onDeleteClick={handleDeleteClick}
              index={startIndex + 1 + idx}
            />
          ))}
          {hasMoreProjects && (
            <SidebarMenuItem>
              <DropDrawer open={dropdownOpen} onOpenChange={setDropdownOpen}>
                <DropDrawerTrigger asChild>
                  <SidebarMenuButton className="text-muted-foreground">
                    <MoreHorizontal className="h-4 w-4" />
                    <span>See more</span>
                  </SidebarMenuButton>
                </DropDrawerTrigger>
                <DropDrawerContent
                  className={isMobileDevice ? "p-0" : "w-56 p-0"}
                  side="right"
                  align="start"
                  sideOffset={4}
                >
                  {isMobileDevice && (
                    <DropDrawerLabel>More Projects</DropDrawerLabel>
                  )}
                  <div
                    ref={scrollContainerRef}
                    className={
                      isMobileDevice
                        ? "overflow-y-auto"
                        : "max-h-64 overflow-y-auto"
                    }
                  >
                    {remainingProjects.map((project) => (
                      <DropDrawerItem
                        key={project.id}
                        className="group flex items-center justify-between cursor-pointer"
                        onClick={() => handleProjectClick(project.id)}
                      >
                        <span className="truncate text-sm flex-1">
                          {project.name}
                        </span>
                        <button
                          type="button"
                          className={
                            isMobileDevice
                              ? "ml-2"
                              : "opacity-0 group-hover:opacity-100 transition-opacity ml-2"
                          }
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteClick(project);
                          }}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </button>
                      </DropDrawerItem>
                    ))}
                    {loadingMore && (
                      <div className="flex justify-center py-2">
                        <Loader className="h-4 w-4 animate-spin text-muted-foreground" />
                      </div>
                    )}
                    <div ref={bottomSentinelRef} className="h-1" />
                  </div>
                </DropDrawerContent>
              </DropDrawer>
            </SidebarMenuItem>
          )}
        </SidebarMenu>
      </SidebarGroup>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-106.25" showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Delete Project</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{" "}
              <span className="font-semibold">
                &quot;{itemToDelete?.name}&quot;?
              </span>
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline" className="rounded-full">
                Cancel
              </Button>
            </DialogClose>
            <Button
              className="rounded-full"
              variant="destructive"
              onClick={handleConfirmDelete}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
