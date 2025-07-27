import React, { useState, useEffect } from 'react';
import { Clock, Zap } from 'lucide-react';

const CountdownTimer = ({ targetDate, title, onExpire }) => {
  const [timeLeft, setTimeLeft] = useState(null);
  const [isExpired, setIsExpired] = useState(false);

  useEffect(() => {
    if (!targetDate) return;

    const target = new Date(targetDate);
    
    const updateTimer = () => {
      const now = new Date();
      const difference = target - now;

      if (difference <= 0) {
        setIsExpired(true);
        setTimeLeft(null);
        if (onExpire) onExpire();
        return;
      }

      const days = Math.floor(difference / (1000 * 60 * 60 * 24));
      const hours = Math.floor((difference % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((difference % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((difference % (1000 * 60)) / 1000);

      setTimeLeft({ days, hours, minutes, seconds });
      setIsExpired(false);
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);

    return () => clearInterval(interval);
  }, [targetDate, onExpire]);

  if (!targetDate) return null;

  if (isExpired) {
    return (
      <div className="flex items-center gap-2 px-4 py-2 bg-red-100 text-red-800 rounded-lg">
        <Zap className="w-5 h-5" />
        <span className="font-medium">{title} has ended!</span>
      </div>
    );
  }

  if (!timeLeft) return null;

  return (
    <div className="flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-orange-100 to-red-100 rounded-lg border border-orange-200">
      <Clock className="w-5 h-5 text-orange-600" />
      <div className="flex-1">
        <div className="text-sm font-medium text-orange-800">{title}</div>
        <div className="flex items-center gap-4 mt-1">
          {timeLeft.days > 0 && (
            <div className="text-center">
              <div className="text-lg font-bold text-orange-900">{timeLeft.days}</div>
              <div className="text-xs text-orange-700">Days</div>
            </div>
          )}
          <div className="text-center">
            <div className="text-lg font-bold text-orange-900">{timeLeft.hours.toString().padStart(2, '0')}</div>
            <div className="text-xs text-orange-700">Hours</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-bold text-orange-900">{timeLeft.minutes.toString().padStart(2, '0')}</div>
            <div className="text-xs text-orange-700">Minutes</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-bold text-orange-900">{timeLeft.seconds.toString().padStart(2, '0')}</div>
            <div className="text-xs text-orange-700">Seconds</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default CountdownTimer;
