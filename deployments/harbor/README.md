# Harbor Private Registry for Dojo Platform

Private OCI registry for air-gapped and enterprise deployments. Stores skills, workflows, and plugins with Cosign signature verification.

## Quick Start

```bash
docker compose up -d
```

Default: `https://localhost:8443` with admin/Harbor12345.

## Configure Dojo Platform

In `gateway-config.yaml`:

```yaml
marketplace:
  registry: harbor.local:8443/dojo-skills
  trust_minimum: 0
  verify_signatures: true
```

## Push Skills

```bash
dojo skill publish ./my-skill
# Skill stored in local CAS, then:
# oras push harbor.local:8443/dojo-skills/my-skill:1.0.0 ...
```

## Production Checklist

- [ ] Change admin password
- [ ] Configure TLS certificates
- [ ] Set up external PostgreSQL (not the bundled one)
- [ ] Enable vulnerability scanning
- [ ] Configure backup for `/data` volume
- [ ] Set up OIDC authentication (optional)
