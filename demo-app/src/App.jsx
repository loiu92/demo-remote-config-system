import React, { useState, useEffect } from 'react';
import { ShoppingBag, Settings, Palette, Eye, DollarSign, Star, Bell, RefreshCw } from 'lucide-react';
import useRemoteConfig from './hooks/useRemoteConfig';
import ConnectionStatus from './components/ConnectionStatus';
import CountdownTimer from './components/CountdownTimer';
import ProductCard from './components/ProductCard';

// Sample product data
const sampleProducts = [
  {
    id: 1,
    name: "Premium Wireless Headphones",
    description: "High-quality sound with noise cancellation",
    price: 199.99,
    originalPrice: 299.99,
    image: "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400&h=300&fit=crop",
    rating: 4.5,
    reviews: 128
  },
  {
    id: 2,
    name: "Smart Fitness Watch",
    description: "Track your health and fitness goals",
    price: 249.99,
    originalPrice: 349.99,
    image: "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=400&h=300&fit=crop",
    rating: 4.3,
    reviews: 89
  },
  {
    id: 3,
    name: "Laptop Stand",
    description: "Ergonomic aluminum laptop stand",
    price: 79.99,
    originalPrice: 99.99,
    image: "https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=400&h=300&fit=crop",
    rating: 4.7,
    reviews: 203
  },
  {
    id: 4,
    name: "Wireless Mouse",
    description: "Precision wireless mouse for productivity",
    price: 49.99,
    originalPrice: 69.99,
    image: "https://images.unsplash.com/photo-1527814050087-3793815479db?w=400&h=300&fit=crop",
    rating: 4.2,
    reviews: 156
  }
];

function App() {
  const [cart, setCart] = useState([]);
  const [showConfigPanel, setShowConfigPanel] = useState(false);
  const [showUpdateNotification, setShowUpdateNotification] = useState(false);

  // Use remote configuration
  const { config, loading, error, connected, lastUpdate, refetch } = useRemoteConfig('demo', 'shopflow', 'production');

  // Show notification when config updates
  useEffect(() => {
    if (lastUpdate && !loading) {
      setShowUpdateNotification(true);
      const timer = setTimeout(() => setShowUpdateNotification(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [lastUpdate, loading]);

  // Default configuration
  const defaultConfig = {
    theme: 'light',
    showPrices: true,
    showRatings: true,
    enablePromotions: true,
    promotionEndTime: null,
    promotionTitle: 'Black Friday Sale',
    bannerMessage: 'Welcome to ShopFlow Lite!',
    showBanner: true,
    maxItemsPerRow: 2
  };

  // Merge remote config with defaults
  const activeConfig = { ...defaultConfig, ...config };

  const addToCart = (product) => {
    setCart(prev => {
      const existing = prev.find(item => item.id === product.id);
      if (existing) {
        return prev.map(item =>
          item.id === product.id
            ? { ...item, quantity: item.quantity + 1 }
            : item
        );
      }
      return [...prev, { ...product, quantity: 1 }];
    });
  };

  const cartTotal = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  const cartItemCount = cart.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <div className={`min-h-screen transition-all duration-500 relative ${
      activeConfig.theme === 'dark'
        ? 'bg-gray-900 text-white'
        : activeConfig.theme === 'colorful'
        ? 'bg-gradient-to-br from-purple-50 via-pink-50 to-orange-50'
        : 'bg-gray-50'
    }`}>
      {/* Update Notification */}
      {showUpdateNotification && (
        <div className="fixed top-4 right-4 z-50 bg-green-500 text-white px-4 py-2 rounded-lg shadow-lg animate-slide-up">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 bg-white rounded-full animate-pulse"></div>
            Configuration updated!
          </div>
        </div>
      )}
      {/* Header */}
      <header className={`sticky top-0 z-50 backdrop-blur-sm border-b transition-all duration-300 ${
        activeConfig.theme === 'dark'
          ? 'bg-gray-800/90 border-gray-700'
          : activeConfig.theme === 'colorful'
          ? 'bg-white/90 border-purple-200'
          : 'bg-white/90 border-gray-200'
      }`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-3">
              <ShoppingBag className="w-8 h-8 text-primary-600" />
              <h1 className="text-xl font-bold">ShopFlow Lite</h1>
              <span className="badge badge-info">Demo</span>
            </div>
            
            <div className="flex items-center gap-4">
              
              <div className="flex items-center gap-2 px-3 py-2 bg-primary-100 text-primary-800 rounded-lg">
                <ShoppingBag className="w-4 h-4" />
                <span className="font-medium">{cartItemCount}</span>
                <span className="text-sm">${cartTotal.toFixed(2)}</span>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Banner */}
      {activeConfig.showBanner && activeConfig.bannerMessage && (
        <div className={`py-3 text-center transition-all duration-300 ${
          activeConfig.theme === 'dark'
            ? 'bg-blue-900 text-blue-100'
            : activeConfig.theme === 'colorful'
            ? 'bg-gradient-to-r from-purple-600 to-pink-600 text-white'
            : 'bg-blue-600 text-white'
        }`}>
          <div className="flex items-center justify-center gap-2">
            <Bell className="w-4 h-4" />
            <span className="font-medium">{activeConfig.bannerMessage}</span>
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="w-full">
            {/* Promotion Timer */}
            {activeConfig.enablePromotions && activeConfig.promotionEndTime && (
              <div className="mb-8">
                <CountdownTimer
                  targetDate={activeConfig.promotionEndTime}
                  title={activeConfig.promotionTitle}
                  onExpire={() => console.log('Promotion expired!')}
                />
              </div>
            )}

            {/* Error State */}
            {error && (
              <div className="mb-6 p-4 bg-red-100 border border-red-200 text-red-800 rounded-lg">
                <div className="font-medium">Configuration Error</div>
                <div className="text-sm mt-1">{error}</div>
              </div>
            )}

            {/* Loading State */}
            {loading && (
              <div className="mb-6 p-4 bg-blue-100 border border-blue-200 text-blue-800 rounded-lg">
                <div className="flex items-center gap-2">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
                  Loading configuration...
                </div>
              </div>
            )}

            {/* Products Grid */}
            <div className={`grid gap-6 ${
              activeConfig.maxItemsPerRow === 1 ? 'grid-cols-1' :
              activeConfig.maxItemsPerRow === 2 ? 'grid-cols-1 md:grid-cols-2' :
              activeConfig.maxItemsPerRow === 3 ? 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3' :
              'grid-cols-1 md:grid-cols-2 lg:grid-cols-4'
            }`}>
              {sampleProducts.map(product => (
                <ProductCard
                  key={product.id}
                  product={product}
                  theme={activeConfig.theme}
                  showPrices={activeConfig.showPrices}
                  showRatings={activeConfig.showRatings}
                  onAddToCart={addToCart}
                />
              ))}
            </div>
        </div>

        {/* Floating Config Components */}
        <>
          {/* Floating Config Button */}
            <button
              onClick={() => setShowConfigPanel(!showConfigPanel)}
              className={`fixed bottom-6 right-6 z-50 w-12 h-12 rounded-full shadow-lg transition-all duration-200 hover:scale-110 ${
                connected ? 'animate-pulse-slow' : ''
              } ${
                activeConfig.theme === 'dark'
                  ? 'bg-gray-700 hover:bg-gray-600 text-white'
                  : activeConfig.theme === 'colorful'
                  ? 'bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-700 hover:to-pink-700 text-white'
                  : 'bg-blue-600 hover:bg-blue-700 text-white'
              }`}
              title="Configuration Panel"
            >
              <Settings className={`w-6 h-6 mx-auto transition-transform duration-200 ${showConfigPanel ? 'rotate-90' : ''}`} />
            </button>

            {/* Floating Config Panel */}
            {showConfigPanel && (
            <div className="fixed inset-0 z-40 flex items-end justify-end p-6">
              {/* Backdrop */}
              <div
                className="absolute inset-0 bg-black bg-opacity-20"
                onClick={() => setShowConfigPanel(false)}
              />

              {/* Panel */}
              <div className={`relative w-96 max-h-[80vh] overflow-y-auto rounded-xl shadow-2xl border transition-all duration-300 ${
                activeConfig.theme === 'dark'
                  ? 'bg-gray-800 border-gray-700 text-white'
                  : activeConfig.theme === 'colorful'
                  ? 'bg-gradient-to-br from-purple-50 to-pink-50 border-purple-200 text-gray-900'
                  : 'bg-white border-gray-200 text-gray-900'
              }`}>
                <div className="p-6">
                  {/* Header */}
                  <div className="flex items-center justify-between mb-6">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                      <Settings className="w-5 h-5" />
                      Live Configuration
                    </h3>
                    <button
                      onClick={() => setShowConfigPanel(false)}
                      className={`p-1 rounded-lg transition-colors ${
                        activeConfig.theme === 'dark'
                          ? 'hover:bg-gray-700'
                          : 'hover:bg-gray-100'
                      }`}
                    >
                      ✕
                    </button>
                  </div>

                  {/* Connection Status */}
                  <div className={`p-4 rounded-lg mb-6 ${
                    activeConfig.theme === 'dark'
                      ? 'bg-gray-700'
                      : activeConfig.theme === 'colorful'
                      ? 'bg-white bg-opacity-60'
                      : 'bg-gray-50'
                  }`}>
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="font-medium">Connection Status</h4>
                      <button
                        onClick={refetch}
                        disabled={loading}
                        className={`flex items-center gap-1 px-3 py-1 text-sm rounded-lg transition-colors disabled:opacity-50 ${
                          activeConfig.theme === 'dark'
                            ? 'bg-blue-600 hover:bg-blue-700 text-white'
                            : 'bg-blue-100 hover:bg-blue-200 text-blue-700'
                        }`}
                      >
                        <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                        Refresh
                      </button>
                    </div>

                    <div className="space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        {error ? (
                          <>
                            <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
                            <span className="text-orange-600">Error</span>
                          </>
                        ) : connected ? (
                          <>
                            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                            <span className="text-green-600">Live Updates Active</span>
                          </>
                        ) : (
                          <>
                            <div className="w-2 h-2 bg-red-500 rounded-full"></div>
                            <span className="text-red-600">Disconnected</span>
                          </>
                        )}
                      </div>

                      {lastUpdate && (
                        <div className={`text-xs ${
                          activeConfig.theme === 'dark' ? 'text-gray-400' : 'text-gray-500'
                        }`}>
                          Last update: {lastUpdate.toLocaleTimeString()}
                        </div>
                      )}

                      {error && (
                        <div className="text-xs text-orange-600 mt-1">
                          {error}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Configuration Values */}
                  <div className="space-y-4 mb-6">
                    <h4 className="font-medium">Current Configuration</h4>

                    <div className="space-y-3 text-sm">
                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <Palette className="w-4 h-4" />
                          Theme
                        </span>
                        <span className={`px-2 py-1 rounded text-xs font-medium ${
                          activeConfig.theme === 'dark'
                            ? 'bg-blue-600 text-white'
                            : 'bg-blue-100 text-blue-800'
                        }`}>
                          {activeConfig.theme}
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <DollarSign className="w-4 h-4" />
                          Show Prices
                        </span>
                        <span className={`px-2 py-1 rounded text-xs font-medium ${
                          activeConfig.showPrices
                            ? (activeConfig.theme === 'dark' ? 'bg-green-600 text-white' : 'bg-green-100 text-green-800')
                            : (activeConfig.theme === 'dark' ? 'bg-red-600 text-white' : 'bg-red-100 text-red-800')
                        }`}>
                          {activeConfig.showPrices ? 'ON' : 'OFF'}
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <Star className="w-4 h-4" />
                          Show Ratings
                        </span>
                        <span className={`px-2 py-1 rounded text-xs font-medium ${
                          activeConfig.showRatings
                            ? (activeConfig.theme === 'dark' ? 'bg-green-600 text-white' : 'bg-green-100 text-green-800')
                            : (activeConfig.theme === 'dark' ? 'bg-red-600 text-white' : 'bg-red-100 text-red-800')
                        }`}>
                          {activeConfig.showRatings ? 'ON' : 'OFF'}
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <Eye className="w-4 h-4" />
                          Items per Row
                        </span>
                        <span className={`px-2 py-1 rounded text-xs font-medium ${
                          activeConfig.theme === 'dark'
                            ? 'bg-blue-600 text-white'
                            : 'bg-blue-100 text-blue-800'
                        }`}>
                          {activeConfig.maxItemsPerRow}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Raw Configuration */}
                  <div>
                    <h4 className="font-medium mb-2">Raw Configuration</h4>
                    <pre className={`text-xs p-3 rounded overflow-auto max-h-40 ${
                      activeConfig.theme === 'dark'
                        ? 'bg-gray-900 text-gray-300'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {JSON.stringify(activeConfig, null, 2)}
                    </pre>
                  </div>
                </div>
              </div>
            </div>
            )}
        </>
      </div>
    </div>
  );
}

export default App;
