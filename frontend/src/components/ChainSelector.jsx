import { useState, useRef, useEffect } from 'react';
import { ChevronDown, Check } from 'lucide-react';

const ChainSelector = ({ chains, selectedChain, onSelect, disabled = [] }) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const availableChains = chains.filter(chain =>
    !disabled.find(d => d.id === chain.id)
  );

  // Debug: Log chains
  console.log('ChainSelector Debug:', {
    totalChains: chains.length,
    disabledChains: disabled.length,
    availableChains: availableChains.length,
    chainNames: availableChains.map(c => c.name)
  });

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between px-4 py-3 bg-white/5 border border-white/20 rounded-xl hover:bg-white/10 transition-all"
      >
        <div className="flex items-center space-x-3">
          <span className="text-2xl">{selectedChain.icon}</span>
          <div className="text-left">
            <div className="font-semibold">{selectedChain.name}</div>
            <div className="text-xs text-gray-400">{selectedChain.symbol}</div>
          </div>
        </div>
        <ChevronDown className={`w-5 h-5 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute z-10 w-full mt-2 bg-gray-900/95 backdrop-blur-lg border border-white/20 rounded-xl shadow-2xl max-h-80 overflow-y-auto">
          {availableChains.map((chain) => (
            <button
              key={chain.id}
              onClick={() => {
                onSelect(chain);
                setIsOpen(false);
              }}
              className="w-full flex items-center justify-between px-4 py-3 hover:bg-white/10 transition-colors first:rounded-t-xl last:rounded-b-xl"
            >
              <div className="flex items-center space-x-3">
                <span className="text-2xl">{chain.icon}</span>
                <div className="text-left">
                  <div className="font-semibold">{chain.name}</div>
                  <div className="text-xs text-gray-400">{chain.symbol}</div>
                </div>
              </div>
              {selectedChain.id === chain.id && (
                <Check className="w-5 h-5 text-green-400" />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default ChainSelector;
