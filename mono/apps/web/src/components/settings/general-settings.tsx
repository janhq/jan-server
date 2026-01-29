import { Avatar, AvatarFallback, AvatarImage } from "@janhq/interfaces/avatar";
import {
  DropDrawer,
  DropDrawerContent,
  DropDrawerItem,
  DropDrawerTrigger,
} from "@janhq/interfaces/dropdrawer";
import { useAuth } from "@/stores/auth-store";
import { getInitialsAvatar, cn } from "@/lib/utils";
import { useTheme } from "@/components/themes/theme-provider";
import { ChevronsUpDown, CircleCheck, Monitor, Moon, Sun } from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { useRef } from "react";
import { THEME } from "@/constants";
import { Separator } from "@janhq/interfaces/ui/separator";
import { useAccentColor } from "@/providers/accent-color";
import { profileService } from "@/services/profile-service";
// import { useSidebarStore } from "@/stores/sidebar-store";

export function GeneralSettings() {
  const user = useAuth((state) => state.user);
  const { theme, setTheme } = useTheme();
  const { accentColor, setAccentColor, availableColors } = useAccentColor();
  // const variant = useSidebarStore((state) => state.variant);
  // const setSidebarVariant = useSidebarStore((state) => state.setSidebarVariant);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleAccentColorChange = async (color: string) => {
    setAccentColor(color);

    // Find the selected color config
    const selectedColor = availableColors.find((c) => c.value === color);
    if (!selectedColor) return;

    // Determine current theme mode
    const isDark =
      theme === "dark" ||
      (theme === "system" &&
        window.matchMedia("(prefers-color-scheme: dark)").matches);

    // Get sidebar color based on theme
    const sidebarColor =
      typeof selectedColor.sidebar === "string"
        ? selectedColor.sidebar
        : isDark
          ? selectedColor.sidebar.dark
          : selectedColor.sidebar.light;

    try {
      await profileService.updatePreferences({
        preferences: {
          theme_color: color,
          theme_config: {
            name: selectedColor.name,
            value: selectedColor.value,
            thumb: selectedColor.thumb,
            primary: selectedColor.primary,
            sidebar: sidebarColor,
          },
        },
      });
    } catch (error) {
      console.error("Failed to save accent color to API:", error);
    }
  };

  const handleThemeChange = (newTheme: "light" | "dark" | "system") => {
    if (!buttonRef.current || !document.startViewTransition) {
      setTheme(newTheme);
      return;
    }

    const { top, left, width, height } =
      buttonRef.current.getBoundingClientRect();
    const x = left + width / 2;
    const y = top + height / 2;
    const endRadius = Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    );

    const transition = document.startViewTransition(() => {
      setTheme(newTheme);
    });

    transition.ready.then(() => {
      const clipPath = [
        `circle(0px at ${x}px ${y}px)`,
        `circle(${endRadius}px at ${x}px ${y}px)`,
      ];

      document.documentElement.animate(
        {
          clipPath: clipPath,
        },
        {
          duration: 500,
          easing: "ease-in-out",
          pseudoElement: "::view-transition-new(root)",
        },
      );
    });
  };

  const getThemeDisplay = () => {
    switch (theme) {
      case THEME.LIGHT:
        return "Light";
      case THEME.DARK:
        return "Dark";
      case THEME.SYSTEM:
        return "System";
      default:
        return "System";
    }
  };

  const getThemeIcon = () => {
    switch (theme) {
      case THEME.LIGHT:
        return <Sun className="size-4 text-muted-foreground" />;
      case THEME.DARK:
        return <Moon className="size-4 text-muted-foreground" />;
      case THEME.SYSTEM:
        return <Monitor className="size-4 text-muted-foreground" />;
      default:
        return <Monitor className="size-4 text-muted-foreground" />;
    }
  };

  return (
    <div>
      <p className="text-base font-semibold mb-4 font-studio">Account</p>
      {/* Profile Section */}
      <div className="flex items-center gap-4 mb-6 bg-muted/50 p-4 rounded-lg">
        <Avatar className="size-12">
          <AvatarImage src={user?.avatar} alt={user?.name} />
          <AvatarFallback className="bg-primary text-background text-xl font-semibold">
            {getInitialsAvatar(user?.name || "")}
          </AvatarFallback>
        </Avatar>
        <div className="flex-1 ">
          <h4 className="font-medium text-base font-studio">{user?.name}</h4>
          <p className="text-xs text-muted-foreground">{user?.email}</p>
        </div>
      </div>

      {/* Appearance Section */}
      <p className="text-base font-semibold mb-4 font-studio">Appearance</p>
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="font-medium text-sm">Color mode</p>
        </div>
        <DropDrawer>
          <DropDrawerTrigger asChild>
            <Button
              ref={buttonRef}
              variant="outline"
              className="justify-between rounded-xl min-w-40"
            >
              <div className="flex items-center gap-2">
                {getThemeIcon()}
                <span className="text-sm text-muted-foreground">
                  {getThemeDisplay()}
                </span>
              </div>
              <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
            </Button>
          </DropDrawerTrigger>
          <DropDrawerContent>
            <DropDrawerItem
              onSelect={() => handleThemeChange(THEME.LIGHT)}
              icon={
                theme === THEME.LIGHT ? (
                  <CircleCheck className="size-4 text-primary" />
                ) : null
              }
            >
              <div className="flex items-center gap-2">
                <Sun className="size-4 text-muted-foreground" />
                <span>Light</span>
              </div>
            </DropDrawerItem>
            <DropDrawerItem
              onSelect={() => handleThemeChange(THEME.DARK)}
              icon={
                theme === THEME.DARK ? (
                  <CircleCheck className="size-4 text-primary" />
                ) : null
              }
            >
              <div className="flex items-center gap-2">
                <Moon className="size-4 text-muted-foreground" />
                <span>Dark</span>
              </div>
            </DropDrawerItem>
            <DropDrawerItem
              onSelect={() => handleThemeChange(THEME.SYSTEM)}
              icon={
                theme === THEME.SYSTEM ? (
                  <CircleCheck className="size-4 text-primary" />
                ) : null
              }
            >
              <div className="flex items-center gap-2">
                <Monitor className="size-4 text-muted-foreground" />
                <span>System</span>
              </div>
            </DropDrawerItem>
          </DropDrawerContent>
        </DropDrawer>
      </div>

      <Separator className="my-4" />

      {/* Accent Color Section */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="font-medium text-sm">Accent color</p>
        </div>
        <div className="flex items-center gap-1.5">
          {availableColors.map((color) => (
            <button
              key={color.value}
              onClick={() => handleAccentColorChange(color.value)}
              className={cn(
                "size-4 rounded-full transition-all hover:scale-110",
                accentColor === color.value
                  ? "ring-2 ring-offset-2 ring-offset-background ring-primary"
                  : "hover:ring-2 hover:ring-offset-2 hover:ring-offset-background hover:ring-muted-foreground/50",
              )}
              style={{ backgroundColor: `${color.thumb}` }}
              title={color.name}
              aria-label={`Select ${color.name} primary color`}
            />
          ))}
        </div>
      </div>

      {/* Temporary disable */}
      {/* <Separator className="my-4" /> */}
      {/* Sidebar Variant Section */}
      {/* <div>
        <p className="font-medium text-sm mb-3">Menu sidebar</p>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <button
            type="button"
            onClick={() => setSidebarVariant("sidebar")}
            className={cn(
              "flex items-center gap-3 p-4 rounded-lg border transition-all",
              variant === "sidebar"
                ? "border-primary bg-primary/10"
                : "border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600"
            )}
          >
            <PanelLeft className="w-5 h-5" />
            <span className="font-medium">Default</span>
          </button>
          <button
            type="button"
            onClick={() => setSidebarVariant("floating")}
            className={cn(
              "flex items-center gap-3 p-4 rounded-lg border transition-all",
              variant === "floating"
                ? "border-primary bg-primary/10"
                : "border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600"
            )}
          >
            <RectangleVertical className="w-5 h-5" />
            <span className="font-medium">Floating</span>
          </button>
        </div>
      </div> */}
    </div>
  );
}
