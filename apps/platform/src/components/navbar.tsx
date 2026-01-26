'use client';

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Skeleton } from '@/components/ui/skeleton';
import { organizations, projects } from '@/lib/dummy';
import { useAuthStore } from '@/store/auth-store';

import { ThemeToggle } from '@/components/theme-toggle';
import { Check, ChevronsUpDown, LogOut, Settings } from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';

export function Navbar() {
  const isLoggedIn = useAuthStore((state) => state.isLoggedIn);
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);

  const [selectedOrg, setSelectedOrg] = useState(organizations[0]);
  const [selectedProject, setSelectedProject] = useState(projects[0]);
  const [mounted, setMounted] = useState(false);
  const pathname = usePathname();

  useEffect(() => {
    setMounted(true);
  }, []);

  const isActive = (path: string) => {
    if (!mounted) return false;

    if (path === '/docs/api-reference') {
      return pathname?.startsWith('/docs/api-reference');
    }
    if (path === '/docs') {
      return (
        pathname === '/docs' ||
        (pathname?.startsWith('/docs') && !pathname?.startsWith('/docs/api-reference'))
      );
    }
    return pathname === path;
  };

  // Prevent hydration mismatch by not rendering auth-dependent content until mounted
  if (!mounted) {
    return (
      <header className="fixed inset-0 z-50 w-full bg-background h-14 border-b border-border">
        <div className="flex h-14 items-center justify-between px-6">
          <div className="flex items-center gap-2">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-6 w-32" />
          </div>
          <nav className="flex items-center gap-4">
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-4 w-30" />
            <div className="ml-2">
              <Skeleton className="h-8 w-16 rounded-md" />
            </div>
          </nav>
        </div>
      </header>
    );
  }

  return (
    <header className="fixed inset-0 z-50 w-full bg-background h-14 border-b border-border">
      <div className="flex h-14 items-center justify-between px-6">
        {/* Left side - Logo & Project */}
        <div className="flex items-center gap-2">
          {isLoggedIn ? (
            <>
              {/* Organization Dropdown */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="gap-2 max-w-[200px]">
                    <Avatar className="size-5 shrink-0">
                      <AvatarFallback className="text-xs bg-foreground text-background uppercase font-medium">
                        {selectedOrg.name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <span className="truncate text-foreground/90">{selectedOrg.name}</span>
                    <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  <DropdownMenuLabel className="text-xs text-muted-foreground uppercase">
                    Organization
                  </DropdownMenuLabel>
                  {organizations.map((org) => (
                    <DropdownMenuItem
                      key={org.id}
                      onClick={() => setSelectedOrg(org)}
                      className="flex items-center gap-2 cursor-pointer"
                    >
                      <Avatar className="size-5 shrink-0">
                        <AvatarFallback className="text-xs bg-foreground text-background uppercase font-medium">
                          {org.name.charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <span className="flex-1 truncate text-foreground/90">{org.name}</span>
                      {selectedOrg.id === org.id && <Check className="size-4 shrink-0" />}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>

              {/* Separator */}
              <div className="h-6 w-px bg-border" />

              {/* Project Dropdown */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="gap-2 max-w-[200px]">
                    <span className="truncate text-foreground/90">{selectedProject.name}</span>
                    <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  <DropdownMenuLabel className="text-xs text-muted-foreground uppercase">
                    Project
                  </DropdownMenuLabel>
                  {projects.map((project) => (
                    <DropdownMenuItem
                      key={project.id}
                      onClick={() => setSelectedProject(project)}
                      className="flex items-center gap-2 cursor-pointer"
                    >
                      <span className="flex-1 truncate text-foreground/90">{project.name}</span>
                      {selectedProject.id === project.id && <Check className="size-4 shrink-0" />}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            <Link href="/" className="text-lg font-semibold">
              Menlo Platform
            </Link>
          )}
        </div>

        {/* Right side - Navigation & User Controls */}
        <nav className="flex items-center gap-4">
          {/* Always visible navigation */}
          <Link
            href="/docs"
            className={`text-sm font-medium transition-colors ${
              isActive('/docs')
                ? 'text-black dark:text-white'
                : 'text-gray-500 dark:text-gray-400 hover:text-black dark:hover:text-white'
            }`}
          >
            Docs
          </Link>
          <Link
            href="/docs/api-reference"
            className={`text-sm font-medium transition-colors ${
              isActive('/docs/api-reference')
                ? 'text-black dark:text-white'
                : 'text-gray-500 dark:text-gray-400 hover:text-black dark:hover:text-white'
            }`}
          >
            API Reference
          </Link>

          <div className="ml-2 flex items-center gap-2">
            <ThemeToggle />
            {isLoggedIn ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon-sm" className="rounded-lg">
                    <Avatar className="size-6">
                      <AvatarImage alt={user?.name || 'User'} />
                      <AvatarFallback className="rounded-lg bg-orange-600 uppercase font-medium">
                        {user?.name?.charAt(0) || user?.email?.charAt(0) || 'U'}
                      </AvatarFallback>
                    </Avatar>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex flex-col space-y-1">
                      <p className="text-sm font-medium leading-none">
                        {user?.name || 'Full Name'}
                      </p>
                      <p className="text-xs leading-none text-muted-foreground">{user?.email}</p>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild>
                    <Link href="/docs" className="cursor-pointer">
                      <Settings className="mr-1 h-4 w-4" />
                      Settings
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={logout} className="cursor-pointer">
                    <LogOut className="mr-1 h-4 w-4" />
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => {
                  import('@/lib/auth/service').then(({ getSharedAuthService }) => {
                    getSharedAuthService().loginWithProvider('keycloak').catch(console.error);
                  });
                }}
              >
                Login
              </Button>
            )}
          </div>
        </nav>
      </div>
    </header>
  );
}
