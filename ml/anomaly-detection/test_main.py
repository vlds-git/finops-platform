import pytest
from fastapi.testclient import TestClient
from main import app

client = TestClient(app)

def test_health():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json()["status"] == "healthy"

def test_ready():
    response = client.get("/ready")
    assert response.status_code == 200

def test_live():
    response = client.get("/live")
    assert response.status_code == 200

def test_anomalies_ensemble():
    response = client.post("/api/v1/anomalies", json={
        "provider": "huawei",
        "account_id": "hw-001",
        "method": "ensemble",
        "sensitivity": 0.05,
        "window": 7
    })
    assert response.status_code == 200
    data = response.json()
    assert data["provider"] == "huawei"
    assert data["total_anomalies"] >= 0
    assert data["baseline_mean"] > 0
    assert data["baseline_std"] > 0

def test_anomalies_isolation_forest():
    response = client.post("/api/v1/anomalies", json={
        "provider": "huawei",
        "method": "isolation_forest",
        "sensitivity": 0.05
    })
    assert response.status_code == 200
    data = response.json()
    assert data["total_anomalies"] >= 0

def test_anomalies_zscore():
    response = client.post("/api/v1/anomalies", json={
        "provider": "huawei",
        "method": "zscore"
    })
    assert response.status_code == 200
    data = response.json()
    assert data["total_anomalies"] >= 0

def test_anomalies_rolling_average():
    response = client.post("/api/v1/anomalies", json={
        "provider": "huawei",
        "method": "rolling_average",
        "window": 7
    })
    assert response.status_code == 200
    data = response.json()
    assert data["total_anomalies"] >= 0

def test_get_anomaly_by_id():
    response = client.get("/api/v1/anomalies/test-001")
    assert response.status_code == 200
    data = response.json()
    assert data["id"] == "test-001"
    assert "severity" in data
