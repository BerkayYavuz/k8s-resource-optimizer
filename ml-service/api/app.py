"""
Flask REST API for ML prediction service.
Exposes endpoints for CPU and memory prediction.
"""

import logging
from datetime import datetime
from flask import Flask, request, jsonify

from models.ensemble import EnsemblePredictor

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create Flask app
app = Flask(__name__)

# Initialize ensemble predictor
predictor = EnsemblePredictor()

logger.info("ML Service API initialized")


@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'ml-service',
        'timestamp': datetime.now().isoformat()
    })


@app.route('/predict/cpu', methods=['POST'])
def predict_cpu():
    """Predict CPU usage for a container"""
    try:
        data = request.get_json()
        
        if not data or 'cpu' not in data:
            return jsonify({
                'success': False,
                'error': 'Missing CPU data in request'
            }), 400
        
        cpu_data = data['cpu']
        container = data.get('container', 'unknown')
        
        logger.info(f"CPU prediction request for container: {container}")
        
        # Make prediction
        result = predictor.predict_cpu(cpu_data)
        
        return jsonify({
            'success': True,
            'container': container,
            'prediction': result,
            'timestamp': datetime.now().isoformat()
        })
        
    except Exception as e:
        logger.error(f"CPU prediction failed: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/predict/memory', methods=['POST'])
def predict_memory():
    """Predict memory usage for a container"""
    try:
        data = request.get_json()
        
        if not data or 'memory' not in data:
            return jsonify({
                'success': False,
                'error': 'Missing memory data in request'
            }), 400
        
        memory_data = data['memory']
        container = data.get('container', 'unknown')
        
        logger.info(f"Memory prediction request for container: {container}")
        
        # Make prediction
        result = predictor.predict_memory(memory_data)
        
        return jsonify({
            'success': True,
            'container': container,
            'prediction': result,
            'timestamp': datetime.now().isoformat()
        })
        
    except Exception as e:
        logger.error(f"Memory prediction failed: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/predict/all', methods=['POST'])
def predict_all():
    """
    Predict both CPU and memory for all containers in a pod.
    Expected format:
    {
        "pod_name": "example",
        "namespace": "default",
        "metrics": {
            "container1": {
                "cpu": [...],
                "memory": [...]
            }
        }
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'metrics' not in data:
            return jsonify({
                'success': False,
                'error': 'Missing metrics in request'
            }), 400
        
        pod_name = data.get('pod_name', 'unknown')
        namespace = data.get('namespace', 'default')
        metrics = data['metrics']
        
        logger.info(f"Full prediction request for pod: {namespace}/{pod_name}, containers: {len(metrics)}")
        
        predictions = {}
        
        # Process each container
        for container_name, container_metrics in metrics.items():
            logger.info(f"Processing container: {container_name}")
            
            # Get CPU and memory data
            cpu_data = container_metrics.get('cpu', [])
            memory_data = container_metrics.get('memory', [])
            
            # Make predictions
            cpu_pred = predictor.predict_cpu(cpu_data) if cpu_data else {
                'avg_prediction': 0.0,
                'peak_prediction': 0.0,
                'confidence': 0.0
            }
            
            mem_pred = predictor.predict_memory(memory_data) if memory_data else {
                'avg_prediction': 0.0,
                'peak_prediction': 0.0,
                'confidence': 0.0
            }
            
            # Combine predictions
            predictions[container_name] = {
                'container': container_name,
                'predicted_avg_cpu': cpu_pred['avg_prediction'],
                'predicted_peak_cpu': cpu_pred['peak_prediction'],
                'predicted_avg_memory': mem_pred['avg_prediction'],
                'predicted_peak_memory': mem_pred['peak_prediction'],
                'confidence': (cpu_pred['confidence'] + mem_pred['confidence']) / 2,
                'model': 'ensemble',
                'details': {
                    'cpu': cpu_pred,
                    'memory': mem_pred
                }
            }
        
        logger.info(f"Completed predictions for {len(predictions)} containers")
        
        return jsonify({
            'success': True,
            'pod_name': pod_name,
            'namespace': namespace,
            'predictions': predictions,
            'timestamp': datetime.now().isoformat()
        })
        
    except Exception as e:
        logger.error(f"Full prediction failed: {e}", exc_info=True)
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.errorhandler(404)
def not_found(e):
    return jsonify({
        'success': False,
        'error': 'Endpoint not found'
    }), 404


@app.errorhandler(500)
def internal_error(e):
    return jsonify({
        'success': False,
        'error': 'Internal server error'
    }), 500


if __name__ == '__main__':
    # Run with debug mode off in production
    app.run(host='0.0.0.0', port=5000, debug=False)
