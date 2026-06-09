import pytest
from fastapi.testclient import TestClient
from main import app

client = TestClient(app)

def test_health():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json()["status"] == "healthy"
    assert response.json()["service"] == "forecast-engine"

def test_ready():
    response = client.get("/ready")
    assert response.status_code == 200
    assert response.json()["status"] == "ready"

def test_live():
    response = client.get("/live")
    assert response.status_code == 200
    assert response.json()["status"] == "alive"

def test_metrics():
    response = client.get("/metrics")
    assert response.status_code == 200
    assert "requests_total" in response.json()

def test_forecast_30d():
    response = client.post("/api/v1/forecast", json={
        "provider": "huawei",
        "account_id": "hw-001",
        "period": "30d",
        "model": "ensemble"
    })
    assert response.status_code == 200
    data = response.json()
    assert data["provider"] == "huawei"
    assert data["period"] == "30d"
    assert data["model"] == "ensemble"
    assert len(data["forecast"]) == 30
    assert data["total_forecast"] > 0
    assert data["confidence"] > 0

def test_forecast_90d():
    response = client.post("/api/v1/forecast", json={
        "provider": "azure",
        "account_id": "az-001",
        "period": "90d",
        "model": "prophet"
    })
    assert response.status_code == 200
    data = response.json()
    assert len(data["forecast"]) == 90

def test_forecast_12m():
    response = client.post("/api/v1/forecast", json={
        "provider": "aws",
        "account_id": "aws-001",
        "period": "12m",
        "model": "arima"
    })
    assert response.status_code == 200
    data = response.json()
    assert len(data["forecast"]) == 365

def test_forecast_by_period():
    response = client.get("/api/v1/forecast/30d?provider=huawei&account_id=hw-001")
    assert response.status_code == 200
    data = response.json()
    assert data["period"] == "30d"
