"""
Ensemble predictor combining Prophet and LSTM models.
Uses weighted averaging based on confidence scores.
"""

import logging
from typing import List, Dict

from .prophet_model import ProphetModel
from .lstm_model import LSTMModel

logger = logging.getLogger(__name__)


class EnsemblePredictor:
    """Ensemble predictor combining multiple models"""
    
    def __init__(self):
        self.prophet = ProphetModel()
        self.lstm = LSTMModel()
        
    def predict(self, data_points: List[Dict], metric_type: str) -> Dict:
        """
        Generate ensemble prediction from Prophet and LSTM.
        
        Args:
            data_points: List of {"timestamp": str, "value": float}
            metric_type: "cpu" or "memory"
            
        Returns:
            Dictionary with ensemble predictions
        """
        if not data_points:
            return {
                "avg_prediction": 0.0,
                "peak_prediction": 0.0,
                "confidence": 0.0,
                "model": "ensemble",
                "error": "No data points"
            }
        
        logger.info(f"Running ensemble prediction for {metric_type} with {len(data_points)} data points")
        
        # Get predictions from both models
        prophet_result = self.prophet.predict(data_points)
        lstm_result = self.lstm.predict(data_points)
        
        # Extract predictions and confidences
        prophet_avg = prophet_result.get("avg_prediction", 0.0)
        prophet_peak = prophet_result.get("peak_prediction", 0.0)
        prophet_conf = prophet_result.get("confidence", 0.5)
        
        lstm_avg = lstm_result.get("avg_prediction", 0.0)
        lstm_peak = lstm_result.get("peak_prediction", 0.0)
        lstm_conf = lstm_result.get("confidence", 0.5)
        
        # Calculate weighted average based on confidence
        total_conf = prophet_conf + lstm_conf
        
        if total_conf > 0:
            prophet_weight = prophet_conf / total_conf
            lstm_weight = lstm_conf / total_conf
        else:
            # Equal weight if no confidence
            prophet_weight = 0.5
            lstm_weight = 0.5
        
        # Ensemble predictions
        avg_prediction = (prophet_avg * prophet_weight) + (lstm_avg * lstm_weight)
        peak_prediction = max(prophet_peak, lstm_peak)  # Use maximum for safety
        
        # Overall confidence (average of both)
        confidence = (prophet_conf + lstm_conf) / 2
        
        logger.info(f"Ensemble result - Avg: {avg_prediction:.4f}, Peak: {peak_prediction:.4f}, "
                   f"Confidence: {confidence:.4f} (Prophet: {prophet_conf:.2f}, LSTM: {lstm_conf:.2f})")
        
        return {
            "avg_prediction": avg_prediction,
            "peak_prediction": peak_prediction,
            "confidence": confidence,
            "model": "ensemble",
            "prophet_avg": prophet_avg,
            "prophet_peak": prophet_peak,
            "prophet_confidence": prophet_conf,
            "lstm_avg": lstm_avg,
            "lstm_peak": lstm_peak,
            "lstm_confidence": lstm_conf,
            "weights": {
                "prophet": prophet_weight,
                "lstm": lstm_weight
            }
        }
    
    def predict_cpu(self, data_points: List[Dict]) -> Dict:
        """Predict CPU usage"""
        return self.predict(data_points, "cpu")
    
    def predict_memory(self, data_points: List[Dict]) -> Dict:
        """Predict memory usage"""
        return self.predict(data_points, "memory")
