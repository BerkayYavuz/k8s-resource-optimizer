"""
Prophet-based time series prediction model.
Handles trend detection, seasonality, and provides confidence intervals.
"""

import logging
from datetime import datetime
from typing import List, Dict, Tuple

import pandas as pd
from prophet import Prophet

logger = logging.getLogger(__name__)


class ProphetModel:
    """Prophet model for time series forecasting"""
    
    def __init__(self):
        self.model = None
        
    def predict(self, data_points: List[Dict], forecast_hours: int = 168) -> Dict:
        """
        Predict future resource usage using Prophet.
        
        Args:
            data_points: List of {"timestamp": str, "value": float}
            forecast_hours: How many hours ahead to forecast (default: 7 days)
            
        Returns:
            Dictionary with predictions and confidence intervals
        """
        if not data_points or len(data_points) < 2:
            logger.warning("Insufficient data points for Prophet prediction")
            return {
                "avg_prediction": 0.0,
                "peak_prediction": 0.0,
                "confidence": 0.0,
                "error": "Insufficient data"
            }
        
        try:
            # Prepare data in Prophet format
            df = self._prepare_data(data_points)
            
            if len(df) < 2:
                return {
                    "avg_prediction": df['y'].mean() if len(df) > 0 else 0.0,
                    "peak_prediction": df['y'].max() if len(df) > 0 else 0.0,
                    "confidence": 0.5,
                    "error": "Not enough data points"
                }
            
            # Initialize and fit Prophet model
            self.model = Prophet(
                daily_seasonality=True,
                weekly_seasonality=True,
                yearly_seasonality=False,  # Not relevant for short-term predictions
                interval_width=0.95,  # 95% confidence interval
                changepoint_prior_scale=0.05,  # Regularization
            )
            
            # Suppress Prophet's verbose output
            self.model.fit(df, algorithm='Newton')
            
            # Create future dataframe
            future = self.model.make_future_dataframe(periods=forecast_hours, freq='h')
            
            # Make predictions
            forecast = self.model.predict(future)
            
            # Extract predictions for the forecast period
            future_forecast = forecast.tail(forecast_hours)
            
            # Calculate average and peak predictions
            avg_prediction = future_forecast['yhat'].mean()
            peak_prediction = future_forecast['yhat_upper'].max()  # Use upper bound for peaks
            
            # Ensure non-negative predictions
            avg_prediction = max(0.0, avg_prediction)
            peak_prediction = max(0.0, peak_prediction)
            
            # Calculate confidence based on prediction interval width
            interval_width = (future_forecast['yhat_upper'] - future_forecast['yhat_lower']).mean()
            avg_value = future_forecast['yhat'].mean()
            
            # Confidence: narrower interval relative to value = higher confidence
            if avg_value > 0:
                confidence = max(0.0, min(1.0, 1.0 - (interval_width / (2 * avg_value))))
            else:
                confidence = 0.5
            
            return {
                "avg_prediction": avg_prediction,
                "peak_prediction": peak_prediction,
                "confidence": confidence,
                "trend": self._extract_trend(forecast),
            }
            
        except Exception as e:
            logger.error(f"Prophet prediction failed: {e}")
            # Fallback to simple statistics
            values = [dp["value"] for dp in data_points]
            return {
                "avg_prediction": sum(values) / len(values),
                "peak_prediction": max(values) * 1.2,  # 20% buffer
                "confidence": 0.3,
                "error": str(e)
            }
    
    def _prepare_data(self, data_points: List[Dict]) -> pd.DataFrame:
        """Convert data points to Prophet format (ds, y)"""
        records = []
        for dp in data_points:
            try:
                # Parse ISO 8601 timestamp
                timestamp = pd.to_datetime(dp["timestamp"])
                value = float(dp["value"])
                records.append({"ds": timestamp, "y": value})
            except Exception as e:
                logger.warning(f"Failed to parse data point: {e}")
                continue
        
        df = pd.DataFrame(records)
        
        # Remove duplicates and sort
        df = df.drop_duplicates(subset=['ds']).sort_values('ds')
        
        return df
    
    def _extract_trend(self, forecast: pd.DataFrame) -> str:
        """Determine if trend is increasing, decreasing, or stable"""
        trend_values = forecast['trend'].tail(168).values  # Last week
        
        if len(trend_values) < 2:
            return "unknown"
        
        start_avg = trend_values[:24].mean()
        end_avg = trend_values[-24:].mean()
        
        change_pct = ((end_avg - start_avg) / start_avg) * 100 if start_avg > 0 else 0
        
        if change_pct > 5:
            return "increasing"
        elif change_pct < -5:
            return "decreasing"
        else:
            return "stable"
