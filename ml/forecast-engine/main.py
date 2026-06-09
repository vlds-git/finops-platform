from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Literal
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
import json
import os
from prophet import Prophet
from statsmodels.tsa.arima.model import ARIMA
from statsmodels.tsa.holtwinters import ExponentialSmoothing
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = FastAPI(title="FinOps Forecast Engine", version="1.0.0")

class ForecastRequest(BaseModel):
    provider: str
    account_id: str
    service: Optional[str] = None
    period: Literal["30d", "90d", "12m"] = "30d"
    model: Literal["prophet", "arima", "holt-winters", "ensemble"] = "ensemble"

class ForecastPoint(BaseModel):
    date: str
    value: float
    lower: float
    upper: float

class ForecastResponse(BaseModel):
    provider: str
    account_id: str
    period: str
    model: str
    forecast: List[ForecastPoint]
    total_forecast: float
    trend: float
    confidence: float
    generated_at: str

class TrendData(BaseModel):
    date: str
    cost: float

# Mock historical data generator
def generate_historical_data(days: int = 365) -> pd.DataFrame:
    np.random.seed(42)
    dates = pd.date_range(end=datetime.now(), periods=days, freq='D')
    base_cost = 5000
    trend = np.linspace(0, 2000, days)
    seasonal = 500 * np.sin(2 * np.pi * np.arange(days) / 30.5)  # Monthly seasonality
    weekly = 300 * np.sin(2 * np.pi * np.arange(days) / 7)       # Weekly seasonality
    noise = np.random.normal(0, 200, days)
    costs = base_cost + trend + seasonal + weekly + noise
    costs = np.maximum(costs, 1000)  # Ensure positive
    return pd.DataFrame({'ds': dates, 'y': costs})

def run_prophet(df: pd.DataFrame, periods: int) -> pd.DataFrame:
    m = Prophet(
        yearly_seasonality=True,
        weekly_seasonality=True,
        daily_seasonality=False,
        changepoint_prior_scale=0.05
    )
    m.fit(df)
    future = m.make_future_dataframe(periods=periods)
    forecast = m.predict(future)
    return forecast[['ds', 'yhat', 'yhat_lower', 'yhat_upper']].tail(periods)

def run_arima(df: pd.DataFrame, periods: int) -> pd.DataFrame:
    try:
        model = ARIMA(df['y'], order=(7, 1, 7))
        fitted = model.fit()
        forecast = fitted.forecast(steps=periods)
        # Simple confidence intervals
        std = df['y'].std()
        result = pd.DataFrame({
            'ds': pd.date_range(start=df['ds'].iloc[-1] + timedelta(days=1), periods=periods, freq='D'),
            'yhat': forecast.values,
            'yhat_lower': forecast.values - 1.96 * std,
            'yhat_upper': forecast.values + 1.96 * std
        })
        return result
    except Exception as e:
        logger.error(f"ARIMA failed: {e}")
        # Fallback to simple moving average
        ma = df['y'].rolling(window=30).mean().iloc[-1]
        result = pd.DataFrame({
            'ds': pd.date_range(start=df['ds'].iloc[-1] + timedelta(days=1), periods=periods, freq='D'),
            'yhat': [ma] * periods,
            'yhat_lower': [ma * 0.8] * periods,
            'yhat_upper': [ma * 1.2] * periods
        })
        return result

def run_holt_winters(df: pd.DataFrame, periods: int) -> pd.DataFrame:
    try:
        model = ExponentialSmoothing(
            df['y'],
            trend='add',
            seasonal='add',
            seasonal_periods=30
        )
        fitted = model.fit()
        forecast = fitted.forecast(steps=periods)
        std = df['y'].std()
        result = pd.DataFrame({
            'ds': pd.date_range(start=df['ds'].iloc[-1] + timedelta(days=1), periods=periods, freq='D'),
            'yhat': forecast.values,
            'yhat_lower': forecast.values - 1.96 * std,
            'yhat_upper': forecast.values + 1.96 * std
        })
        return result
    except Exception as e:
        logger.error(f"Holt-Winters failed: {e}")
        return run_arima(df, periods)

def run_ensemble(df: pd.DataFrame, periods: int) -> pd.DataFrame:
    p = run_prophet(df, periods)
    a = run_arima(df, periods)
    h = run_holt_winters(df, periods)

    result = pd.DataFrame({
        'ds': p['ds'],
        'yhat': (p['yhat'].values + a['yhat'].values + h['yhat'].values) / 3,
        'yhat_lower': np.minimum(np.minimum(p['yhat_lower'].values, a['yhat_lower'].values), h['yhat_lower'].values),
        'yhat_upper': np.maximum(np.maximum(p['yhat_upper'].values, a['yhat_upper'].values), h['yhat_upper'].values)
    })
    return result

@app.get("/health")
def health():
    return {"status": "healthy", "service": "forecast-engine"}

@app.get("/ready")
def ready():
    return {"status": "ready"}

@app.get("/live")
def live():
    return {"status": "alive"}

@app.get("/metrics")
def metrics():
    return {"requests_total": 0, "errors_total": 0}

@app.post("/api/v1/forecast", response_model=ForecastResponse)
def forecast(req: ForecastRequest):
    logger.info(f"Forecast request: {req}")

    period_map = {"30d": 30, "90d": 90, "12m": 365}
    periods = period_map.get(req.period, 30)

    df = generate_historical_data()

    if req.model == "prophet":
        result = run_prophet(df, periods)
    elif req.model == "arima":
        result = run_arima(df, periods)
    elif req.model == "holt-winters":
        result = run_holt_winters(df, periods)
    else:
        result = run_ensemble(df, periods)

    forecast_points = []
    for _, row in result.iterrows():
        forecast_points.append(ForecastPoint(
            date=row['ds'].strftime('%Y-%m-%d'),
            value=round(float(row['yhat']), 2),
            lower=round(float(row['yhat_lower']), 2),
            upper=round(float(row['yhat_upper']), 2)
        ))

    total = sum(p.value for p in forecast_points)
    trend = ((forecast_points[-1].value - forecast_points[0].value) / forecast_points[0].value) * 100 if forecast_points[0].value > 0 else 0

    return ForecastResponse(
        provider=req.provider,
        account_id=req.account_id,
        period=req.period,
        model=req.model,
        forecast=forecast_points,
        total_forecast=round(total, 2),
        trend=round(trend, 2),
        confidence=0.85,
        generated_at=datetime.now().isoformat()
    )

@app.get("/api/v1/forecast/{period}")
def forecast_by_period(period: str, provider: str = "all", account_id: str = "all"):
    req = ForecastRequest(provider=provider, account_id=account_id, period=period)
    return forecast(req)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", 8001)))
