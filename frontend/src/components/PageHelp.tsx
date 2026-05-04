import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface PageHelpProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  description: React.ReactNode;
  impacts?: React.ReactNode;
  videoId?: string;
}

export const PageHelp = ({ isOpen, onClose, title, description, impacts, videoId }: PageHelpProps) => {
  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/20 backdrop-blur-sm p-4 md:p-12">
          <motion.div
            initial={{ opacity: 0, y: 10, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 10, scale: 0.98 }}
            transition={{ duration: 0.2 }}
            className="bg-white rounded-xl p-8 md:p-10 w-full max-w-3xl max-h-[90vh] overflow-y-auto shadow-2xl"
          >
            <div className="flex justify-between items-start mb-6 pb-4 border-b border-gray-100">
              <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>
              <button 
                onClick={onClose}
                className="text-gray-400 hover:text-gray-600 p-2 rounded-md hover:bg-gray-100 transition-colors"
                aria-label="Fechar"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 6L6 18M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="space-y-8">
              <section>
                <h3 className="text-sm font-medium text-gray-500 mb-2">Para que serve</h3>
                <div className="text-base text-gray-800 leading-relaxed">{description}</div>
              </section>
              
              {impacts && (
                <section>
                  <h3 className="text-sm font-medium text-gray-500 mb-2">Impactos no Software</h3>
                  <div className="text-base text-gray-800 leading-relaxed bg-gray-50 rounded-lg p-4">
                    {impacts}
                  </div>
                </section>
              )}
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};
