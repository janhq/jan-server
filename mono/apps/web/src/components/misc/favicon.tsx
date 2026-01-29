import { LinkIcon } from "lucide-react";
import { useState } from "react";

export const Favicon = ({ url }: { url: string }) => {
  const [error, setError] = useState(false);
  if (error) {
    return <LinkIcon className="size-4 shrink-0 mt-0.5 text-muted-foreground" />;
  }
  try {
    const domain = new URL(url).hostname;
    const faviconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=32`;

    return (
      <img
        src={faviconUrl}
        alt=""
        className="size-4 shrink-0 mt-0.5 rounded-full"
        onError={() => setError(true)}
      />
    );
  } catch {
    return <LinkIcon className="size-4 shrink-0 mt-0.5 text-muted-foreground" />;
  }
};