/* eslint-disable react-hooks/purity */
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import ChatInput from "@/components/chat-input";
import { usePrivateChat } from "./stores/private-chat-store";
import { useCapabilities } from "./stores/capabilities-store";
import { HatGlassesIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { useMemo, useState, useEffect } from "react";

const newYearPhrases = ["How can I help you today?"];
const agentPhrases = ["What should we build together?"];

function AgentModeHero({ phrase }: { phrase: string }) {
  const [displayedText, setDisplayedText] = useState("");
  const [isTypingComplete, setIsTypingComplete] = useState(false);

  useEffect(() => {
    setDisplayedText("");
    setIsTypingComplete(false);
    let currentIndex = 0;
    const interval = setInterval(() => {
      if (currentIndex < phrase.length) {
        setDisplayedText(phrase.slice(0, currentIndex + 1));
        currentIndex++;
      } else {
        setIsTypingComplete(true);
        clearInterval(interval);
      }
    }, 30);
    return () => clearInterval(interval);
  }, [phrase]);

  return (
    <div className="flex flex-col items-center justify-center mb-6 relative">
      {/* Glow effect behind text */}
      {/* <motion.div
        className="absolute inset-0 blur-3xl"
        initial={{ opacity: 0 }}
        animate={{ opacity: isTypingComplete ? 0.1 : 0 }}
        transition={{ duration: 1 }}
      >
        <div className="w-full h-full bg-linear-to-r from-primary/50 via-primary to-primary/50 rounded-full" />
      </motion.div> */}

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
        className="relative"
      >
        <h2 className="text-2xl font-medium font-studio inline">
          {/* Render each character with staggered animation */}
          {displayedText.split("").map((char, i) => (
            <motion.span
              key={i}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.15, delay: i * 0.02 }}
            >
              {char}
            </motion.span>
          ))}
        </h2>

        {/* Shimmer effect on complete */}
        {isTypingComplete && (
          <motion.div
            className="absolute inset-0 bg-linear-to-r from-transparent via-white/20 to-transparent -skew-x-12 blur-[2px]"
            initial={{ x: "-150%", opacity: 0 }}
            animate={{ x: "250%", opacity: [0, 1, 1, 0] }}
            transition={{
              duration: 2,
              delay: 0.2,
              ease: [0.25, 0.1, 0.25, 1]
            }}
          />
        )}
      </motion.div>

      {/* Subtitle that appears after typing */}
      <motion.p
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: isTypingComplete ? 1 : 0, y: isTypingComplete ? 0 : 10 }}
        transition={{ duration: 0.5, delay: 0.2 }}
        className="text-sm text-muted-foreground mt-2"
      >
        I can browse, search, code, and create slides for you
      </motion.p>
    </div>
  );
}

function AppPageContent() {
  const isPrivateChat = usePrivateChat((state) => state.isPrivateChat);
  const agentModeEnabled = useCapabilities((state) => state.agentModeEnabled);
  const agentModeAvailable = useCapabilities(
    (state) => state.agentModeAvailable,
  );

  const randomPhrase = useMemo(
    () => newYearPhrases[Math.floor(Math.random() * newYearPhrases.length)],
    [],
  );

  const agentPhrase = useMemo(
    () => agentPhrases[Math.floor(Math.random() * agentPhrases.length)],
    [],
  );

  return (
    <>
      <AppSidebar />
      <SidebarInset>
        <NavHeader />
        <div className="flex flex-1 flex-col items-center justify-center h-full gap-4 px-4 py-10 max-w-3xl w-full mx-auto ">
          <div className="mx-auto flex justify-center items-center h-full w-full rounded-xl">
            <div className="w-full text-center -mt-24">
              {isPrivateChat ? (
                <>
                  <div className="flex items-center justify-center mb-3">
                    <motion.div
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        duration: 0.3,
                        delay: 0,
                        ease: [0, 0.71, 0.2, 1.01],
                      }}
                      className="bg-foreground size-10 rounded-full flex items-center justify-center"
                    >
                      <HatGlassesIcon className="text-background/80 size-6" />
                    </motion.div>
                    <motion.div
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        duration: 0.3,
                        delay: 0.1,
                        ease: [0, 0.71, 0.2, 1.01],
                      }}
                      className="bg-foreground h-10 px-4 w-auto rounded-full flex items-center justify-center"
                    >
                      <span className="text-background/80 text-2xl font-medium">
                        private
                      </span>
                    </motion.div>
                    <motion.div
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        duration: 0.3,
                        delay: 0.2,
                        ease: [0, 0.71, 0.2, 1.01],
                      }}
                      className="bg-foreground h-10 px-4 w-auto rounded-full flex items-center justify-center"
                    >
                      <span className="text-background/80 text-2xl font-medium">
                        chat
                      </span>
                    </motion.div>
                    <motion.div
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        duration: 0.3,
                        delay: 0.3,
                        ease: [0, 0.71, 0.2, 1.01],
                      }}
                      className="bg-foreground h-10 px-4 w-auto rounded-full flex items-center justify-center"
                    >
                      <div className="flex items-center justify-center gap-1">
                        <motion.div
                          animate={{ y: [0, -3, 0] }}
                          transition={{
                            duration: 1,
                            repeat: Infinity,
                            ease: "easeInOut",
                            delay: 0,
                          }}
                          className="size-3 bg-background/80 rounded-full"
                        />
                        <motion.div
                          animate={{ y: [0, -3, 0] }}
                          transition={{
                            duration: 1,
                            repeat: Infinity,
                            ease: "easeInOut",
                            delay: 0.15,
                          }}
                          className="size-3 bg-background/80 rounded-full"
                        />
                        <motion.div
                          animate={{ y: [0, -3, 0] }}
                          transition={{
                            duration: 1,
                            repeat: Infinity,
                            ease: "easeInOut",
                            delay: 0.3,
                          }}
                          className="size-3 bg-background/80 rounded-full"
                        />
                      </div>
                    </motion.div>
                  </div>
                  <motion.p
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.4, delay: 0.5 }}
                    className="w-full text-muted-foreground md:w-3/5 mx-auto mb-6"
                  >
                    This is a temporary chat. It won't be saved to your
                    conversation history, can't use memory, and will be deleted
                    when you close it.
                  </motion.p>
                </>
              ) : agentModeEnabled && agentModeAvailable ? (
                <AgentModeHero phrase={agentPhrase} />
              ) : (
                <div className="flex items-center justify-center gap-2">
                  <motion.h2
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.4 }}
                    className="text-2xl font-medium mb-6 font-studio"
                  >
                    {randomPhrase}
                  </motion.h2>
                </div>
              )}

              <ChatInput initialConversation={true} />
            </div>
          </div>
        </div>
      </SidebarInset>
    </>
  );
}

export default function AppPage() {
  const isPrivateChat = usePrivateChat((state) => state.isPrivateChat);

  return (
    <SidebarProvider
      className={cn(
        isPrivateChat &&
          '**:data-[slot="sidebar"]:opacity-0 **:data-[slot="sidebar"]:-translate-x-full **:data-[slot="sidebar-gap"]:w-0 **:data-[slot="sidebar"]:transition-all **:data-[slot="sidebar-gap"]:transition-all **:data-[slot="sidebar"]:duration-300 **:data-[slot="sidebar-gap"]:duration-300',
      )}
    >
      <AppPageContent />
    </SidebarProvider>
  );
}
