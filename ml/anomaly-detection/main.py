from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Literal
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
from sklearn.ensemble import IsolationForest
import os
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

USD_TO_BRL_RATE = float(os.getenv("USD_TO_BRL_RATE", "5.15"))

app = FastAPI(title="FinOps Anomaly Detection Engine", version="1.0.0")

class AnomalyRequest(BaseModel):
    provider: str
    account_id: str
    service: Optional[str] = None
    method: Literal["isolation_forest", "zscore", "rolling_average", "ensemble"] = "ensemble"
    sensitivity: float = 0.05  # For isolation forest contamination
    window: int = 7  # For rolling average

class AnomalyPoint(BaseModel):
    date: str
    value: float
    expected: float
    deviation: float
    severity: str
    score: float
    method: str

class AnomalyResponse(BaseModel):
    provider: str
    account_id: str
    anomalies: List[AnomalyPoint]
    total_anomalies: int
    total_impact: float
    baseline_mean: float
    baseline_std: float
    generated_at: str

class AnomalyRule(BaseModel):
    id: str
    name: str
    condition: str
    threshold: float
    severity: str
    enabled: bool = True

def generate_cost_data(days: int = 90) -> pd.DataFrame:
    np.random.seed(42)
    dates = pd.date_range(end=datetime.now(), periods=days, freq='D')
    base = 5000
    trend = np.linspace(0, 1000, days)
    seasonal = 300 * np.sin(2 * np.pi * np.arange(days) / 7)
    noise = np.random.normal(0, 150, days)
    costs = base + trend + seasonal + noise

    # Inject anomalies
    anomaly_indices = [20, 45, 70]
    for idx in anomaly_indices:
        if idx < days:
            costs[idx] *= 2.5  # 150% spike

    return pd.DataFrame({'date': dates, 'cost': costs})

def detect_isolation_forest(df: pd.DataFrame, contamination: float) -> pd.DataFrame:
    model = IsolationForest(contamination=contamination, random_state=42, n_estimators=100)
    df['score'] = model.fit_predict(df[['cost']])
    df['anomaly_score'] = model.decision_function(df[['cost']])
    df['is_anomaly'] = df['score'] == -1
    return df

def detect_zscore(df: pd.DataFrame, threshold: float = 3.0) -> pd.DataFrame:
    mean = df['cost'].mean()
    std = df['cost'].std()
    df['zscore'] = (df['cost'] - mean) / std
    df['is_anomaly'] = np.abs(df['zscore']) > threshold
    df['anomaly_score'] = np.abs(df['zscore'])
    return df

def detect_rolling_average(df: pd.DataFrame, window: int = 7) -> pd.DataFrame:
    df['rolling_mean'] = df['cost'].rolling(window=window, center=True).mean()
    df['rolling_std'] = df['cost'].rolling(window=window, center=True).std()
    df['upper_bound'] = df['rolling_mean'] + 2 * df['rolling_std']
    df['lower_bound'] = df['rolling_mean'] - 2 * df['rolling_std']
    df['is_anomaly'] = (df['cost'] > df['upper_bound']) | (df['cost'] < df['lower_bound'])
    df['anomaly_score'] = np.abs(df['cost'] - df['rolling_mean']) / df['rolling_std']
    df['anomaly_score'] = df['anomaly_score'].fillna(0)
    return df

def ensemble_detection(df: pd.DataFrame, sensitivity: float, window: int) -> pd.DataFrame:
    if1 = detect_isolation_forest(df.copy(), sensitivity)
    zs = detect_zscore(df.copy(), threshold=2.5)
    ra = detect_rolling_average(df.copy(), window)

    df['is_anomaly'] = if1['is_anomaly'] | zs['is_anomaly'] | ra['is_anomaly']
    df['anomaly_score'] = (if1['anomaly_score'].fillna(0) + zs['anomaly_score'].fillna(0) + ra['anomaly_score'].fillna(0)) / 3
    df['expected'] = ra['rolling_mean']
    return df

@app.get("/health")
def health():
    return {"status": "healthy", "service": "anomaly-detection"}

@app.get("/ready")
def ready():
    return {"status": "ready"}

@app.get("/live")
def live():
    return {"status": "alive"}

@app.get("/metrics")
def metrics():
    return {"requests_total": 0, "errors_total": 0}

@app.post("/api/v1/anomalies", response_model=AnomalyResponse)
def detect_anomalies(req: AnomalyRequest):
    logger.info(f"Anomaly detection request: {req}")

    df = generate_cost_data()
    baseline_mean = df['cost'].mean()
    baseline_std = df['cost'].std()

    if req.method == "isolation_forest":
        result = detect_isolation_forest(df, req.sensitivity)
        result['expected'] = baseline_mean
    elif req.method == "zscore":
        result = detect_zscore(df)
        result['expected'] = baseline_mean
    elif req.method == "rolling_average":
        result = detect_rolling_average(df, req.window)
    else:
        result = ensemble_detection(df, req.sensitivity, req.window)

    anomalies = []
    total_impact = 0.0

    for _, row in result.iterrows():
        if row['is_anomaly']:
            deviation = abs(row['cost'] - row.get('expected', baseline_mean))
            severity = "low"
            if deviation > 3 * baseline_std:
                severity = "critical"
            elif deviation > 2 * baseline_std:
                severity = "high"
            elif deviation > baseline_std:
                severity = "medium"

            anomalies.append(AnomalyPoint(
                date=row['date'].strftime('%Y-%m-%d'),
                value=round(float(row['cost']), 2),
                expected=round(float(row.get('expected', baseline_mean)), 2),
                deviation=round(float(deviation), 2),
                severity=severity,
                score=round(float(row['anomaly_score']), 4),
                method=req.method
            ))
            total_impact += deviation

    return AnomalyResponse(
        provider=req.provider,
        account_id=req.account_id,
        anomalies=anomalies,
        total_anomalies=len(anomalies),
        total_impact=round(total_impact, 2),
        baseline_mean=round(baseline_mean, 2),
        baseline_std=round(baseline_std, 2),
        generated_at=datetime.now().isoformat()
    )

@app.get("/api/v1/anomalies/{id}")
def get_anomaly(id: str):
    return {
        "id": id,
        "date": "2024-01-15",
        "value": 12500.00,
        "expected": 5000.00,
        "deviation": 7500.00,
        "severity": "critical",
        "service": "Compute",
        "resource": "vm-001",
        "recommendation": "Investigate unexpected compute spike"
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", 8002)))
