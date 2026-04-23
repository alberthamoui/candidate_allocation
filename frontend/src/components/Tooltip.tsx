import React, { useState, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface TooltipProps {
  content: React.ReactNode;
  children: React.ReactNode;
}

export function Tooltip({ content, children }: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  
  return (
    <div 
      className="relative inline-flex items-center justify-center group" 
      onMouseEnter={() => setIsVisible(true)}
      onMouseLeave={() => setIsVisible(false)}
      ref={containerRef}
    >
      {children}
      <AnimatePresence>
        {isVisible && (
          <motion.div
            initial={{ opacity: 0, y: 5 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 5 }}
            transition={{ duration: 0.15 }}
            className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 z-[9999] w-max max-w-xs minimal-panel p-3 text-xs shadow-lg pointer-events-none"
            style={{ minWidth: '200px', backgroundColor: 'var(--bg-panel)', color: 'var(--text)' }}
          >
            {content}
            <div className="absolute top-full left-1/2 -translate-x-1/2 w-2 h-2 bg-white rotate-45 -mt-[4px] border-b border-r border-gray-100"></div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export const HelpIcon = ({ text }: { text: string }) => (
  <Tooltip content={<div className="font-sans leading-relaxed text-xs">{text}</div>}>
    <span className="minimal-help-icon leading-none select-none">?</span>
  </Tooltip>
);
