import { fetchJsonWithAuth } from "@/lib/api-client";
import { QUERY_LIMIT } from "@/constants";

declare const JAN_API_BASE_URL: string;

export const projectService = {
  /**
   * Get all projects
   */
  getProjects: async (
    limit?: number,
    after?: string,
  ): Promise<ProjectsResponse> => {
    const params = new URLSearchParams();
    if (limit) {
      params.set("limit", String(limit));
    }
    if (after) {
      params.set("after", after);
    }
    const queryString = params.toString();
    return fetchJsonWithAuth<ProjectsResponse>(
      `${JAN_API_BASE_URL}v1/projects${queryString ? `?${queryString}` : ""}`,
    );
  },

  /**
   * Get a single project by ID
   */
  loadMoreProjects: async (after?: string): Promise<ProjectsResponse> => {
    return projectService.getProjects(QUERY_LIMIT.PROJECTS_PAGINATION, after);
  },

  getProject: async (projectId: string): Promise<Project> => {
    return fetchJsonWithAuth<Project>(
      `${JAN_API_BASE_URL}v1/projects/${projectId}`,
    );
  },

  /**
   * Create a new project
   */
  createProject: async (data: CreateProjectRequest): Promise<Project> => {
    return fetchJsonWithAuth<Project>(`${JAN_API_BASE_URL}v1/projects`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  /**
   * Update an existing project
   */
  updateProject: async (
    projectId: string,
    data: UpdateProjectRequest,
  ): Promise<Project> => {
    return fetchJsonWithAuth<Project>(
      `${JAN_API_BASE_URL}v1/projects/${projectId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
  },

  /**
   * Delete a project (soft delete)
   */
  deleteProject: async (projectId: string): Promise<DeleteProjectResponse> => {
    return fetchJsonWithAuth<DeleteProjectResponse>(
      `${JAN_API_BASE_URL}v1/projects/${projectId}`,
      {
        method: "DELETE",
      },
    );
  },
};
