#!/usr/bin/env python3
"""Crochets Scaleway du tenant de qualification : identité, inventaire, nettoyage.

Trois sous-commandes, appelées par tools/qualification/qualify.py :

    identity                      quel compte les identifiants natifs désignent-ils,
                                  d'après l'API (jamais d'après une variable locale)
    variables --live|--plan-only <identity.json>
                                  les variables Terraform propres au tenant, et
                                  l'échéance à attendre avant de scanner
    extra apply|destroy <outputs.json>
                                  ce que Terraform ne sait pas créer (rien ici)
    inventory <tenant.yaml>       ce que le compte contient, famille par famille, et
                                  ce qui, dedans, appartient au tenant (tag / préfixe)
    cleanup <tenant.yaml> [leftovers.json]
                                  supprime ce qui appartient au tenant — chemin de
                                  SECOURS quand `terraform destroy` n'a pas fini

# Pourquoi l'API plutôt qu'une variable

`scw config get default-project-id` dit ce qu'un profil déclare ; il ne dit pas sur
quel compte une clé ouvre réellement. La porte demande donc à l'API à qui la clé
appartient (GET /iam/v1alpha1/api-keys/{access_key} → default_project_id, puis
GET /account/v3/projects/{id} → organization_id), et compare à ce qu'expected.yaml
épingle. Les identifiants sont résolus EXACTEMENT comme le provider Terraform et
`pepin scan --live` le font : variables SCW_* d'abord, puis ~/.config/scw/config.yaml
(profil actif ou SCW_PROFILE). Aucun n'est écrit, affiché ni journalisé.

# Pourquoi onze familles et un delta

`Destroy complete!` dit que Terraform a supprimé ce qu'il connaissait, pas que le
compte est vide. L'inventaire liste donc famille par famille (serveurs, IP, groupes
de sécurité, volumes Instance et Block, snapshots, images, bases managées et leurs
sauvegardes, réseaux privés, VPC, load balancers, passerelles, applications,
politiques et clés IAM, buckets) ce que le compte contient, en marquant ce qui porte
le tag ou le préfixe du tenant. Le runner compare en plus l'inventaire APRÈS destroy
à celui d'AVANT apply : une ressource apparue entre les deux, même sans tag (un volume
racine renommé, une sauvegarde automatique), est un reste.

Les URL sont celles que la CLI scw appelle (relevées avec `scw … -D` le 2026-09-09).
"""
import base64
import datetime as dt
import hashlib
import hmac
import json
import os
import pathlib
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

import yaml

API = "https://api.scaleway.com"


# ─── identifiants natifs ───────────────────────────────────────────────────────

def _config_file():
    p = os.environ.get("SCW_CONFIG_PATH") or os.path.join(
        os.environ.get("XDG_CONFIG_HOME", os.path.expanduser("~/.config")), "scw", "config.yaml")
    try:
        d = yaml.safe_load(open(p)) or {}
    except FileNotFoundError:
        return {}
    profile = os.environ.get("SCW_PROFILE") or d.get("active_profile")
    if profile and profile in (d.get("profiles") or {}):
        merged = dict(d)
        merged.update(d["profiles"][profile] or {})
        return merged
    return d


def creds():
    """access_key, secret_key : l'environnement d'abord, le fichier ensuite."""
    f = _config_file()
    ak = os.environ.get("SCW_ACCESS_KEY") or f.get("access_key")
    sk = os.environ.get("SCW_SECRET_KEY") or f.get("secret_key")
    if not ak or not sk:
        sys.exit("aucun identifiant Scaleway : SCW_ACCESS_KEY/SCW_SECRET_KEY ou ~/.config/scw/config.yaml")
    return ak, sk


def api(method, path, params=None, body=None):
    url = API + path
    if params:
        url += ("&" if "?" in path else "?") + urllib.parse.urlencode(params)
    _, sk = creds()
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("X-Auth-Token", sk)
    req.add_header("Accept", "application/json")
    if data is not None:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")[:300]
        raise RuntimeError(f"{method} {path} → HTTP {e.code} : {detail}") from None


def paged(path, key, params=None, per_page_param="per_page"):
    """Toutes les pages d'un listing (page=1..n, jusqu'à une page incomplète)."""
    out, page = [], 1
    params = dict(params or {})
    while True:
        params.update({"page": page, per_page_param: 100})
        d = api("GET", path, params)
        items = d.get(key) or []
        out.extend(items)
        if len(items) < 100:
            return out
        page += 1


# ─── S3 (SigV4 minimal : ListBuckets, DeleteBucket) ────────────────────────────

def _sign(key, msg):
    return hmac.new(key, msg.encode(), hashlib.sha256).digest()


def s3_request(method, region, bucket=None, ak=None, sk=None):
    """Requête S3 signée SigV4, sans dépendance : GET / (liste) ou DELETE /bucket."""
    ak, sk = (ak, sk) if ak and sk else creds()
    host = f"s3.{region}.scw.cloud" if not bucket else f"{bucket}.s3.{region}.scw.cloud"
    now = dt.datetime.now(dt.timezone.utc)
    amz_date = now.strftime("%Y%m%dT%H%M%SZ")
    date = now.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(b"").hexdigest()
    canonical_headers = f"host:{host}\nx-amz-content-sha256:{payload_hash}\nx-amz-date:{amz_date}\n"
    signed = "host;x-amz-content-sha256;x-amz-date"
    canonical = f"{method}\n/\n\n{canonical_headers}\n{signed}\n{payload_hash}"
    scope = f"{date}/{region}/s3/aws4_request"
    to_sign = f"AWS4-HMAC-SHA256\n{amz_date}\n{scope}\n{hashlib.sha256(canonical.encode()).hexdigest()}"
    k = _sign(_sign(_sign(_sign(("AWS4" + sk).encode(), date), region), "s3"), "aws4_request")
    sig = hmac.new(k, to_sign.encode(), hashlib.sha256).hexdigest()
    auth = f"AWS4-HMAC-SHA256 Credential={ak}/{scope}, SignedHeaders={signed}, Signature={sig}"
    req = urllib.request.Request(f"https://{host}/", method=method)
    req.add_header("Host", host)
    req.add_header("x-amz-date", amz_date)
    req.add_header("x-amz-content-sha256", payload_hash)
    req.add_header("Authorization", auth)
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status, r.read().decode(errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="replace")[:300]


def s3_buckets(region):
    status, body = s3_request("GET", region)
    if status != 200:
        raise RuntimeError(f"ListBuckets {region} → HTTP {status} : {body}")
    import re
    return re.findall(r"<Name>([^<]+)</Name>", body)


# ─── identité ──────────────────────────────────────────────────────────────────

def identity():
    ak, _ = creds()
    key = api("GET", f"/iam/v1alpha1/api-keys/{ak}")
    project = key.get("default_project_id")
    if not project:
        sys.exit("la clé d'API n'a pas de projet par défaut : impossible d'établir le compte")
    p = api("GET", f"/account/v3/projects/{project}")
    bearer = "application" if key.get("application_id") else "user"
    return {
        "project_id": project,
        "project_name": p.get("name"),
        "organization_id": p.get("organization_id"),
        "bearer": bearer,
        # Le principal de politique de bucket (version 2023-04-17) qui désigne
        # l'identité qui qualifie : `user_id:…` ou `application_id:…`. Passé à
        # Terraform par le runner, jamais écrit dans le dépôt.
        "principal": f"{bearer}_id:{key.get('application_id') or key.get('user_id')}",
    }


# ─── inventaire ────────────────────────────────────────────────────────────────

def _owned(item, tag, prefix):
    tags = item.get("tags") or []
    name = item.get("name") or item.get("description") or ""
    return tag in tags or name.startswith(prefix)


def inventory(tenant):
    region, zone = tenant["region"], tenant["zone"]
    tag, prefix = tenant["tag"], tenant["name_prefix"]
    org = identity()["organization_id"]
    fam = {}

    def put(family, items, key_id="id", owned=None):
        owned = owned or (lambda i: _owned(i, tag, prefix))
        fam[family] = {
            "all": sorted(i.get(key_id) or i.get("name") for i in items),
            "tenant": [{"id": i.get(key_id), "name": i.get("name") or i.get("description")}
                       for i in items if owned(i)],
        }

    put("servers", paged(f"/instance/v1/zones/{zone}/servers", "servers"))
    put("security_groups", paged(f"/instance/v1/zones/{zone}/security_groups", "security_groups"))
    put("flexible_ips", paged(f"/instance/v1/zones/{zone}/ips", "ips"))
    put("instance_volumes", paged(f"/instance/v1/zones/{zone}/volumes", "volumes"))
    put("instance_snapshots", paged(f"/instance/v1/zones/{zone}/snapshots", "snapshots"))
    put("instance_images", paged(f"/instance/v1/zones/{zone}/images", "images", {"public": "false"}))
    put("block_volumes", paged(f"/block/v1alpha1/zones/{zone}/volumes", "volumes", per_page_param="page_size"))
    put("block_snapshots", paged(f"/block/v1alpha1/zones/{zone}/snapshots", "snapshots", per_page_param="page_size"))
    put("rdb_instances", paged(f"/rdb/v1/regions/{region}/instances", "instances", per_page_param="page_size"))
    backups = paged(f"/rdb/v1/regions/{region}/backups", "database_backups", per_page_param="page_size")
    put("rdb_backups", backups, owned=lambda b: (b.get("instance_name") or "").startswith(prefix))
    put("rdb_snapshots", paged(f"/rdb/v1/regions/{region}/snapshots", "snapshots", per_page_param="page_size"))
    put("private_networks", paged(f"/vpc/v2/regions/{region}/private-networks", "private_networks", per_page_param="page_size"))
    put("vpcs", paged(f"/vpc/v2/regions/{region}/vpcs", "vpcs", per_page_param="page_size"))
    put("load_balancers", paged(f"/lb/v1/zones/{zone}/lbs", "lbs", per_page_param="page_size"))
    put("public_gateways", paged(f"/vpc-gw/v2/zones/{zone}/gateways", "gateways", per_page_param="page_size"))
    apps = paged("/iam/v1alpha1/applications", "applications", {"organization_id": org}, per_page_param="page_size")
    put("iam_applications", apps)
    put("iam_policies", paged("/iam/v1alpha1/policies", "policies", {"organization_id": org}, per_page_param="page_size"))
    app_ids = {a["id"] for a in apps if _owned(a, tag, prefix)}
    keys = paged("/iam/v1alpha1/api-keys", "api_keys", {"organization_id": org}, per_page_param="page_size")
    put("iam_api_keys", keys, key_id="access_key",
        owned=lambda k: k.get("application_id") in app_ids or (k.get("description") or "").startswith(prefix))
    buckets = [{"id": b, "name": b} for b in s3_buckets(region)]
    put("buckets", buckets, owned=lambda b: b["name"].startswith(prefix))
    return fam


# ─── nettoyage de secours ──────────────────────────────────────────────────────

def scw(*args):
    """La CLI scw : elle emploie encore la route v1 pour ce que la v2 refuse."""
    cmd = ["scw", *args]
    r = subprocess.run(cmd, capture_output=True, text=True)
    ok = r.returncode == 0
    print(("  ✔ " if ok else "  ✘ ") + " ".join(args) + ("" if ok else f"\n    {r.stderr.strip()[:200]}"))
    return ok


def cleanup(tenant):
    region, zone = tenant["region"], tenant["zone"]
    inv = inventory(tenant)
    rest = {f: v["tenant"] for f, v in inv.items() if v["tenant"]}
    if not rest:
        print("rien à nettoyer : aucune ressource du tenant")
        return True
    ok = True
    # L'ordre est celui des dépendances : ce qui est attaché avant ce qui porte.
    for s in rest.get("servers", []):
        # with-ip=true : sans lui, l'adresse survit au serveur et reste facturée.
        ok &= scw("instance", "server", "delete", s["id"], f"zone={zone}",
                  "with-volumes=all", "with-ip=true", "force-shutdown=true")
    for ip in rest.get("flexible_ips", []):
        ok &= scw("instance", "ip", "delete", ip["id"], f"zone={zone}")
    for v in rest.get("instance_volumes", []):
        ok &= scw("instance", "volume", "delete", v["id"], f"zone={zone}")
    for v in rest.get("block_volumes", []):
        ok &= scw("block", "volume", "delete", v["id"], f"zone={zone}")
    for sg in rest.get("security_groups", []):
        ok &= scw("instance", "security-group", "delete", sg["id"], f"zone={zone}")
    for db in rest.get("rdb_instances", []):
        ok &= scw("rdb", "instance", "delete", db["id"], f"region={region}")
    for b in rest.get("rdb_backups", []):
        ok &= scw("rdb", "backup", "delete", b["id"], f"region={region}")
    for pn in rest.get("private_networks", []):
        ok &= scw("vpc", "private-network", "delete", pn["id"], f"region={region}")
    for v in rest.get("vpcs", []):
        ok &= scw("vpc", "vpc", "delete", v["id"], f"region={region}")
    for b in rest.get("buckets", []):
        status, body = s3_request("DELETE", region, bucket=b["name"])
        good = status in (204, 404)
        ok &= good
        print(("  ✔ " if good else "  ✘ ") + f"DeleteBucket {b['name']} → {status}")
    for k in rest.get("iam_api_keys", []):
        ok &= scw("iam", "api-key", "delete", k["id"])
    for p in rest.get("iam_policies", []):
        ok &= scw("iam", "policy", "delete", p["id"])
    for a in rest.get("iam_applications", []):
        ok &= scw("iam", "application", "delete", a["id"])
    return ok


# ─── variables propres au tenant ───────────────────────────────────────────────

def variables(mode, identity_path):
    """Les variables Terraform que ce tenant calcule à chaque run, jamais écrites :
    les deux échéances de clé d'API, le principal de politique de bucket (l'identité
    qui qualifie, lue dans identity.json), et un mot de passe jetable pour les bases
    managées du plan complet. Le runner les passe à Terraform par l'environnement.

    La clé « expirée » : une échéance que l'apply précède et que le scan suit — le
    runner attend `wait_until`. En mode plan seul, rien n'est appliqué et le scan du
    plan a lieu tout de suite : l'échéance est posée dans le passé."""
    import datetime as dt
    import secrets
    who = json.load(open(identity_path))
    now = dt.datetime.now(dt.timezone.utc)
    rfc = "%Y-%m-%dT%H:%M:%SZ"
    expired = now + (dt.timedelta(minutes=3) if mode == "--live" else -dt.timedelta(minutes=1))
    return {
        "tf_vars": {
            "key_expires_at": (now + dt.timedelta(days=2)).strftime(rfc),
            "key_expired_at": expired.strftime(rfc),
            "bucket_policy_principal": who["principal"],
            "rdb_password": secrets.token_urlsafe(24) + "Aa1!",
        },
        "wait_until": expired.strftime(rfc) if mode == "--live" else None,
        "note": f"4 variables ; clé expirée à {expired.strftime('%H:%M:%SZ')}",
    }


# ─── point d'entrée ────────────────────────────────────────────────────────────

def main(argv):
    if len(argv) < 2 or argv[1] not in ("identity", "inventory", "cleanup", "variables", "extra"):
        sys.exit(__doc__)
    if argv[1] == "identity":
        print(json.dumps(identity(), indent=2))
        return 0
    if argv[1] == "variables":
        print(json.dumps(variables(argv[2], argv[3])))
        return 0
    if argv[1] == "extra":
        # Tout ce que ce tenant crée passe par Terraform : rien hors Terraform.
        print(json.dumps({"created": [], "deleted": [], "left": []}))
        return 0
    tenant = yaml.safe_load(open(argv[2]))
    if argv[1] == "inventory":
        print(json.dumps(inventory(tenant), indent=2, sort_keys=True))
        return 0
    return 0 if cleanup(tenant) else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
