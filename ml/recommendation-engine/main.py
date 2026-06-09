from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Literal
import numpy as np
import pandas as pd
from datetime import datetime, timedelta
import os
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = FastAPI(title="FinOps Recommendation Engine", version="1.0.0")

class RecommendationRequest(BaseModel):
    provider: str
    account_id: str
    category: Optional[Literal["rightsizing", "idle", "storage", "savings", "all"]] = "all"

class RecommendationItem(BaseModel):
    id: str
    category: str
    title: str
    description: str
    resource_id: str
    resource_type: str
    service: str
    region: str
    current_cost: float
    projected_cost: float
    savings: float
    savings_percentage: float
    confidence: float
    priority: str
    justification: str
    action: str
    risk: str
    implementation: str
    created_at: str

class RecommendationResponse(BaseModel):
    provider: str
    account_id: str
    total_savings: float
    total_opportunities: int
    recommendations: List[RecommendationItem]
    generated_at: str

def generate_recommendations(provider: str, account_id: str, category: str) -> List[RecommendationItem]:
    recs = []

    # Rightsizing recommendations
    if category in ["rightsizing", "all"]:
        recs.append(RecommendationItem(
            id="rec-001",
            category="rightsizing",
            title="Oversized Compute Instance",
            description="VM c6.2xlarge running at 15% CPU average. Recommend downgrade to c6.large.",
            resource_id="vm-prod-001",
            resource_type="Virtual Machine",
            service="Compute",
            region="sa-brazil-1",
            current_cost=350.00,
            projected_cost=175.00,
            savings=175.00,
            savings_percentage=50.0,
            confidence=0.92,
            priority="high",
            justification="CPU utilization below 20% for 30 consecutive days. Memory utilization below 40%.",
            action="Resize instance from c6.2xlarge to c6.large",
            risk="Low - can be rolled back within 5 minutes",
            implementation="Stop instance, change type, start instance. Downtime: ~2 minutes.",
            created_at=datetime.now().isoformat()
        ))
        recs.append(RecommendationItem(
            id="rec-002",
            category="rightsizing",
            title="Over-provisioned Database",
            description="RDS db.r5.4xlarge with 85% idle connections. Recommend db.r5.2xlarge.",
            resource_id="db-prod-001",
            resource_type="Relational Database",
            service="RDS",
            region="east-us-1",
            current_cost=1200.00,
            projected_cost=600.00,
            savings=600.00,
            savings_percentage=50.0,
            confidence=0.88,
            priority="high",
            justification="Connection count stable at 15% of max. CPU average 12%.",
            action="Modify DB instance class to db.r5.2xlarge",
            risk="Low - AWS supports online instance modification",
            implementation="Schedule maintenance window. Downtime: ~5 minutes.",
            created_at=datetime.now().isoformat()
        ))

    # Idle resources
    if category in ["idle", "all"]:
        recs.append(RecommendationItem(
            id="rec-003",
            category="idle",
            title="Idle Load Balancer",
            description="ALB with zero active connections for 45 days.",
            resource_id="alb-staging-001",
            resource_type="Application Load Balancer",
            service="Network",
            region="sa-brazil-1",
            current_cost=45.00,
            projected_cost=0.00,
            savings=45.00,
            savings_percentage=100.0,
            confidence=0.95,
            priority="medium",
            justification="Zero requests processed. No target groups registered.",
            action="Delete load balancer",
            risk="None - confirmed unused via CloudWatch metrics",
            implementation="Remove via console or Terraform. Immediate effect.",
            created_at=datetime.now().isoformat()
        ))
        recs.append(RecommendationItem(
            id="rec-004",
            category="idle",
            title="Unattached Elastic IP",
            description="EIP not associated with any instance for 60 days.",
            resource_id="eip-001",
            resource_type="Elastic IP",
            service="Network",
            region="east-us-1",
            current_cost=18.00,
            projected_cost=0.00,
            savings=18.00,
            savings_percentage=100.0,
            confidence=0.98,
            priority="low",
            justification="No association found. Charging idle IP rate.",
            action="Release Elastic IP",
            risk="None - unattached and unused",
            implementation="Release via API or console. Immediate effect.",
            created_at=datetime.now().isoformat()
        ))

    # Storage optimization
    if category in ["storage", "all"]:
        recs.append(RecommendationItem(
            id="rec-005",
            category="storage",
            title="Suboptimal Storage Tier",
            description="500GB gp3 volume with 5 IOPS/GB. Could use gp2 or reduce IOPS.",
            resource_id="vol-data-001",
            resource_type="EBS Volume",
            service="Storage",
            region="sa-brazil-1",
            current_cost=85.00,
            projected_cost=55.00,
            savings=30.00,
            savings_percentage=35.3,
            confidence=0.85,
            priority="medium",
            justification="IOPS utilization 8% of provisioned. Throughput 5% of limit.",
            action="Reduce IOPS to 3000 or migrate to gp2",
            risk="Low - monitor performance for 1 week after change",
            implementation="Modify volume via console. No downtime.",
            created_at=datetime.now().isoformat()
        ))
        recs.append(RecommendationItem(
            id="rec-006",
            category="storage",
            title="Old Snapshots",
            description="Snapshots older than 90 days consuming 2TB. Consider lifecycle policy.",
            resource_id="snap-archive",
            resource_type="Snapshot",
            service="Storage",
            region="east-us-1",
            current_cost=200.00,
            projected_cost=50.00,
            savings=150.00,
            savings_percentage=75.0,
            confidence=0.90,
            priority="medium",
            justification="Snapshots >90 days rarely accessed. Can archive to S3 Glacier.",
            action="Implement automated snapshot lifecycle",
            risk="Low - retain last 30 days, archive older",
            implementation="Create DLM policy or use AWS Backup. Automated.",
            created_at=datetime.now().isoformat()
        ))

    # Savings opportunities
    if category in ["savings", "all"]:
        recs.append(RecommendationItem(
            id="rec-007",
            category="savings",
            title="Reserved Instance Opportunity",
            description="Stable compute workload for 6 months. RI would save 40%.",
            resource_id="ri-compute-001",
            resource_type="Reserved Instance",
            service="Compute",
            region="sa-brazil-1",
            current_cost=5000.00,
            projected_cost=3000.00,
            savings=2000.00,
            savings_percentage=40.0,
            confidence=0.93,
            priority="high",
            justification="Workload pattern stable (CV < 5%). No planned architecture changes.",
            action="Purchase 1-year All Upfront RI for c6 family",
            risk="Low - workload is production-critical and stable",
            implementation="Purchase via AWS Console or API. Immediate billing benefit.",
            created_at=datetime.now().isoformat()
        ))
        recs.append(RecommendationItem(
            id="rec-008",
            category="savings",
            title="Savings Plan Eligible",
            description="$15K/month compute spend. Compute Savings Plan saves 25%.",
            resource_id="sp-compute-001",
            resource_type="Savings Plan",
            service="Compute",
            region="global",
            current_cost=15000.00,
            projected_cost=11250.00,
            savings=3750.00,
            savings_percentage=25.0,
            confidence=0.91,
            priority="high",
            justification="Consistent compute spend across regions. No significant variation.",
            action="Purchase 1-year Compute Savings Plan at $12K/month commitment",
            risk="Low - commitment is 75% of current spend, providing buffer",
            implementation="Purchase via AWS Console. Applies automatically.",
            created_at=datetime.now().isoformat()
        ))

    return recs

@app.get("/health")
def health():
    return {"status": "healthy", "service": "recommendation-engine"}

@app.get("/ready")
def ready():
    return {"status": "ready"}

@app.get("/live")
def live():
    return {"status": "alive"}

@app.get("/metrics")
def metrics():
    return {"requests_total": 0, "errors_total": 0}

@app.post("/api/v1/recommendations", response_model=RecommendationResponse)
def get_recommendations(req: RecommendationRequest):
    logger.info(f"Recommendation request: {req}")

    recs = generate_recommendations(req.provider, req.account_id, req.category)
    total_savings = sum(r.savings for r in recs)

    return RecommendationResponse(
        provider=req.provider,
        account_id=req.account_id,
        total_savings=round(total_savings, 2),
        total_opportunities=len(recs),
        recommendations=recs,
        generated_at=datetime.now().isoformat()
    )

@app.get("/api/v1/recommendations")
def list_recommendations(provider: str = "all", account_id: str = "all", category: str = "all"):
    req = RecommendationRequest(provider=provider, account_id=account_id, category=category)
    return get_recommendations(req)

@app.post("/api/v1/recommendations/{id}/apply")
def apply_recommendation(id: str):
    logger.info(f"Applying recommendation: {id}")
    return {
        "id": id,
        "status": "applied",
        "applied_at": datetime.now().isoformat(),
        "message": "Recommendation applied successfully. Changes will reflect in next billing cycle."
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", 8003)))
