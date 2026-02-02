import '@/app/global.css';
import { AuthProvider } from '@/components/auth-provider';
import { SiteHeader } from '@/components/site-header';
import { RootProvider } from 'fumadocs-ui/provider/next';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Jan Platform',
  description: 'Jan Platform Documentation',
  icons: {
    icon: '/favicon.ico',
    shortcut: '/favicon.ico',
    apple: '/favicon.ico',
  },
};

export default function Layout({ children }: LayoutProps<'/'>) {
  return (
    <html lang="en" className="font-sans" suppressHydrationWarning>
      <body className="flex flex-col min-h-screen" suppressHydrationWarning>
        <AuthProvider>
          <div className="relative flex min-h-screen flex-col">
            <SiteHeader />
            <div className="flex-1">
              <RootProvider>{children}</RootProvider>
            </div>
          </div>
        </AuthProvider>
      </body>
    </html>
  );
}
