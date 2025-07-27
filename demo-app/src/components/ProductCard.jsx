import React from 'react';
import { ShoppingCart, Star, Tag } from 'lucide-react';

const ProductCard = ({ product, theme, showPrices, showRatings, onAddToCart }) => {
  const themeClasses = {
    light: 'bg-white border-gray-200 text-gray-900',
    dark: 'bg-gray-800 border-gray-700 text-white',
    colorful: 'bg-gradient-to-br from-purple-50 to-pink-50 border-purple-200 text-gray-900'
  };

  const buttonClasses = {
    light: 'bg-blue-600 hover:bg-blue-700 text-white',
    dark: 'bg-blue-500 hover:bg-blue-600 text-white',
    colorful: 'bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-700 hover:to-pink-700 text-white'
  };

  return (
    <div className={`card ${themeClasses[theme]} transition-all duration-300 hover:shadow-lg group`}>
      <div className="aspect-w-16 aspect-h-12 mb-4">
        <img
          src={product.image}
          alt={product.name}
          className="w-full h-48 object-cover rounded-lg group-hover:scale-105 transition-transform duration-300"
        />
      </div>
      
      <div className="space-y-3">
        <div>
          <h3 className="font-semibold text-lg">{product.name}</h3>
          <p className="text-sm opacity-75 mt-1">{product.description}</p>
        </div>

        {showRatings && product.rating && (
          <div className="flex items-center gap-2">
            <div className="flex items-center">
              {[...Array(5)].map((_, i) => (
                <Star
                  key={i}
                  className={`w-4 h-4 ${
                    i < Math.floor(product.rating)
                      ? 'text-yellow-400 fill-current'
                      : 'text-gray-300'
                  }`}
                />
              ))}
            </div>
            <span className="text-sm opacity-75">({product.reviews} reviews)</span>
          </div>
        )}

        <div className="flex items-center justify-between">
          {showPrices && (
            <div className="flex items-center gap-2">
              {product.originalPrice && product.originalPrice !== product.price && (
                <span className="text-sm line-through opacity-60">
                  ${product.originalPrice}
                </span>
              )}
              <span className="text-xl font-bold text-green-600">
                ${product.price}
              </span>
              {product.originalPrice && product.originalPrice !== product.price && (
                <span className="badge badge-error">
                  <Tag className="w-3 h-3 mr-1" />
                  {Math.round(((product.originalPrice - product.price) / product.originalPrice) * 100)}% OFF
                </span>
              )}
            </div>
          )}
          
          <button
            onClick={() => onAddToCart(product)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-all duration-200 ${buttonClasses[theme]}`}
          >
            <ShoppingCart className="w-4 h-4" />
            Add to Cart
          </button>
        </div>
      </div>
    </div>
  );
};

export default ProductCard;
