import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'FinOps Enterprise Platform',
  description: 'Modern FinOps Enterprise Platform for multi-cloud cost management',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="pt-BR" className="dark">
      <body className="h-screen overflow-hidden">
        {children}
      </body>
    </html>
  );
}
