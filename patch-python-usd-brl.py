#!/usr/bin/env python3
import os

print("📦 Aplicando USD→BRL nos serviços Python...")

for service in ["ml/forecast-engine", "ml/anomaly-detection", "ml/recommendation-engine"]:
    main_file = f"{service}/main.py"
    if os.path.exists(main_file):
        print(f"   → Patchando {main_file}...")
        
        with open(main_file, "r") as f:
            content = f.read()
        
        # Adicionar import os se não existir
        if "import os" not in content:
            content = content.replace("import ", "import os\nimport ", 1)
        
        # Adicionar USD_TO_BRL_RATE
        if "USD_TO_BRL_RATE" not in content:
            content = content.replace(
                "app = FastAPI",
                "USD_TO_BRL_RATE = float(os.getenv(\"USD_TO_BRL_RATE\", \"5.15\"))\n\napp = FastAPI"
            )
        
        with open(main_file, "w") as f:
            f.write(content)
        
        print(f"   ✅ {main_file} patchado")
    else:
        print(f"   ⚠️  {main_file} não encontrado")

print("")
print("================================================================================")
print("✅ SERVIÇOS PYTHON PATCHADOS!")
print("================================================================================")
print("")
