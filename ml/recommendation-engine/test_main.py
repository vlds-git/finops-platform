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

def test_recommendations_all():
    response = client.post("/api/v1/recommendations", json={
        "provider": "huawei",
        "account_id": "hw-001",
        "category": "all"
    })
    assert response.status_code == 200
    data = response.json()
    assert data["provider"] == "huawei"
    assert data["total_opportunities"] > 0
    assert data["total_savings"] > 0
    assert len(data["recommendations"]) > 0

def test_recommendations_rightsizing():
    response = client.post("/api/v1/recommendations", json={
        "provider": "huawei",
        "category": "rightsizing"
    })
    assert response.status_code == 200
    data = response.json()
    for rec in data["recommendations"]:
        assert rec["category"] == "rightsizing"

def test_recommendations_idle():
    response = client.post("/api/v1/recommendations", json={
        "provider": "huawei",
        "category": "idle"
    })
    assert response.status_code == 200
    data = response.json()
    for rec in data["recommendations"]:
        assert rec["category"] == "idle"

def test_recommendations_storage():
    response = client.post("/api/v1/recommendations", json={
        "provider": "huawei",
        "category": "storage"
    })
    assert response.status_code == 200

def test_recommendations_savings():
    response = client.post("/api/v1/recommendations", json={
        "provider": "huawei",
        "category": "savings"
    })
    assert response.status_code == 200

def test_list_recommendations():
    response = client.get("/api/v1/recommendations?provider=huawei&category=all")
    assert response.status_code == 200
    data = response.json()
    assert data["total_opportunities"] > 0

def test_apply_recommendation():
    response = client.post("/api/v1/recommendations/rec-001/apply")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "applied"
    assert "applied_at" in data
