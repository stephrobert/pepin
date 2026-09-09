#!/usr/bin/env python3
"""Crochets Exoscale du tenant de qualification : identité, variables, extra,
inventaire, nettoyage.

    identity                      quelle organisation les identifiants natifs ouvrent-ils,
                                  d'après l'API, jamais d'après un profil local
    variables --live|--plan-only <identity.json>
                                  les variables Terraform propres au tenant (aucune ici)
    extra apply|destroy <outputs.json>
                                  les buckets SOS : le provider n'a pas de ressource de bucket
    inventory <tenant.yaml>       ce que l'organisation contient, famille par famille, dans
                                  chaque zone que tenant.yaml déclare, et ce qui, dedans,
                                  appartient au tenant
    cleanup <tenant.yaml> [leftovers.json]
                                  supprime ce qui appartient au tenant — chemin de SECOURS

# L'API v2 directement, signée comme le collecteur la signe

Identifiants résolus EXACTEMENT comme `pepin scan --live` et le provider Terraform :
EXOSCALE_API_KEY / EXOSCALE_API_SECRET d'abord, puis ~/.config/exoscale/exoscale.toml
(compte EXOSCALE_ACCOUNT ou `defaultaccount` ; EXOSCALE_CONFIG surcharge le chemin).
Chaque appel est signé EXO2-HMAC-SHA256 — réplique de internal/collect/auth.go, elle-même
réplique d'egoscale/v3 : « METHOD /path \\n body \\n valeurs des paramètres signés \\n
en-têtes (aucun) \\n expiration ». Aucun identifiant n'est écrit, affiché ni journalisé.

# L'identité, telle que l'API la donne à CETTE clé

`GET /organization` est refusé à une clé dont le rôle ne couvre pas l'organisation
(« Forbidden by role policy for organization »), et une clé de qualification n'a
aucune raison de le couvrir. Ce que l'API expose à toute clé, c'est SA PROPRE fiche :
`GET /api-key/{key}` rend `org-id`, le nom de la clé et son rôle. C'est cet `org-id`
que l'environnement doit confirmer (`PEPIN_QUAL_EXO_ORG`), et c'est lui qui suffixe
les buckets SOS, globaux.

# SOS, par la CLI de référence et par S3

`exo` (egoscale v3) est la CLI de référence : elle crée et supprime les buckets
(`exo storage mb|rb|setacl|bucket versioning`) et sert aux listings qu'un mainteneur
rejoue. Elle ne sait pas créer un bucket VERROUILLÉ (Object Lock) : le contre-exemple
`hardened` passe par l'API S3 de SOS (CreateBucket + x-amz-bucket-object-lock-enabled,
SigV4, sos-<zone>.exo.io), avec la même clé — SOS l'accepte, et active le versioning
avec.

Mesuré ici, et à savoir avant de lire un verdict d'étiquetage sur un bucket SOS :
SOS ACCEPTE PutBucketTagging (200, par SigV4 comme par la CLI aws) et NE PERSISTE PAS
les tags — GetBucketTagging rend NoSuchTagSet juste après. Un bucket SOS n'a donc
jamais d'étiquette, quoi qu'on fasse ; les trois buckets du tenant portent
`governance_resource_required_tags` par construction, et expected.yaml le dit.

# Ce qu'un inventaire couvre

Instances, réseaux privés, volumes et snapshots block storage, clusters SKS, IP
élastiques, pools d'instances, NLB, modèles privés, clés SSH, groupes d'anti-affinité
sont listés dans CHAQUE zone de `zones:` ; les groupes de sécurité, rôles IAM et clés
d'API sont d'organisation (l'API les rend identiques dans toute zone : dédoublonnés
par identifiant) ; les buckets SOS par `exo storage list`, toutes zones.
"""
import base64
import datetime as dt
import hashlib
import hmac
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

import yaml

# ─── identifiants natifs ───────────────────────────────────────────────────────

def try_creds():
    key = os.environ.get("EXOSCALE_API_KEY")
    secret = os.environ.get("EXOSCALE_API_SECRET")
    zone = os.environ.get("EXOSCALE_ZONE")
    if not (key and secret):
        import tomllib
        path = os.environ.get("EXOSCALE_CONFIG") or os.path.expanduser("~/.config/exoscale/exoscale.toml")
        try:
            conf = tomllib.load(open(path, "rb"))
        except FileNotFoundError:
            return None
        want = os.environ.get("EXOSCALE_ACCOUNT") or conf.get("defaultaccount") or ""
        for a in conf.get("accounts") or []:
            if not want or a.get("name") == want:
                key, secret = key or a.get("key"), secret or a.get("secret")
                zone = zone or a.get("defaultZone")
                break
    if not (key and secret):
        return None
    return key, secret, zone or "ch-gva-2"


def creds():
    c = try_creds()
    if c is None:
        sys.exit("aucun identifiant Exoscale : EXOSCALE_API_KEY/EXOSCALE_API_SECRET ou ~/.config/exoscale/exoscale.toml")
    return c


def api(method, zone, path, params=None, body=None):
    """Un appel API v2 signé EXO2-HMAC-SHA256 (api-<zone>.exoscale.com/v2)."""
    key, secret, _ = creds()
    url = f"https://api-{zone}.exoscale.com/v2{path}"
    names, values = [], []
    if params:
        for k in sorted(params):
            names.append(k)
            values.append(str(params[k]))
        url += "?" + urllib.parse.urlencode(params)
    data = json.dumps(body).encode() if body is not None else None
    expires = int(time.time()) + 600
    sig_parts = [f"{method} /v2{path}", data.decode() if data else "", "".join(values), "", str(expires)]
    mac = hmac.new(secret.encode(), "\n".join(sig_parts).encode(), hashlib.sha256).digest()
    header = [f"EXO2-HMAC-SHA256 credential={key}"]
    if names:
        header.append("signed-query-args=" + ";".join(names))
    header += [f"expires={expires}", "signature=" + base64.b64encode(mac).decode()]
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", ",".join(header))
    req.add_header("Accept", "application/json")
    if data is not None:
        req.add_header("Content-Type", "application/json")
    # Mesuré : GET /private-network sur api-de-fra-1 reste parfois sans réponse
    # (délai dépassé, une fois sur trois ou quatre), puis répond en 0,2 s. Un listing
    # de preuve ne doit pas rendre « inconnu » sur un aléa : trois tentatives.
    for attempt in range(3):
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                raw = r.read()
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as e:
            detail = e.read().decode(errors="replace")[:300]
            raise RuntimeError(f"{method} {zone} {path} → HTTP {e.code} : {detail}") from None
        except (TimeoutError, urllib.error.URLError) as e:
            if attempt == 2:
                raise RuntimeError(f"{method} {zone} {path} : sans réponse après 3 tentatives ({e})") from None
            time.sleep(2)


def exo(*args):
    """La CLI de référence, sortie JSON. Ses identifiants viennent de son propre fichier."""
    r = subprocess.run(["exo", *args, "-O", "json"], capture_output=True, text=True)
    if r.returncode != 0:
        raise RuntimeError(f"exo {' '.join(args[:3])} : {(r.stderr or r.stdout).strip()[:300]}")
    try:
        return json.loads(r.stdout or "null")
    except json.JSONDecodeError:
        return None


# ─── S3 (SigV4 minimal : CreateBucket verrouillé, sondes) ──────────────────────

def _sign(key, msg):
    return hmac.new(key, msg.encode(), hashlib.sha256).digest()


def sos_request(method, zone, bucket, query, body=b"", headers=None):
    """Requête S3 signée SigV4 vers SOS (path-style, région = zone)."""
    ak, sk, _ = creds()
    host = f"sos-{zone}.exo.io"
    now = dt.datetime.now(dt.timezone.utc)
    amz_date, date = now.strftime("%Y%m%dT%H%M%SZ"), now.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(body).hexdigest()
    extra = dict(sorted((k.lower(), v) for k, v in (headers or {}).items()))
    all_headers = dict(extra, **{"host": host, "x-amz-content-sha256": payload_hash, "x-amz-date": amz_date})
    signed = ";".join(sorted(all_headers))
    canonical_headers = "".join(f"{k}:{all_headers[k]}\n" for k in sorted(all_headers))
    canonical = f"{method}\n/{bucket}\n{query}\n{canonical_headers}\n{signed}\n{payload_hash}"
    scope = f"{date}/{zone}/s3/aws4_request"
    to_sign = f"AWS4-HMAC-SHA256\n{amz_date}\n{scope}\n{hashlib.sha256(canonical.encode()).hexdigest()}"
    k = _sign(_sign(_sign(_sign(("AWS4" + sk).encode(), date), zone), "s3"), "aws4_request")
    sig = hmac.new(k, to_sign.encode(), hashlib.sha256).hexdigest()
    url = f"https://{host}/{bucket}" + (f"?{query}" if query else "")
    req = urllib.request.Request(url, data=body or None, method=method)
    for k2, v in all_headers.items():
        req.add_header(k2, v)
    req.add_header("Authorization", f"AWS4-HMAC-SHA256 Credential={ak}/{scope}, SignedHeaders={signed}, Signature={sig}")
    if body:
        req.add_header("Content-Type", "application/xml")
        req.add_header("Content-MD5", base64.b64encode(hashlib.md5(body).digest()).decode())  # nosec B324 — exigé par S3, pas cryptographique
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status, r.read().decode(errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="replace")[:300]


def create_locked_bucket(zone, bucket):
    """CreateBucket avec Object Lock dès la création (la seule façon de l'obtenir :
    PutObjectLockConfiguration exige un bucket créé verrouillé). SOS répond 200 et
    active le versioning avec."""
    status, body = sos_request("PUT", zone, bucket, "", b"", headers={"x-amz-bucket-object-lock-enabled": "true"})
    if status not in (200, 204):
        raise RuntimeError(f"CreateBucket(object-lock) {bucket} → HTTP {status} : {body}")


# ─── identité ──────────────────────────────────────────────────────────────────

def identity():
    key, _, zone = creds()
    me = api("GET", zone, f"/api-key/{key}")
    role = next((r.get("name") for r in api("GET", zone, "/iam-role").get("iam-roles") or [] if r.get("id") == me.get("role-id")), None)
    return {"organization_id": me.get("org-id"), "api_key_name": me.get("name"), "role": role,
            "label": f"clé « {me.get('name')} », rôle « {role} »"}


# ─── variables propres au tenant ───────────────────────────────────────────────

def variables(mode, identity_path):
    # `organization_id` est posée par le runner lui-même (TF_VAR_organization_id =
    # l'attendu confirmé par l'API). Rien d'autre n'est propre à ce tenant, et rien
    # n'est à attendre avant le scan. Les sources de données (modèles) lisent l'API
    # au plan : le plan seul exige donc les mêmes identifiants que le live.
    #
    # Le provider Terraform ne lit PAS ~/.config/exoscale/exoscale.toml (« missing or
    # incomplete API credentials ») : il attend EXOSCALE_API_KEY / EXOSCALE_API_SECRET.
    # Le crochet les lui passe par l'environnement du run — le même que `pepin scan`
    # reçoit —, sans les écrire ni les journaliser, et seulement s'ils n'y sont pas déjà.
    env = {}
    if not (os.environ.get("EXOSCALE_API_KEY") and os.environ.get("EXOSCALE_API_SECRET")):
        key, secret, zone = creds()
        env = {"EXOSCALE_API_KEY": key, "EXOSCALE_API_SECRET": secret, "EXOSCALE_ZONE": zone}
    note = "aucune variable propre ; identifiants du fichier exoscale.toml passés au provider par l'environnement"
    if mode == "--live":
        note += " ; " + wait_for_quota()
    return {"tf_vars": {}, "env": env, "wait_until": None, "note": note}


def wait_for_quota(max_seconds=900):
    """Le QUOTA d'instances de l'organisation (GET /quota : `instance`, 4 ici) compte
    encore, plusieurs minutes après leur suppression, des instances que plus aucun
    listing ne montre. Mesuré : usage=2 avec 0 instance listée, 5 min après un destroy
    réussi — et l'apply suivant refusé (« Usage of resource 'instance' has been
    exceeded »). Un run qui part trop tôt échoue donc à l'apply : on attend que le
    compteur retombe à ce que les listings montrent, borné."""
    _, _, zone = creds()
    t0 = time.time()
    while True:
        usage = next((x["usage"] for x in api("GET", zone, "/quota")["quotas"] if x["resource"] == "instance"), 0)
        if usage == 0 or time.time() - t0 > max_seconds:
            waited = int(time.time() - t0)
            return f"quota instance : usage={usage} après {waited}s d'attente" if waited else "quota instance libre"
        print(f"quota instance : usage={usage}, 0 instance listée ; attente…", file=sys.stderr)
        time.sleep(30)


# ─── hors Terraform : buckets SOS ──────────────────────────────────────────────

def _buckets(outputs):
    return {"public": outputs["bucket_public"], "unversioned": outputs["bucket_unversioned"], "hardened": outputs["bucket_hardened"]}


def extra_apply(outputs):
    """Trois buckets dans la zone scannée : `public` (ACL public-read, versionné, sans
    verrou) → CLD-STO-1 ; `unversioned` (jamais versionné, sans verrou) → CLD-STO-4 et
    CLD-STO-8 ; `hardened` (privé, Object Lock, donc versionné) = contre-exemple.
    Aucun n'est étiqueté : SOS ne persiste pas les tags de bucket (cf. en-tête)."""
    zone = outputs["zone"]
    b = _buckets(outputs)
    created = []
    exo("storage", "mb", f"sos://{b['public']}", "--zone", zone, "--acl", "public-read")
    created.append(b["public"])
    # `--zone` : sans elle, la CLI adresse la zone par défaut de son profil et SOS répond
    # 301 PermanentRedirect (mesuré au run 2, bucket en de-fra-1, profil en ch-gva-2).
    exo("storage", "bucket", "versioning", "enable", f"sos://{b['public']}", "--zone", zone)
    exo("storage", "mb", f"sos://{b['unversioned']}", "--zone", zone, "--acl", "private")
    created.append(b["unversioned"])
    create_locked_bucket(zone, b["hardened"])
    created.append(b["hardened"])
    return {"created": created}


def _rb(name):
    r = subprocess.run(["exo", "storage", "rb", f"sos://{name}", "-r", "-f"], capture_output=True, text=True)
    out = (r.stderr + r.stdout).lower()
    return r.returncode == 0 or "nosuchbucket" in out or "not found" in out, (r.stderr or r.stdout).strip()[:120]


def extra_destroy(outputs):
    deleted, left = [], []
    for name in _buckets(outputs).values():
        ok, msg = _rb(name)
        (deleted if ok else left).append(name if ok else f"{name} ({msg})")
    return {"deleted": deleted, "left": left}


# ─── inventaire ────────────────────────────────────────────────────────────────

ZONAL = [("instances", "/instance", "instances"), ("private_networks", "/private-network", "private-networks"),
         ("block_volumes", "/block-storage", "block-storage-volumes"), ("block_snapshots", "/block-storage-snapshot", "block-storage-snapshots"),
         ("sks_clusters", "/sks-cluster", "sks-clusters"), ("elastic_ips", "/elastic-ip", "elastic-ips"),
         ("instance_pools", "/instance-pool", "instance-pools"), ("load_balancers", "/load-balancer", "load-balancers"),
         ("templates", "/template", "templates"), ("ssh_keys", "/ssh-key", "ssh-keys"),
         ("anti_affinity_groups", "/anti-affinity-group", "anti-affinity-groups"),
         ("security_groups", "/security-group", "security-groups")]
GLOBAL = [("iam_roles", "/iam-role", "iam-roles"), ("api_keys", "/api-key", "api-keys")]


def _owned(item, tag, prefix):
    return tag in (item.get("labels") or {}) or str(item.get("name") or "").startswith(prefix)


def inventory(tenant):
    tag, prefix = tenant["tag"], tenant["name_prefix"]
    zones = tenant.get("zones") or [tenant["zone"]]
    fam = {}

    def put(family, items):
        fam[family] = {"all": sorted(items),
                       "tenant": [{"id": i, "name": v.get("name"), "zone": v.get("zone")} for i, v in sorted(items.items()) if _owned(v, tag, prefix)]}

    for family, path, key in ZONAL:
        items = {}
        for z in zones:
            params = {"visibility": "private"} if family == "templates" else None
            for i in (api("GET", z, path, params).get(key) or []):
                iid = str(i.get("id") or i.get("name"))
                items.setdefault(iid, dict(i, zone=z))   # même id dans deux zones = ressource d'organisation
        put(family, items)
    _, _, zone = creds()
    for family, path, key in GLOBAL:
        put(family, {str(i.get("id") or i.get("key")): dict(i, zone=None) for i in (api("GET", zone, path).get(key) or [])})
    buckets = exo("storage", "list") or []
    put("buckets", {b["name"]: {"name": b["name"], "zone": b.get("zone")} for b in buckets if isinstance(b, dict) and b.get("name")})
    return fam


# ─── nettoyage de secours ──────────────────────────────────────────────────────

ORDER = [("instances", "/instance"), ("sks_clusters", "/sks-cluster"), ("block_snapshots", "/block-storage-snapshot"),
         ("block_volumes", "/block-storage"), ("elastic_ips", "/elastic-ip"), ("private_networks", "/private-network"),
         ("security_groups", "/security-group"), ("templates", "/template"), ("ssh_keys", "/ssh-key"),
         ("anti_affinity_groups", "/anti-affinity-group"), ("iam_roles", "/iam-role"), ("api_keys", "/api-key")]


def cleanup(tenant, leftovers_path=None):
    zones = tenant.get("zones") or [tenant["zone"]]
    inv = inventory(tenant)
    rest = {f: {x["id"]: x.get("zone") for x in v["tenant"]} for f, v in inv.items() if v["tenant"]}
    if leftovers_path:
        for f, ids in (json.load(open(leftovers_path)).get("delta") or {}).items():
            for i in ids:
                rest.setdefault(f, {}).setdefault(i, None)
    if not any(rest.values()):
        print("rien à nettoyer : aucune ressource du tenant")
        return True
    ok = True

    def delete(family, path, iid, zone):
        nonlocal ok
        last = None
        for z in ([zone] if zone else zones):
            try:
                api("DELETE", z, f"{path}/{iid}")
                print(f"  ✔ DELETE {family} {iid} ({z})")
                return
            except RuntimeError as e:
                last = e
                if "404" not in str(e):
                    break
        ok = False
        print(f"  ✘ DELETE {family} {iid} : {last}")

    for family, path in ORDER:
        for iid, zone in (rest.get(family) or {}).items():
            delete(family, path, iid, zone)
        if family == "instances" and rest.get("instances"):
            time.sleep(30)   # les volumes ne se suppriment qu'une fois l'instance partie
    for name in rest.get("buckets") or {}:
        done, msg = _rb(name)
        ok &= done
        print(("  ✔ " if done else "  ✘ ") + f"exo storage rb {name}" + ("" if done else f" : {msg}"))
    # Mesuré : SKS crée une clé d'API `sks-ccm-<cluster>` pour le CCM de chaque cluster
    # et la supprime d'ordinaire avec lui ; une fois, elle a survécu au cluster (rôle
    # supprimé, clé listée), et DELETE /api-key répond 403 « API Key not in
    # organization » — gérée par la plateforme. On tente, on dit, on ne bloque pas.
    roles = {r["id"] for r in api("GET", zones[0], "/iam-role").get("iam-roles") or []}
    for k in api("GET", zones[0], "/api-key").get("api-keys") or []:
        if str(k.get("name", "")).startswith("sks-ccm-") and k.get("role-id") not in roles:
            try:
                api("DELETE", zones[0], f"/api-key/{k['key']}")
                print(f"  ✔ DELETE api-key orpheline {k['name']}")
            except RuntimeError as e:
                print(f"  ⚠ clé d'API orpheline {k['name']} (rôle supprimé) : {e} — à signaler à Exoscale")
    return ok


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
        outputs = json.load(open(argv[3]))
        print(json.dumps(extra_apply(outputs) if argv[2] == "apply" else extra_destroy(outputs)))
        return 0
    tenant = yaml.safe_load(open(argv[2]))
    if argv[1] == "inventory":
        print(json.dumps(inventory(tenant), indent=2, sort_keys=True))
        return 0
    return 0 if cleanup(tenant, argv[3] if len(argv) > 3 else None) else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
