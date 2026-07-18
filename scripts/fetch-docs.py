#!/usr/bin/env python3
"""Met en cache local, en Markdown, la doc officielle des providers (ancrage §2).

Beaucoup de portails (community.exoscale.com, docs.outscale.com, scaleway.com…)
rendent leur contenu côté client : un GET « brut » ne renvoie que la coquille de
navigation. trafilatura télécharge le HTML complet et en EXTRAIT le contenu utile,
ce qui perce ces rendus. On fige le résultat sous references/docs/<provider>/ pour
ancrer les contrats/contrôles sur une source vérifiable et hors-ligne.

Deux modes, déclarés dans references/docs/sources.yaml :
  - `pages:` {provider: [urls]}        — pages ciblées (liste explicite)
  - `crawl:` {provider: {sitemaps, include, exclude}} — TOUTE la doc via sitemap
    (les sitemap-index sont résolus récursivement ; include/exclude filtrent les URLs).

Usage :
    python3 scripts/fetch-docs.py [provider ...]   # tous, ou ceux listés
    REFRESH=1 python3 scripts/fetch-docs.py …       # réécrit même les pages en cache
Reprise : par défaut une page déjà en cache est sautée (idempotent, reprenable).
"""
from __future__ import annotations

import datetime
import hashlib
import os
import pathlib
import re
import sys
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from xml.etree import ElementTree

import trafilatura
import yaml

ROOT = pathlib.Path(__file__).resolve().parent.parent
DOCS = ROOT / "references" / "docs"
SOURCES = DOCS / "sources.yaml"
UA = {"User-Agent": "Mozilla/5.0 (pepin-doc-cache)"}
WORKERS = int(os.environ.get("WORKERS", "8"))
REFRESH = os.environ.get("REFRESH") == "1"


def slug(url: str) -> str:
    """Nom de fichier déterministe dérivé du chemin de l'URL (sans le domaine)."""
    path = re.sub(r"^https?://[^/]+/", "", url).rstrip("/")
    s = re.sub(r"[^a-zA-Z0-9]+", "-", path).strip("-").lower() or "index"
    if len(s) > 120:  # tronque mais garde l'unicité via un hash court
        s = s[:110] + "-" + hashlib.sha1(url.encode()).hexdigest()[:8]
    return s


def sitemap_urls(sitemap: str, seen: set[str]) -> list[str]:
    """Retourne les URLs de page d'un sitemap, en résolvant les index récursivement."""
    if sitemap in seen:
        return []
    seen.add(sitemap)
    try:
        req = urllib.request.Request(sitemap, headers=UA)
        raw = urllib.request.urlopen(req, timeout=30).read()
    except Exception as e:  # noqa: BLE001 — un sitemap KO ne doit pas tout casser
        print(f"  ! sitemap inaccessible {sitemap} : {e}")
        return []
    try:
        root = ElementTree.fromstring(raw)
    except ElementTree.ParseError as e:
        print(f"  ! sitemap illisible {sitemap} : {e}")
        return []
    locs = [el.text.strip() for el in root.iter() if el.tag.endswith("loc") and el.text]
    out: list[str] = []
    for loc in locs:
        if loc.endswith(".xml"):  # sitemap-index → descendre
            out.extend(sitemap_urls(loc, seen))
        else:
            out.append(loc)
    return out


def keep(url: str, include: list[str], exclude: list[str]) -> bool:
    if any(x in url for x in exclude):
        return False
    return not include or any(x in url for x in include)


def collect_urls(cfg: dict, provider: str) -> list[str]:
    """Construit la liste d'URLs d'un provider (pages explicites + crawl sitemap)."""
    urls: list[str] = list((cfg.get("pages") or {}).get(provider) or [])
    crawl = (cfg.get("crawl") or {}).get(provider)
    if crawl:
        seen: set[str] = set()
        found: list[str] = []
        for sm in crawl.get("sitemaps", []):
            found.extend(sitemap_urls(sm, seen))
        inc, exc = crawl.get("include", []), crawl.get("exclude", [])
        urls.extend(u for u in found if keep(u, inc, exc))
    # dédoublonne en gardant l'ordre
    return list(dict.fromkeys(urls))


def grab(url: str, outdir: pathlib.Path, today: str) -> tuple[str, str]:
    """Récupère + extrait une URL. Retourne (statut, url) : ok|skip|vide|erreur."""
    dest = outdir / f"{slug(url)}.md"
    if dest.exists() and not REFRESH:
        return "skip", url
    try:
        html = trafilatura.fetch_url(url)
        md = (
            trafilatura.extract(
                html, output_format="markdown", include_tables=True, include_links=True
            )
            if html
            else None
        )
    except Exception as e:  # noqa: BLE001
        return f"erreur ({e})", url
    if not md:
        return "vide", url
    header = (
        f"<!-- source : {url}\n"
        f"     récupéré le {today} via trafilatura — généré, ne pas éditer à la main. -->\n\n"
    )
    dest.write_text(header + md + "\n", encoding="utf-8")
    return "ok", url


def main(argv: list[str]) -> int:
    cfg = yaml.safe_load(SOURCES.read_text(encoding="utf-8")) or {}
    providers = set(cfg.get("pages", {})) | set(cfg.get("crawl", {}))
    wanted = (set(argv) & providers) or (providers if not argv else set())
    if argv and not wanted:
        print(f"  ! aucun provider connu parmi {argv} ; connus : {sorted(providers)}")
        return 1
    today = datetime.date.today().isoformat()
    grand_ok = grand_total = 0
    for provider in sorted(wanted):
        urls = collect_urls(cfg, provider)
        outdir = DOCS / provider
        outdir.mkdir(parents=True, exist_ok=True)
        print(f"\n# {provider} : {len(urls)} pages")
        ok = skip = fail = 0
        with ThreadPoolExecutor(max_workers=WORKERS) as pool:
            futs = {pool.submit(grab, u, outdir, today): u for u in urls}
            for i, fut in enumerate(as_completed(futs), 1):
                status, url = fut.result()
                if status == "ok":
                    ok += 1
                elif status == "skip":
                    skip += 1
                else:
                    fail += 1
                    print(f"  ✗ {status} — {url}")
                if i % 100 == 0:
                    print(f"  … {i}/{len(urls)} (ok={ok} skip={skip} échecs={fail})")
        print(f"  → {provider} : ok={ok} skip(cache)={skip} échecs={fail}")
        grand_ok += ok + skip
        grand_total += len(urls)
    print(f"\n{grand_ok}/{grand_total} pages en cache sous references/docs/.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
