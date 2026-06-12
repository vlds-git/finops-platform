#!/usr/bin/env python3
import os

print("📦 Aplicando ZIP unzip + USD→BRL no ingestion-service...")

def read_file(path):
    try:
        with open(path, "r") as f:
            return f.read()
    except FileNotFoundError:
        return None

def write_file(path, content):
    with open(path, "w") as f:
        f.write(content)

content = read_file("backend/ingestion-service/main.go")
if not content:
    print("   ⚠️  backend/ingestion-service/main.go não encontrado")
    exit(1)

# 1. Adicionar import archive/zip
if "archive/zip" not in content:
    content = content.replace('"strings"', '"archive/zip"\n\t"strings"')

# 2. Adicionar import strconv
if "strconv" not in content:
    content = content.replace('"strings"', '"strconv"\n\t"strings"')

# 3. Adicionar EffectiveCostBRL no FocusRecord
if "EffectiveCostBRL" not in content:
    content = content.replace('EffectiveCost      float64           `json:"effective_cost"`', 'EffectiveCost      float64           `json:"effective_cost"`\n\tEffectiveCostBRL   float64           `json:"effective_cost_brl"`')

# 4. Adicionar variável usdToBRLRate
if "usdToBRLRate" not in content:
    init_code = """

var usdToBRLRate float64

func init() {
    rateStr := getEnv("USD_TO_BRL_RATE", "5.15")
    if r, err := strconv.ParseFloat(rateStr, 64); err == nil {
        usdToBRLRate = r
    } else {
        usdToBRLRate = 5.15
    }
}
"""
    content = content.replace("func main() {", init_code + "\nfunc main() {")

# 5. Modificar parseFile para incluir .zip
if ".zip" not in content:
    content = content.replace('if ext == ".csv" {', 'if ext == ".zip" {\n\t\trecords = s.parseZIP(filename, provider)\n\t} else if ext == ".csv" {')

# 6. Adicionar EffectiveCostBRL no processIngestion
if "usdToBRLRate" not in content or "EffectiveCostBRL" not in content:
    content = content.replace('rec.BusinessUnit = rec.Tags["business_unit"]\n\t\t\t\n\t\t\t\tdata, _ := json.Marshal(rec)', 'rec.BusinessUnit = rec.Tags["business_unit"]\n\t\t\t\trec.EffectiveCostBRL = rec.EffectiveCost * usdToBRLRate\n\t\t\t\t\n\t\t\t\tdata, _ := json.Marshal(rec)')

# 7. Adicionar usd_to_brl no statusHandler
if "usd_to_brl" not in content:
    content = content.replace('"checkpoints": len(s.checkpoints),\n\t\t"timestamp":   time.Now().UTC(),', '"checkpoints": len(s.checkpoints),\n\t\t"usd_to_brl":  usdToBRLRate,\n\t\t"timestamp":   time.Now().UTC(),')

write_file("backend/ingestion-service/main.go", content)
print("   ✅ ingestion-service/main.go patchado (ZIP + USD→BRL)")

print("")
print("================================================================================")
print("✅ INGESTION SERVICE PATCHADO!")
print("================================================================================")
print("")
