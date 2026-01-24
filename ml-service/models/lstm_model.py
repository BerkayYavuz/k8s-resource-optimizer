"""
LSTM-based time series prediction model.
Handles complex patterns and short-term predictions.
"""

import logging
from typing import List, Dict

import numpy as np
import pandas as pd
from sklearn.preprocessing import MinMaxScaler
import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers

logger = logging.getLogger(__name__)


class LSTMModel:
    """LSTM model for time series forecasting"""
    
    def __init__(self, lookback_steps: int = 24, forecast_steps: int = 168):
        """
        Args:
            lookback_steps: Number of past time steps to use for prediction
            forecast_steps: Number of future time steps to predict
        """
        self.lookback_steps = lookback_steps
        self.forecast_steps = forecast_steps
        self.model = None
        self.scaler = MinMaxScaler(feature_range=(0, 1))
        
    def predict(self, data_points: List[Dict]) -> Dict:
        """
        Predict future resource usage using LSTM.
        
        Args:
            data_points: List of {"timestamp": str, "value": float}
            
        Returns:
            Dictionary with predictions and confidence
        """
        if not data_points or len(data_points) < self.lookback_steps + 1:
            logger.warning("Insufficient data points for LSTM prediction")
            values = [dp["value"] for dp in data_points] if data_points else [0]
            return {
                "avg_prediction": sum(values) / len(values),
                "peak_prediction": max(values) * 1.2 if values else 0.0,
                "confidence": 0.2,
                "error": "Insufficient data"
            }
        
        try:
            # Prepare data
            values = np.array([dp["value"] for dp in data_points]).reshape(-1, 1)
            
            # Normalize data
            scaled_data = self.scaler.fit_transform(values)
            
            # Build and train model
            self._build_model()
            X, y = self._create_sequences(scaled_data)
            
            if len(X) < 10:  # Need minimum data for training
                logger.warning("Not enough sequences for LSTM training")
                return {
                    "avg_prediction": np.mean(values),
                    "peak_prediction": np.max(values) * 1.2,
                    "confidence": 0.3,
                    "error": "Not enough sequences"
                }
            
            # Train model (with validation split)
            history = self.model.fit(
                X, y,
                epochs=50,
                batch_size=32,
                validation_split=0.2,
                verbose=0,
                callbacks=[
                    keras.callbacks.EarlyStopping(
                        monitor='val_loss',
                        patience=5,
                        restore_best_weights=True
                    )
                ]
            )
            
            # Make predictions
            predictions = self._predict_future(scaled_data, steps=self.forecast_steps)
            
            # Inverse transform to original scale
            predictions = self.scaler.inverse_transform(predictions.reshape(-1, 1)).flatten()
            
            # Calculate metrics
            avg_prediction = np.mean(predictions)
            peak_prediction = np.max(predictions)
            
            # Ensure non-negative
            avg_prediction = max(0.0, avg_prediction)
            peak_prediction = max(0.0, peak_prediction)
            
            # Calculate confidence from training metrics
            final_loss = history.history['loss'][-1]
            val_loss = history.history['val_loss'][-1] if 'val_loss' in history.history else final_loss
            
            # Lower loss = higher confidence
            confidence = max(0.0, min(1.0, 1.0 - min(final_loss, val_loss)))
            
            return {
                "avg_prediction": float(avg_prediction),
                "peak_prediction": float(peak_prediction),
                "confidence": float(confidence),
                "training_loss": float(final_loss),
            }
            
        except Exception as e:
            logger.error(f"LSTM prediction failed: {e}")
            # Fallback to simple statistics
            values = [dp["value"] for dp in data_points]
            return {
                "avg_prediction": sum(values) / len(values),
                "peak_prediction": max(values) * 1.2,
                "confidence": 0.2,
                "error": str(e)
            }
    
    def _build_model(self):
        """Build LSTM model architecture"""
        self.model = keras.Sequential([
            layers.LSTM(64, activation='relu', return_sequences=True, 
                       input_shape=(self.lookback_steps, 1)),
            layers.Dropout(0.2),
            layers.LSTM(32, activation='relu'),
            layers.Dropout(0.2),
            layers.Dense(16, activation='relu'),
            layers.Dense(1)
        ])
        
        self.model.compile(
            optimizer=keras.optimizers.Adam(learning_rate=0.001),
            loss='mse',
            metrics=['mae']
        )
    
    def _create_sequences(self, data: np.ndarray):
        """Create sequences for supervised learning"""
        X, y = [], []
        
        for i in range(len(data) - self.lookback_steps):
            X.append(data[i:i + self.lookback_steps])
            y.append(data[i + self.lookback_steps])
        
        return np.array(X), np.array(y)
    
    def _predict_future(self, data: np.ndarray, steps: int) -> np.ndarray:
        """Predict multiple steps into the future"""
        # Use the last lookback_steps as starting point
        current_sequence = data[-self.lookback_steps:].reshape(1, self.lookback_steps, 1)
        predictions = []
        
        for _ in range(steps):
            # Predict next step
            next_pred = self.model.predict(current_sequence, verbose=0)[0, 0]
            predictions.append(next_pred)
            
            # Update sequence (slide window)
            current_sequence = np.append(current_sequence[:, 1:, :], 
                                        [[next_pred]]).reshape(1, self.lookback_steps, 1)
        
        return np.array(predictions)
