#!/usr/bin/env python3
"""Generate a flattened compose file for podman-compose.

podman-compose doesn't support the 'include' directive with path objects,
so we merge all compose files into a single file.

Usage: python3 scripts/gen-podman-compose.py [output-file]
"""

import sys
from pathlib import Path

import yaml


def main():
    project_root = Path(__file__).parent.parent
    output_file = (
        sys.argv[1] if len(sys.argv) > 1 else project_root / "docker-compose.podman.yml"
    )

    compose_files = [
        "infra/docker/infrastructure.yml",
        "infra/docker/services-api.yml",
        "infra/docker/services-mcp.yml",
        "infra/docker/services-memory.yml",
        "infra/docker/services-realtime.yml",
        "infra/docker/apps-platform.yml",
        "infra/docker/apps-web.yml",
        "infra/docker/inference.yml",
    ]

    merged = {
        "volumes": {},
        "networks": {},
        "services": {},
    }

    for cf in compose_files:
        path = project_root / cf
        if not path.exists():
            print(f"Warning: {cf} not found, skipping", file=sys.stderr)
            continue

        with open(path) as f:
            data = yaml.safe_load(f) or {}

        if "volumes" in data and data["volumes"]:
            merged["volumes"].update(data["volumes"])

        if "networks" in data and data["networks"]:
            merged["networks"].update(data["networks"])

        if "services" in data and data["services"]:
            # Fix relative paths - original files are in infra/docker/
            for svc_name, svc_data in data["services"].items():
                if svc_data and "env_file" in svc_data:
                    fixed_env_files = []
                    for ef in svc_data["env_file"]:
                        # Fix ../../.env -> .env (now relative to project root)
                        if isinstance(ef, str) and "../../.env" in ef:
                            fixed_env_files.append(ef.replace("../../.env", ".env"))
                        else:
                            fixed_env_files.append(ef)
                    svc_data["env_file"] = fixed_env_files

                # Fix bind mount paths - adjust relative paths for project root
                if svc_data and "volumes" in svc_data:
                    fixed_volumes = []
                    cf_dir = str(Path(cf).parent)  # e.g., "infra/docker"
                    for vol in svc_data["volumes"]:
                        if isinstance(vol, str) and ":" in vol:
                            # This is a bind mount like ./path:/container/path:ro
                            parts = vol.split(":")
                            host_path = parts[0]
                            if host_path.startswith("./"):
                                # Fix relative path: ./init-db -> ./infra/docker/init-db
                                parts[0] = host_path.replace("./", f"./{cf_dir}/", 1)
                                fixed_volumes.append(":".join(parts))
                            elif host_path.startswith("../../"):
                                # Fix ../../path -> ./path (relative to project root)
                                parts[0] = host_path.replace("../../", "./", 1)
                                fixed_volumes.append(":".join(parts))
                            else:
                                fixed_volumes.append(vol)
                        else:
                            # Named volume or other format - keep as is
                            fixed_volumes.append(vol)
                    svc_data["volumes"] = fixed_volumes

                # Replace build with pre-built image for services using additional_contexts
                # These need to be built separately with: ./scripts/podman-build-services.sh
                if svc_data and "build" in svc_data:
                    build_cfg = svc_data["build"]
                    if (
                        isinstance(build_cfg, dict)
                        and "additional_contexts" in build_cfg
                    ):
                        # Remove build config and use pre-built image instead
                        del svc_data["build"]
                        svc_data["image"] = f"localhost/jan-server-{svc_name}:latest"
                    elif isinstance(build_cfg, dict) and "context" in build_cfg:
                        ctx = build_cfg["context"]
                        if ctx.startswith("../"):
                            # ../../services/x -> services/x
                            build_cfg["context"] = ctx.replace("../../", "")

            merged["services"].update(data["services"])

    # Clean up empty sections
    merged = {k: v for k, v in merged.items() if v}

    # Fix nested variable interpolation that podman-compose doesn't handle well
    # Replace ${VAR:-${NESTED:-default}} patterns with simpler ${VAR:-default}
    def fix_nested_vars(obj):
        if isinstance(obj, str):
            # Replace common nested patterns with direct defaults
            replacements = {
                "${POSTGRES_USER:-jan_user}": "jan_user",
                "${POSTGRES_PASSWORD:-jan_password}": "jan_password",
                "${POSTGRES_DB:-jan_llm_api}": "jan_llm_api",
                "${POSTGRES_HOST:-api-db}": "api-db",
                "${POSTGRES_PORT:-5432}": "5432",
            }
            for old, new in replacements.items():
                obj = obj.replace(old, new)
            return obj
        elif isinstance(obj, dict):
            return {k: fix_nested_vars(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [fix_nested_vars(i) for i in obj]
        return obj

    merged = fix_nested_vars(merged)

    with open(output_file, "w") as f:
        f.write("# Auto-generated Podman-compatible compose file\n")
        f.write("# Generated by: python3 scripts/gen-podman-compose.py\n")
        f.write("# Do not edit manually - regenerate with the script\n\n")
        yaml.dump(
            merged, f, default_flow_style=False, sort_keys=False, allow_unicode=True
        )

    print(f"Generated: {output_file}")
    print(f"  Services: {len(merged.get('services', {}))}")
    print(f"  Volumes:  {len(merged.get('volumes', {}))}")
    print(f"  Networks: {len(merged.get('networks', {}))}")


if __name__ == "__main__":
    main()
