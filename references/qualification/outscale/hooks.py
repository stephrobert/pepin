#!/usr/bin/env python3
"""Crochets Outscale du tenant de qualification : identité, variables, extra,
inventaire, nettoyage.

    identity                      quel compte les identifiants natifs ouvrent-ils,
                                  d'après l'API (ReadAccounts), jamais d'après un profil
    variables --live|--plan-only <identity.json>
                                  les variables Terraform propres au tenant, et
                                  l'échéance à attendre avant de scanner
    extra apply|destroy <outputs.json>
                                  ce que le provider Terraform ne sait pas créer :
                                  les buckets OOS et la politique EIM INLINE
    inventory <tenant.yaml>       ce que le compte contient, famille par famille, et
                                  ce qui, dedans, appartient au tenant (tag / préfixe)
    cleanup <tenant.yaml> [leftovers.json]
                                  supprime ce qui appartient au tenant, ou ce qui est
                                  apparu pendant le run — chemin de SECOURS

# L'OAPI directement, comme le crochet Scaleway parle à son API

Les identifiants sont résolus EXACTEMENT comme `pepin scan --live` et le provider
Terraform le font : variables OSC_ACCESS_KEY / OSC_SECRET_KEY / OSC_REGION d'abord,
puis ~/.osc/config.json (profil OSC_PROFILE, « default » à défaut ; OSC_CONFIG_FILE
surcharge le chemin). Chaque appel est POST /api/v1/<Call> signé en V4 (service
« oapi »), c'est-à-dire ce que le descripteur providers/outscale.yaml déclare.
Aucun identifiant n'est écrit, affiché ni journalisé — et l'on n'appelle jamais
`octl profile current`, qui imprime la clé secrète en clair.

La CLI de référence du fournisseur est `octl` ; elle sert aux listings lisibles
qu'un mainteneur rejoue à la main (README). Ici, l'API : un crochet qui prouve le
vide ne doit dépendre d'aucun outil dont la surface peut bouger.

# Ce que Terraform ne sait pas exprimer, et que `extra` crée

Le provider outscale/outscale 1.8.0 n'a aucune ressource de bucket OOS, et aucune
ressource de politique EIM INLINE (l'incident fondateur de l'ADR-0006 : une politique
inline `Action: *` échappait à tous les contrôles). Les deux sont créés ici, après
l'apply, et supprimés avant le destroy — OOS par la CLI `aws` (S3 sur
oos.<région>.outscale.com), la politique inline par l'OAPI (PutUserPolicy). Ils
portent le préfixe du tenant, donc la preuve de destruction les relit.
"""
import base64
import datetime as dt
import hashlib
import hmac
import json
import os
import subprocess
import sys
import urllib.error
import urllib.request

import yaml

# ─── identifiants natifs ───────────────────────────────────────────────────────

def creds():
    """(access_key, secret_key, region) : environnement d'abord, fichier ensuite."""
    ak = os.environ.get("OSC_ACCESS_KEY")
    sk = os.environ.get("OSC_SECRET_KEY")
    region = os.environ.get("OSC_REGION")
    if not (ak and sk):
        path = os.environ.get("OSC_CONFIG_FILE") or os.path.expanduser("~/.osc/config.json")
        try:
            d = json.load(open(path))
        except FileNotFoundError:
            sys.exit("aucun identifiant Outscale : OSC_ACCESS_KEY/OSC_SECRET_KEY ou ~/.osc/config.json")
        prof = d.get(os.environ.get("OSC_PROFILE") or "default") or {}
        ak, sk = ak or prof.get("access_key"), sk or prof.get("secret_key")
        region = region or prof.get("region") or prof.get("region_name")
    if not (ak and sk):
        sys.exit("aucun identifiant Outscale utilisable")
    return ak, sk, region or "eu-west-2"


def _sign(key, msg):
    return hmac.new(key, msg.encode(), hashlib.sha256).digest()


def _sigv4(method, host, path, body, service, region, ak, sk, extra_headers=None, query=""):
    now = dt.datetime.now(dt.timezone.utc)
    amz_date, date = now.strftime("%Y%m%dT%H%M%SZ"), now.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(body).hexdigest()
    headers = {"host": host, "x-amz-content-sha256": payload_hash, "x-amz-date": amz_date}
    headers.update({k.lower(): v for k, v in (extra_headers or {}).items()})
    signed = ";".join(sorted(headers))
    canonical_headers = "".join(f"{k}:{headers[k].strip()}\n" for k in sorted(headers))
    canonical = f"{method}\n{path}\n{query}\n{canonical_headers}\n{signed}\n{payload_hash}"
    scope = f"{date}/{region}/{service}/aws4_request"
    to_sign = f"AWS4-HMAC-SHA256\n{amz_date}\n{scope}\n{hashlib.sha256(canonical.encode()).hexdigest()}"
    k = _sign(_sign(_sign(_sign(("AWS4" + sk).encode(), date), region), service), "aws4_request")
    sig = hmac.new(k, to_sign.encode(), hashlib.sha256).hexdigest()
    headers["authorization"] = f"AWS4-HMAC-SHA256 Credential={ak}/{scope}, SignedHeaders={signed}, Signature={sig}"
    return headers


def oapi(call, body=None):
    """POST /api/v1/<Call>, signé V4 (service oapi, région du profil)."""
    ak, sk, region = creds()
    host = f"api.{region}.outscale.com"
    data = json.dumps(body or {}).encode()
    headers = _sigv4("POST", host, f"/api/v1/{call}", data, "oapi", region, ak, sk,
                     {"content-type": "application/json"})
    req = urllib.request.Request(f"https://{host}/api/v1/{call}", data=data, method="POST")
    for k, v in headers.items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")[:400]
        raise RuntimeError(f"{call} → HTTP {e.code} : {detail}") from None


def paged(call, key, body=None, per_page=1000):
    """Toutes les pages d'un listing OAPI (NextPageToken)."""
    out, token = [], None
    body = dict(body or {})
    while True:
        b = dict(body, ResultsPerPage=per_page)
        if token:
            b["NextPageToken"] = token
        d = oapi(call, b)
        out.extend(d.get(key) or [])
        token = d.get("NextPageToken")
        if not token:
            return out


# ─── OOS (S3) par la CLI aws, identifiants passés par l'environnement ─────────

def aws_env():
    ak, sk, region = creds()
    env = dict(os.environ, AWS_ACCESS_KEY_ID=ak, AWS_SECRET_ACCESS_KEY=sk, AWS_DEFAULT_REGION=region,
               AWS_EC2_METADATA_DISABLED="true")
    env.pop("AWS_PROFILE", None)
    env.pop("AWS_SESSION_TOKEN", None)
    return env, f"https://oos.{region}.outscale.com"


def s3(*args, check=True, capture=True):
    env, endpoint = aws_env()
    r = subprocess.run(["aws", "--endpoint-url", endpoint, "s3api", *args], env=env, capture_output=capture, text=True)
    if check and r.returncode != 0:
        raise RuntimeError(f"aws s3api {' '.join(args[:2])} : {r.stderr.strip()[:300]}")
    return r


def s3_buckets():
    r = s3("list-buckets", "--query", "Buckets[].Name", "--output", "json")
    return json.loads(r.stdout or "[]")


# ─── identité ──────────────────────────────────────────────────────────────────

def identity():
    d = oapi("ReadAccounts")
    accts = d.get("Accounts") or []
    if not accts:
        sys.exit("ReadAccounts ne rend aucun compte")
    a = accts[0]
    _, _, region = creds()
    return {"account_id": a.get("AccountId"), "region": region, "label": f"région {region}"}


# ─── variables propres au tenant ───────────────────────────────────────────────

def variables(mode, identity_path):
    """Échéance de la clé conforme (J+2), et rien d'autre : chez Outscale, la clé
    sans échéance se crée (le contrôle CLD-IAM-2 s'exerce en live), la rotation ne
    se construit pas (une date de création ne s'antidate pas)."""
    now = dt.datetime.now(dt.timezone.utc)
    rfc = "%Y-%m-%dT%H:%M:%SZ"
    return {
        "tf_vars": {"key_expires_at": (now + dt.timedelta(days=2)).strftime(rfc)},
        "wait_until": None,
        "note": "1 variable ; rien à attendre",
    }


# ─── hors Terraform : buckets OOS et politique EIM inline ──────────────────────

def _bucket_names(outputs):
    sfx = outputs["bucket_suffix"]
    return {
        "public": f"pepin-qual-{sfx}-public",
        "policy": f"pepin-qual-{sfx}-policy",
        "unversioned": f"pepin-qual-{sfx}-unversioned",
        "unlocked": f"pepin-qual-{sfx}-unlocked",
        "unencrypted": f"pepin-qual-{sfx}-unencrypted",
        "hardened": f"pepin-qual-{sfx}-hardened",
    }


GOV_TAGS = [{"Key": "CostCenter", "Value": "qualification"}, {"Key": "Project", "Value": "pepin"},
            {"Key": "Env", "Value": "qualification"}, {"Key": "Owner", "Value": "pepin-maintainer"},
            {"Key": "pepin-qual", "Value": "tenant"}]


def extra_apply(outputs):
    created = []
    names = _bucket_names(outputs)
    # Six buckets vides. Une faute par bucket ; `hardened` est le contre-exemple de
    # tous : privé, versionné, verrouillé, chiffré (SSE opt-in chez OOS), étiqueté.
    # Verrou d'objets : partout où le versioning existe, sauf `unlocked` (sa faute) ;
    # `unversioned` ne peut pas en avoir (pas de verrou sans versioning : deux écarts
    # structurellement liés). SSE : partout sauf `unencrypted` (sa faute).
    lock = {"public", "policy", "unencrypted", "hardened"}
    for role, name in names.items():
        args = ["create-bucket", "--bucket", name]
        if role in lock:
            args += ["--object-lock-enabled-for-bucket"]
        s3(*args)
        created.append(name)
        s3("put-bucket-tagging", "--bucket", name, "--tagging", json.dumps({"TagSet": GOV_TAGS}))
        # Mesuré : un bucket créé avec Object Lock a son versioning activé d'office et
        # FIGÉ (« InvalidBucketState … the versioning state cannot be changed »). Seul
        # `unlocked` reçoit le versioning explicitement.
        if role == "unlocked":
            s3("put-bucket-versioning", "--bucket", name, "--versioning-configuration", "Status=Enabled")
        if role != "unencrypted":
            s3("put-bucket-encryption", "--bucket", name, "--server-side-encryption-configuration",
               json.dumps({"Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}}]}))
    s3("put-bucket-acl", "--bucket", names["public"], "--acl", "public-read")
    s3("put-bucket-policy", "--bucket", names["policy"], "--policy", json.dumps({
        "Version": "2012-10-17",
        "Statement": [{"Sid": "AnyoneReads", "Effect": "Allow", "Principal": "*",
                       "Action": ["s3:GetObject"], "Resource": [f"arn:aws:s3:::{names['policy']}/*"]}]}))
    # La politique EIM INLINE `Action: *` sur l'utilisateur du tenant (l'incident
    # fondateur) : hors Terraform, par l'OAPI.
    user = outputs["user_name"]
    oapi("PutUserPolicy", {"UserName": user, "PolicyName": "pepin-qual-inline-admin",
                           "PolicyDocument": json.dumps({"Statement": [{"Effect": "Allow", "Action": ["*"], "Resource": ["*"]}]})})
    created.append(f"{user}/pepin-qual-inline-admin")
    return {"created": created}


def extra_destroy(outputs):
    deleted, left = [], []
    names = _bucket_names(outputs)
    for name in names.values():
        try:
            s3("delete-bucket-policy", "--bucket", name, check=False)
            s3("delete-bucket", "--bucket", name)
            deleted.append(name)
        except RuntimeError as e:
            if "NoSuchBucket" in str(e):
                continue
            left.append(f"{name} ({e})")
    user = outputs.get("user_name")
    if user:
        try:
            oapi("DeleteUserPolicy", {"UserName": user, "PolicyName": "pepin-qual-inline-admin"})
            deleted.append(f"{user}/pepin-qual-inline-admin")
        except RuntimeError as e:
            # 5105 : « The policy doesn't exist for the specified entity » — déjà absente
            # (un apply interrompu avant sa création, par exemple) : rien à laisser.
            if "5105" not in str(e) and "doesn't exist" not in str(e):
                left.append(f"inline policy ({e})")
    return {"deleted": deleted, "left": left}


# ─── inventaire ────────────────────────────────────────────────────────────────

def _tagged(item, tag):
    return any(t.get("Key") == tag for t in item.get("Tags") or [])


def inventory(tenant):
    tag, prefix = tenant["tag"], tenant["name_prefix"]
    me = identity()["account_id"]
    fam = {}

    def put(family, items, key_id, owned=None):
        owned = owned or (lambda i: _tagged(i, tag))
        fam[family] = {
            "all": sorted(str(i.get(key_id)) for i in items),
            "tenant": [{"id": i.get(key_id), "name": next((t["Value"] for t in i.get("Tags") or [] if t.get("Key") == "Name"), None)}
                       for i in items if owned(i)],
        }

    put("vms", [v for v in paged("ReadVms", "Vms") if v.get("State") != "terminated"], "VmId")
    put("nics", paged("ReadNics", "Nics"), "NicId")
    put("public_ips", paged("ReadPublicIps", "PublicIps"), "PublicIpId")
    put("security_groups", paged("ReadSecurityGroups", "SecurityGroups"), "SecurityGroupId",
        owned=lambda s: _tagged(s, tag) or str(s.get("SecurityGroupName", "")).startswith(prefix))
    put("volumes", paged("ReadVolumes", "Volumes"), "VolumeId")
    put("snapshots", paged("ReadSnapshots", "Snapshots", {"Filters": {"AccountIds": [me]}}), "SnapshotId")
    put("images", paged("ReadImages", "Images", {"Filters": {"AccountIds": [me]}}), "ImageId",
        owned=lambda i: _tagged(i, tag) or str(i.get("ImageName", "")).startswith(prefix))
    put("nets", paged("ReadNets", "Nets"), "NetId")
    put("subnets", paged("ReadSubnets", "Subnets"), "SubnetId")
    put("internet_services", paged("ReadInternetServices", "InternetServices"), "InternetServiceId")
    put("nat_services", paged("ReadNatServices", "NatServices"), "NatServiceId")
    put("route_tables", paged("ReadRouteTables", "RouteTables"), "RouteTableId")
    # Un appairage supprimé reste listé un moment en état « deleted » : ce n'est pas un reste.
    put("net_peerings", [p for p in paged("ReadNetPeerings", "NetPeerings")
                         if (p.get("State") or {}).get("Name") not in ("deleted", "rejected", "expired", "failed")], "NetPeeringId")
    put("load_balancers", oapi("ReadLoadBalancers").get("LoadBalancers") or [], "LoadBalancerName",
        owned=lambda l: _tagged(l, tag) or str(l.get("LoadBalancerName", "")).startswith(prefix))
    put("keypairs", paged("ReadKeypairs", "Keypairs"), "KeypairName",
        owned=lambda k: str(k.get("KeypairName", "")).startswith(prefix))
    put("api_access_rules", oapi("ReadApiAccessRules").get("ApiAccessRules") or [], "ApiAccessRuleId",
        owned=lambda r: str(r.get("Description", "")).startswith(prefix))
    users = oapi("ReadUsers").get("Users") or []
    put("users", users, "UserName", owned=lambda u: str(u.get("UserName", "")).startswith(prefix))
    put("policies", oapi("ReadPolicies", {"Filters": {"Scope": "LOCAL"}}).get("Policies") or [], "PolicyName",
        owned=lambda p: str(p.get("PolicyName", "")).startswith(prefix))
    keys = []
    for k in oapi("ReadAccessKeys").get("AccessKeys") or []:
        keys.append(dict(k, owner="<root>"))
    for u in users:
        for k in oapi("ReadAccessKeys", {"UserName": u["UserName"]}).get("AccessKeys") or []:
            keys.append(dict(k, owner=u["UserName"]))
    put("access_keys", keys, "AccessKeyId",
        owned=lambda k: str(k.get("owner", "")).startswith(prefix) or str(k.get("Tag") or "").startswith(prefix))
    put("buckets", [{"Name": b} for b in s3_buckets()], "Name", owned=lambda b: b["Name"].startswith(prefix))
    return fam


# ─── nettoyage de secours ──────────────────────────────────────────────────────

def cleanup(tenant, leftovers_path=None):
    inv = inventory(tenant)
    rest = {f: [x["id"] for x in v["tenant"]] for f, v in inv.items() if v["tenant"]}
    if leftovers_path:
        delta = json.load(open(leftovers_path)).get("delta") or {}
        for f, ids in delta.items():
            rest.setdefault(f, [])
            rest[f] = sorted(set(rest[f]) | set(ids))
    if not any(rest.values()):
        print("rien à nettoyer : aucune ressource du tenant")
        return True
    ok = True

    def do(label, call, body):
        nonlocal ok
        try:
            oapi(call, body)
            print(f"  ✔ {label}")
        except RuntimeError as e:
            ok = False
            print(f"  ✘ {label} : {e}")

    # L'ordre est celui des dépendances. Une VM protégée se déprotège d'abord (#88).
    for vm in rest.get("vms", []):
        do(f"UpdateVm {vm} DeletionProtection=false", "UpdateVm", {"VmId": vm, "DeletionProtection": False})
    if rest.get("vms"):
        do(f"DeleteVms {rest['vms']}", "DeleteVms", {"VmIds": rest["vms"]})
        # Attendre la terminaison avant de toucher NIC, SG, subnet et net.
        import time
        for _ in range(60):
            live = [v for v in paged("ReadVms", "Vms", {"Filters": {"VmIds": rest["vms"]}}) if v.get("State") != "terminated"]
            if not live:
                break
            time.sleep(5)
    for ip in rest.get("public_ips", []):
        do(f"UnlinkPublicIp/DeletePublicIp {ip}", "DeletePublicIp", {"PublicIpId": ip})
    for nic in rest.get("nics", []):
        do(f"DeleteNic {nic}", "DeleteNic", {"NicId": nic})
    for lb in rest.get("load_balancers", []):
        do(f"DeleteLoadBalancer {lb}", "DeleteLoadBalancer", {"LoadBalancerName": lb})
    for img in rest.get("images", []):
        do(f"DeleteImage {img}", "DeleteImage", {"ImageId": img})
    for vol in rest.get("volumes", []):
        do(f"DeleteVolume {vol}", "DeleteVolume", {"VolumeId": vol})
    for snap in rest.get("snapshots", []):
        do(f"DeleteSnapshot {snap}", "DeleteSnapshot", {"SnapshotId": snap})
    for sg in rest.get("security_groups", []):
        do(f"DeleteSecurityGroup {sg}", "DeleteSecurityGroup", {"SecurityGroupId": sg})
    for pe in rest.get("net_peerings", []):
        do(f"DeleteNetPeering {pe}", "DeleteNetPeering", {"NetPeeringId": pe})
    for rt in rest.get("route_tables", []):
        for link in (oapi("ReadRouteTables", {"Filters": {"RouteTableIds": [rt]}}).get("RouteTables") or [{}])[0].get("LinkRouteTables") or []:
            if not link.get("Main"):
                do(f"UnlinkRouteTable {link.get('LinkRouteTableId')}", "UnlinkRouteTable", {"LinkRouteTableId": link.get("LinkRouteTableId")})
        do(f"DeleteRouteTable {rt}", "DeleteRouteTable", {"RouteTableId": rt})
    for isvc in rest.get("internet_services", []):
        for net in rest.get("nets", []):
            oapi_try("UnlinkInternetService", {"InternetServiceId": isvc, "NetId": net})
        do(f"DeleteInternetService {isvc}", "DeleteInternetService", {"InternetServiceId": isvc})
    for sn in rest.get("subnets", []):
        do(f"DeleteSubnet {sn}", "DeleteSubnet", {"SubnetId": sn})
    for net in rest.get("nets", []):
        do(f"DeleteNet {net}", "DeleteNet", {"NetId": net})
    for b in rest.get("buckets", []):
        try:
            s3("delete-bucket-policy", "--bucket", b, check=False)
            s3("delete-bucket", "--bucket", b)
            print(f"  ✔ DeleteBucket {b}")
        except RuntimeError as e:
            ok = False
            print(f"  ✘ DeleteBucket {b} : {e}")
    for rule in rest.get("api_access_rules", []):
        do(f"DeleteApiAccessRule {rule}", "DeleteApiAccessRule", {"ApiAccessRuleId": rule})
    for k in rest.get("access_keys", []):
        # Une clé du tenant appartient au root (clé `root`) ou à l'utilisateur du tenant :
        # l'appel sans UserName vise le root, celui de l'utilisateur suit plus bas.
        do(f"DeleteAccessKey {k}", "DeleteAccessKey", {"AccessKeyId": k})
    for u in rest.get("users", []):
        for pol in (oapi_try("ReadUserPolicies", {"UserName": u}) or {}).get("PolicyNames") or []:
            do(f"DeleteUserPolicy {u}/{pol}", "DeleteUserPolicy", {"UserName": u, "PolicyName": pol})
        for k in (oapi_try("ReadAccessKeys", {"UserName": u}) or {}).get("AccessKeys") or []:
            do(f"DeleteAccessKey {u}/{k['AccessKeyId']}", "DeleteAccessKey", {"UserName": u, "AccessKeyId": k["AccessKeyId"]})
        for lp in (oapi_try("ReadLinkedPolicies", {"UserName": u}) or {}).get("Policies") or []:
            do(f"UnlinkPolicy {u}/{lp.get('PolicyName')}", "UnlinkPolicy", {"UserName": u, "PolicyOrn": lp.get("Orn")})
        do(f"DeleteUser {u}", "DeleteUser", {"UserName": u})
    for pol in rest.get("policies", []):
        orn = next((p.get("Orn") for p in oapi("ReadPolicies", {"Filters": {"Scope": "LOCAL"}}).get("Policies") or [] if p.get("PolicyName") == pol), None)
        if orn:
            do(f"DeletePolicy {pol}", "DeletePolicy", {"PolicyOrn": orn})
    return ok


def oapi_try(call, body):
    try:
        return oapi(call, body)
    except RuntimeError:
        return None


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
