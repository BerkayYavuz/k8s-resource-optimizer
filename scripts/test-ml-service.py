"""
Test script for ML Service
Tests the prediction endpoints with sample data
"""

import requests
import json
from datetime import datetime, timedelta

BASE_URL = "http://localhost:5000"

def test_health():
    """Test health endpoint"""
    print("🏥 Testing health endpoint...")
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        if response.status_code == 200:
            print("   ✅ Health check passed!")
            print(f"   {json.dumps(response.json(), indent=6)}")
            return True
        else:
            print(f"   ❌ Health check failed with status {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Could not connect to ML service: {e}")
        print(f"   Make sure the ML service is running on {BASE_URL}")
        return False

def generate_sample_data(days=7, points_per_day=24):
    """Generate sample time series data with daily pattern"""
    base_time = datetime.now() - timedelta(days=days)
    data = []
    
    for i in range(days * points_per_day):
        timestamp = (base_time + timedelta(hours=i)).isoformat()
        # Create a pattern: higher during "business hours" (8-18 UTC)
        hour_of_day = i % 24
        base_value = 0.2
        if 8 <= hour_of_day <= 18:
            base_value = 0.5
        
        # Add some variation
        import random
        value = base_value + random.uniform(-0.05, 0.05)
        
        data.append({
            "timestamp": timestamp,
            "value": max(0.01, value)  # Ensure positive
        })
    
    return data

def test_cpu_prediction():
    """Test CPU prediction endpoint"""
    print("\n🔮 Testing CPU prediction...")
    
    sample_data = generate_sample_data(days=7, points_per_day=24)
    
    payload = {
        "container": "test-container",
        "cpu": sample_data
    }
    
    try:
        response = requests.post(
            f"{BASE_URL}/predict/cpu",
            json=payload,
            timeout=30
        )
        
        if response.status_code == 200:
            result = response.json()
            print("   ✅ CPU prediction succeeded!")
            
            if result.get('success'):
                pred = result.get('prediction', {})
                print(f"   📊 Average prediction: {pred.get('avg_prediction', 0):.4f} cores")
                print(f"   📊 Peak prediction: {pred.get('peak_prediction', 0):.4f} cores")
                print(f"   📊 Confidence: {pred.get('confidence', 0):.2%}")
                print(f"   📊 Model: {pred.get('model', 'unknown')}")
                return True
            else:
                print(f"   ❌ Prediction failed: {result.get('error')}")
                return False
        else:
            print(f"   ❌ Request failed with status {response.status_code}")
            print(f"   {response.text}")
            return False
            
    except Exception as e:
        print(f"   ❌ Error during prediction: {e}")
        return False

def test_memory_prediction():
    """Test memory prediction endpoint"""
    print("\n💾 Testing memory prediction...")
    
    # Generate memory data (in MB)
    sample_data = generate_sample_data(days=7, points_per_day=24)
    # Scale values to represent memory in MB (e.g., 200-500 MB)
    for point in sample_data:
        point['value'] = point['value'] * 1000 + 200
    
    payload = {
        "container": "test-container",
        "memory": sample_data
    }
    
    try:
        response = requests.post(
            f"{BASE_URL}/predict/memory",
            json=payload,
            timeout=30
        )
        
        if response.status_code == 200:
            result = response.json()
            print("   ✅ Memory prediction succeeded!")
            
            if result.get('success'):
                pred = result.get('prediction', {})
                print(f"   📊 Average prediction: {pred.get('avg_prediction', 0):.2f} MB")
                print(f"   📊 Peak prediction: {pred.get('peak_prediction', 0):.2f} MB")
                print(f"   📊 Confidence: {pred.get('confidence', 0):.2%}")
                return True
            else:
                print(f"   ❌ Prediction failed: {result.get('error')}")
                return False
        else:
            print(f"   ❌ Request failed with status {response.status_code}")
            return False
            
    except Exception as e:
        print(f"   ❌ Error during prediction: {e}")
        return False

def test_full_prediction():
    """Test full prediction endpoint (both CPU and memory)"""
    print("\n🎯 Testing full prediction (CPU + Memory)...")
    
    cpu_data = generate_sample_data(days=7, points_per_day=24)
    memory_data = generate_sample_data(days=7, points_per_day=24)
    
    # Scale memory to MB range
    for point in memory_data:
        point['value'] = point['value'] * 1000 + 200
    
    payload = {
        "pod_name": "test-pod",
        "namespace": "default",
        "metrics": {
            "container1": {
                "cpu": cpu_data,
                "memory": memory_data
            }
        }
    }
    
    try:
        response = requests.post(
            f"{BASE_URL}/predict/all",
            json=payload,
            timeout=60
        )
        
        if response.status_code == 200:
            result = response.json()
            print("   ✅ Full prediction succeeded!")
            
            if result.get('success'):
                predictions = result.get('predictions', {})
                for container, pred in predictions.items():
                    print(f"\n   📦 Container: {container}")
                    print(f"      CPU Avg:  {pred.get('predicted_avg_cpu', 0):.4f} cores")
                    print(f"      CPU Peak: {pred.get('predicted_peak_cpu', 0):.4f} cores")
                    print(f"      Mem Avg:  {pred.get('predicted_avg_memory', 0):.2f} MB")
                    print(f"      Mem Peak: {pred.get('predicted_peak_memory', 0):.2f} MB")
                    print(f"      Confidence: {pred.get('confidence', 0):.2%}")
                return True
            else:
                print(f"   ❌ Prediction failed: {result.get('error')}")
                return False
        else:
            print(f"   ❌ Request failed with status {response.status_code}")
            return False
            
    except Exception as e:
        print(f"   ❌ Error during prediction: {e}")
        return False

def main():
    """Run all tests"""
    print("=" * 60)
    print("🧪 ML Service Test Suite")
    print("=" * 60)
    
    results = []
    
    # Test health first
    if not test_health():
        print("\n❌ Cannot proceed - ML service is not responding")
        print("   Please start the ML service first:")
        print("   cd ml-service")
        print("   python api/app.py")
        return
    
    # Run prediction tests
    results.append(("CPU Prediction", test_cpu_prediction()))
    results.append(("Memory Prediction", test_memory_prediction()))
    results.append(("Full Prediction", test_full_prediction()))
    
    # Summary
    print("\n" + "=" * 60)
    print("📊 Test Summary")
    print("=" * 60)
    
    passed = sum(1 for _, result in results if result)
    total = len(results)
    
    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"  {status} - {test_name}")
    
    print(f"\n  Total: {passed}/{total} tests passed")
    
    if passed == total:
        print("\n🎉 All tests passed!")
    else:
        print(f"\n⚠️  {total - passed} test(s) failed")
    
    print("=" * 60)

if __name__ == "__main__":
    main()
