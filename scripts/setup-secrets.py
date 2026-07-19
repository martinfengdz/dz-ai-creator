#!/usr/bin/env python3
"""Generate deployment secrets for dz-ai-creator.
Run BEFORE first deploy:  python3 scripts/setup-secrets.py
"""
import secrets
import base64
import os

secrets_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "secrets")
os.makedirs(secrets_dir, exist_ok=True)

pg_pass = "dz_creator_" + base64.urlsafe_b64encode(secrets.token_bytes(12)).decode().rstrip("=").replace("-", "x")
with open(os.path.join(secrets_dir, "postgres_password"), "w") as f:
    f.write(pg_pass)
print("[✓] postgres_password")

db_url = f"postgres://dz_ai_creator:{pg_pass}@postgres:5432/dz_ai_creator?sslmode=disable"
with open(os.path.join(secrets_dir, "database_url"), "w") as f:
    f.write(db_url)
print("[✓] database_url")

master_key = base64.b64encode(secrets.token_bytes(32)).decode()
with open(os.path.join(secrets_dir, "app_secrets_master_key"), "w") as f:
    f.write(master_key)
print("[✓] app_secrets_master_key")

print(f"\n✅ Secrets generated at {secrets_dir}/")
print("Next: docker compose -f deploy/compose.yaml up -d")
