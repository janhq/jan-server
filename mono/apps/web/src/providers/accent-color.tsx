/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useState } from "react";
import { useTheme } from "@/components/themes/theme-provider";
import { LOCAL_STORAGE_KEY } from "@/constants";
import { profileService } from "@/services/profile-service";
import { useAuth } from "@/stores/auth-store";

export const THEME_COLORS = [
  {
    name: "Gray",
    value: "gray",
    thumb: "#3F3F46",
    primary: "#f17455",
    sidebar: {
      light: "#fafafa",
      dark: "#171717",
    },
  },
  {
    name: "Red",
    value: "red",
    thumb: "#F0614B",
    primary: "#F0614B",
    sidebar: {
      light: "#F3CBC4",
      dark: "#5E1308",
    },
  },
  {
    name: "Orange",
    value: "orange",
    thumb: "#E9A23F",
    primary: "#E9A23F",
    sidebar: {
      light: "#F3DFC4",
      dark: "#5C3A0A",
    },
  },
  {
    name: "Green",
    value: "green",
    thumb: "#88BA42",
    primary: "#88BA42",
    sidebar: {
      light: "#DFF3C4",
      dark: "#374B1B",
    },
  },
  {
    name: "Emerald",
    value: "emerald",
    thumb: "#38AB51",
    primary: "#38AB51",
    sidebar: {
      light: "#C4F3CE",
      dark: "#194D24",
    },
  },
  {
    name: "Teal",
    value: "teal",
    thumb: "#38AB8D",
    primary: "#38AB8D",
    sidebar: {
      light: "#C4F3E6",
      dark: "#194D3F",
    },
  },
  {
    name: "Cyan",
    value: "cyan",
    thumb: "#45BBDE",
    primary: "#45BBDE",
    sidebar: {
      light: "#C4E8F3",
      dark: "#0F4657",
    },
  },
  {
    name: "Blue",
    value: "blue",
    thumb: "#456BDE",
    primary: "#456BDE",
    sidebar: {
      light: "#C4D0F3",
      dark: "#0F2157",
    },
  },
  {
    name: "Purple",
    value: "purple",
    thumb: "#865EEA",
    primary: "#865EEA",
    sidebar: {
      light: "#D2C4F3",
      dark: "#220C5A",
    },
  },
  {
    name: "Pink",
    value: "pink",
    thumb: "#D55EF3",
    primary: "#D55EF3",
    sidebar: {
      light: "#E9C4F3",
      dark: "#4D075F",
    },
  },
  {
    name: "Rose",
    value: "rose",
    thumb: "#F655B8",
    primary: "#F655B8",
    sidebar: {
      light: "#F3C4E1",
      dark: "#61053E",
    },
  },
] as const;

const DEFAULT_COLOR = "gray";

type AccentColorContextType = {
  accentColor: string;
  setAccentColor: (color: string) => void;
  availableColors: typeof THEME_COLORS;
  isLoading: boolean;
};

const AccentColorContext = createContext<AccentColorContextType | undefined>(
  undefined,
);

export function AccentColorProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { theme } = useTheme();
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const [accentColor, setAccentColorState] = useState<string>(() => {
    if (typeof window === "undefined") return DEFAULT_COLOR;
    return (
      localStorage.getItem(LOCAL_STORAGE_KEY.ACCENT_COLOR) || DEFAULT_COLOR
    );
  });
  const [isLoading, setIsLoading] = useState(true);

  // Fetch accent color from API on mount (only if authenticated)
  useEffect(() => {
    const fetchAccentColorFromAPI = async () => {
      setIsLoading(true);

      if (!isAuthenticated || isGuest) {
        // Fallback to default when not authenticated or guest
        setAccentColorState(DEFAULT_COLOR);
        localStorage.setItem(LOCAL_STORAGE_KEY.ACCENT_COLOR, DEFAULT_COLOR);
        setIsLoading(false);
        return;
      }

      try {
        const response = await profileService.getPreferences();
        const apiThemeColor = response.preferences?.theme_color;

        if (
          apiThemeColor &&
          THEME_COLORS.find((c) => c.value === apiThemeColor)
        ) {
          // User has a saved theme color preference
          setAccentColorState(apiThemeColor);
          localStorage.setItem(LOCAL_STORAGE_KEY.ACCENT_COLOR, apiThemeColor);
        } else {
          // No theme color in preferences, use default
          setAccentColorState(DEFAULT_COLOR);
          localStorage.setItem(LOCAL_STORAGE_KEY.ACCENT_COLOR, DEFAULT_COLOR);
        }
      } catch (error) {
        console.error("Failed to fetch accent color from API:", error);
        // On error, keep current localStorage value or use default
        const storedColor = localStorage.getItem(
          LOCAL_STORAGE_KEY.ACCENT_COLOR,
        );
        if (storedColor && THEME_COLORS.find((c) => c.value === storedColor)) {
          setAccentColorState(storedColor);
        } else {
          setAccentColorState(DEFAULT_COLOR);
          localStorage.setItem(LOCAL_STORAGE_KEY.ACCENT_COLOR, DEFAULT_COLOR);
        }
      } finally {
        setIsLoading(false);
      }
    };

    fetchAccentColorFromAPI();
  }, [isAuthenticated, isGuest]);

  useEffect(() => {
    const selectedColor = THEME_COLORS.find((c) => c.value === accentColor);
    if (!selectedColor) return;

    const root = document.documentElement;

    const updateColors = () => {
      // Determine if we're in dark mode
      const isDark =
        theme === "dark" ||
        (theme === "system" &&
          window.matchMedia("(prefers-color-scheme: dark)").matches);

      // Update CSS variables with hex colors
      const sidebarColor =
        typeof selectedColor.sidebar === "string"
          ? selectedColor.sidebar
          : isDark
            ? selectedColor.sidebar.dark
            : selectedColor.sidebar.light;

      // Use requestAnimationFrame to ensure this runs after theme class changes
      requestAnimationFrame(() => {
        root.style.setProperty("--sidebar", sidebarColor);
        root.style.setProperty("--primary", selectedColor.primary);
      });
    };

    // Initial update
    updateColors();

    // Listen for system theme changes when in system mode
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      if (theme === "system") {
        updateColors();
      }
    };

    mediaQuery.addEventListener("change", handleChange);

    // Store in localStorage
    localStorage.setItem(LOCAL_STORAGE_KEY.ACCENT_COLOR, accentColor);

    return () => {
      mediaQuery.removeEventListener("change", handleChange);
    };
  }, [accentColor, theme]);

  const setAccentColor = (color: string) => {
    const colorExists = THEME_COLORS.find((c) => c.value === color);
    if (colorExists) {
      setAccentColorState(color);
    }
  };

  const value = {
    accentColor,
    setAccentColor,
    availableColors: THEME_COLORS,
    isLoading,
  };

  return (
    <AccentColorContext.Provider value={value}>
      {children}
    </AccentColorContext.Provider>
  );
}

export function useAccentColor() {
  const context = useContext(AccentColorContext);
  if (context === undefined) {
    throw new Error("useAccentColor must be used within AccentColorProvider");
  }
  return context;
}
