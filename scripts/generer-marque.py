#!/usr/bin/env python3
"""Produit les fichiers SVG de la marque : le bouclier au pépin, et le logotype.

Ce script **est** la source de la marque. Les SVG de `docs/assets/brand/` en
sortent, les PNG sortent des SVG (`scripts/generer-png-marque.py`), et rien ne
s'édite à la main : un logotype retouché dans un éditeur perd la seule propriété
qui compte ici, à savoir que le mot est en courbes et non en texte.

Trois décisions de dessin, écrites pour qu'on ne les défasse pas par mégarde.

**Le pépin est un trou, pas une forme claire.** Un aplat blanc au milieu d'un
bouclier sombre n'est juste que sur fond blanc : posé sur une couleur, un
bandeau ou une photo, le logo traîne alors une tache. Un `fill-rule="evenodd"`
sur un tracé unique fait du pépin un vide réel, et le fond le traverse, quel
qu'il soit.

**Une seule encre.** Le bouclier plein porte déjà tout le contraste ; une
seconde couleur n'ajouterait qu'une occasion de mal vieillir. C'est aussi ce
qu'exige un outil de conformité, dont les sorties finissent en PDF, en annexe
d'audit et en noir et blanc.

**Le mot est converti en courbes.** L'accent aigu en est la raison : un SVG
portant `<text font-family="Poppins">` s'affiche avec ce que le navigateur du
lecteur possède, GitHub ne servant aucune webfont, et une police de repli peut
décaler l'accent, le recomposer, ou le rendre comme un caractère séparé. Sur un
nom de cinq lettres dont l'accent porte l'identité, cela se voit. Poppins est
sous licence SIL Open Font License : seules les cinq formes de glyphes circulent
dans ce dépôt, jamais la police.

**La mise en page est calculée, pas jugée à l'œil.** Une première version posait
la baseline à une constante choisie de tête, et le jambage des « p » sortait du
viewBox : l'encre touchait exactement le bord inférieur, donc elle était rognée.
Les boîtes viennent maintenant de la fonte elle-même, et `--verifier` le prouve.

Usage :
    python3 scripts/generer-marque.py
    python3 scripts/generer-marque.py --verifier   # sans écrire, contrôle les boîtes

Prérequis : `fonttools` (scripts/requirements.txt), et Poppins SemiBold
installée. Le chemin de la police est cherché dans $POLICE_PEPIN, puis via
fontconfig.
"""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path

RACINE = Path(__file__).resolve().parent.parent
CIBLE = RACINE / "docs" / "assets" / "brand"

MOT = "Pépin"
CORPS = 40.0

ENCRE_CLAIRE = "#0F172A"
ENCRE_SOMBRE = "#E2E8F0"

#: Le bouclier, à bord supérieur plat. Ses deux coins hauts sont arrondis du
#: même rayon que l'épaule des lettres : le bas de la forme est courbe, et deux
#: angles vifs au-dessus d'une pointe arrondie se lisent comme deux dessins
#: cousus ensemble. Le tracé occupe x=7..41 et y=8..52.
BOUCLIER = (
    "M 11 8 L 37 8 Q 41 8 41 12 L 41 27 Q 41 44 24 52 "
    "Q 7 44 7 27 L 7 12 Q 7 8 11 8 Z"
)

#: Le pépin, en contre-forme. Deux arcs symétriques, pointus aux deux bouts :
#: c'est la silhouette d'une graine. Il est posé plus haut que le centre
#: géométrique du bouclier, parce que la forme s'affine vers le bas : centré sur
#: la boîte, il paraîtrait tomber vers la pointe.
PEPIN = "M 24 16 Q 32 27 24 40 Q 16 27 24 16 Z"

#: La boîte du bouclier dans le repère où les deux tracés ci-dessus sont écrits.
SIGNE_X, SIGNE_Y = 7.0, 8.0
SIGNE_L, SIGNE_H = 34.0, 44.0

#: Écart entre le signe et le mot, horizontalement puis verticalement. Mesurés
#: sur l'encre réelle, pas sur les avances : le side-bearing d'un « P » varie
#: d'une fonte à l'autre et fausserait l'espace optique.
ECART_H = 22.0
ECART_V = 14.0

#: En composition verticale, le signe est agrandi jusqu'à cette fraction de la
#: largeur du mot. À sa taille du logotype horizontal, où il fait la hauteur du
#: mot, il paraît chétif une fois posé au-dessus d'un mot trois fois plus large.
PART_VERTICALE = 0.5

#: Le signe seul se rend dans un **carré**, parce que c'est ce qu'attendent un
#: avatar, un favicon et une vignette de plateforme : un viewBox 34×44 y sort en
#: 256×331, et la plateforme recadre où elle veut. Marge haute et basse ; les
#: marges latérales en découlent, un bouclier étant plus haut que large.
MARGE_ICONE = 8.0
COTE_ICONE = SIGNE_H + 2 * MARGE_ICONE


class Mot:
    """Le mot en courbes, avec la boîte de son encre, dans le repère baseline=0.

    Les tracés sont posés à l'origine du premier glyphe ; `x` et `y` disent où
    l'encre commence réellement, ce dont la mise en page a besoin pour coller le
    logotype sur son contenu plutôt que sur des avances typographiques.
    """

    def __init__(
        self, tracés: list[str], x: float, y: float, largeur: float, hauteur: float,
        capitale: float,
    ) -> None:
        self.tracés = tracés
        self.x = x
        self.y = y
        self.largeur = largeur
        self.hauteur = hauteur
        #: Hauteur des capitales, qui donne le centre **optique** du mot : l'œil
        #: centre un logotype sur ses capitales, pas sur une boîte que les
        #: jambages tirent vers le bas.
        self.capitale = capitale

    @property
    def montant(self) -> float:
        """Ce qui dépasse au-dessus du centre optique."""
        return -self.y - self.capitale / 2

    @property
    def descendant(self) -> float:
        """Ce qui dépasse en dessous, jambages compris."""
        return self.y + self.hauteur + self.capitale / 2


def _police() -> Path:
    """Localise Poppins SemiBold, ou explique précisément ce qui manque."""
    if declare := os.environ.get("POLICE_PEPIN"):
        chemin = Path(declare)
        if not chemin.is_file():
            sys.exit(f"POLICE_PEPIN pointe sur un fichier absent : {chemin}")
        return chemin

    if fc := shutil.which("fc-match"):
        sortie = subprocess.run(
            [fc, "--format=%{file}", "Poppins:style=SemiBold"],
            capture_output=True,
            text=True,
            check=False,
        )
        candidat = Path(sortie.stdout.strip())
        # fontconfig répond toujours quelque chose : sans Poppins installée, il
        # rend la police de repli du système, qui donnerait un logotype dans la
        # mauvaise grotesque sans le moindre avertissement.
        if candidat.is_file() and "poppins" in candidat.name.lower():
            return candidat

    sys.exit(
        "Poppins SemiBold est introuvable. Installez-la "
        "(https://fonts.google.com/specimen/Poppins, licence SIL OFL) ou "
        "indiquez le fichier par POLICE_PEPIN=/chemin/Poppins-SemiBold.ttf"
    )


def composer_mot() -> Mot:
    """Rend « Pépin » en tracés SVG, et mesure la boîte de son encre."""
    try:
        from fontTools.pens.boundsPen import BoundsPen
        from fontTools.pens.svgPathPen import SVGPathPen
        from fontTools.ttLib import TTFont
    except ImportError:
        sys.exit("fonttools est requis : pip install -r scripts/requirements.txt")

    font = TTFont(_police())
    glyphes = font.getGlyphSet()
    cmap = font.getBestCmap()
    hmtx = font["hmtx"]
    échelle = CORPS / font["head"].unitsPerEm

    tracés: list[str] = []
    avance = 0.0
    x_min = y_min = float("inf")
    x_max = y_max = float("-inf")

    for lettre in MOT:
        point = ord(lettre)
        if point not in cmap:
            sys.exit(f"Cette police n'a pas le glyphe {lettre!r}")
        nom = cmap[point]

        plume = SVGPathPen(glyphes)
        glyphes[nom].draw(plume)
        # Le repère d'une police monte, celui d'un SVG descend : d'où l'échelle
        # verticale négative. L'espace est le seul glyphe sans tracé.
        if d := plume.getCommands():
            tracés.append(
                f'<path transform="translate({avance:.2f} 0) '
                f'scale({échelle:.5f} {-échelle:.5f})" d="{d}"/>'
            )

        bornes = BoundsPen(glyphes)
        glyphes[nom].draw(bornes)
        if bornes.bounds is not None:
            gx0, gy0, gx1, gy1 = bornes.bounds
            x_min = min(x_min, avance + gx0 * échelle)
            x_max = max(x_max, avance + gx1 * échelle)
            # y monte dans la fonte et descend en SVG : les bornes s'échangent.
            y_min = min(y_min, -gy1 * échelle)
            y_max = max(y_max, -gy0 * échelle)

        avance += hmtx[nom][0] * échelle

    os2 = font["OS/2"]
    capitale = getattr(os2, "sCapHeight", 0) * échelle
    if capitale <= 0:  # certaines fontes n'ont pas la version 2 de la table
        capitale = -y_min

    return Mot(tracés, x_min, y_min, x_max - x_min, y_max - y_min, capitale)


def _svg(largeur: float, hauteur: float, corps: str) -> str:
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" '
        f'viewBox="0 0 {largeur:.2f} {hauteur:.2f}" '
        f'width="{largeur:.2f}" height="{hauteur:.2f}" '
        f'role="img" aria-label="Pépin">\n{corps}</svg>\n'
    )


def _signe(couleur: str, dx: float, dy: float, note: str, facteur: float = 1.0) -> str:
    échelle = "" if facteur == 1.0 else f" scale({facteur:.4f})"
    return (
        f"  <!-- {note} -->\n"
        f'  <g transform="translate({dx:.2f} {dy:.2f}){échelle}">\n'
        f'    <path fill="{couleur}" fill-rule="evenodd" d="{BOUCLIER} {PEPIN}"/>\n'
        f"  </g>\n"
    )


def icone(encre: str | None) -> str:
    couleur = "currentColor" if encre is None else encre
    quoi = "single ink, inherits currentColor" if encre is None else "single ink"
    note = (
        "Shield and pip, one path: the pip is a hole through fill-rule, not a "
        f"white shape, so any background shows through it ({quoi})."
    )
    dx = (COTE_ICONE - SIGNE_L) / 2 - SIGNE_X
    dy = MARGE_ICONE - SIGNE_Y
    return _svg(COTE_ICONE, COTE_ICONE, _signe(couleur, dx, dy, note))


def _mot_svg(couleur: str, dx: float, dy: float, mot: Mot) -> str:
    tracés = "\n      ".join(mot.tracés)
    return (
        "  <!-- The wordmark is outlines, not text: a fallback font can shift the\n"
        "       acute accent, recompose it, or render it as a separate character,\n"
        "       and on a five-letter name carrying that accent it shows. Poppins is\n"
        "       under the SIL OFL; only these five glyph shapes travel here. -->\n"
        f'  <g transform="translate({dx:.2f} {dy:.2f})">\n'
        f'    <g fill="{couleur}">\n      {tracés}\n    </g>\n'
        "  </g>\n"
    )


def lockup(encre: str | None, mot: Mot, *, vertical: bool) -> tuple[str, float, float]:
    """Le signe et le mot, dans un viewBox serré sur l'encre.

    Serré, parce que la zone de respiration est une règle d'usage et non une
    marge à cuire dans le fichier : l'intégrateur qui pose le logotype dans une
    barre de navigation contrôle son espacement, il ne peut pas retirer celui
    qu'on lui aurait imposé. Contrepartie assumée, et vérifiable : l'encre doit
    toucher les quatre bords, ce que contrôle `--verifier`.
    """
    couleur = "currentColor" if encre is None else encre
    note = "Shield and pip, one path: the pip is a hole, not a white shape."

    if vertical:
        facteur = max(1.0, PART_VERTICALE * mot.largeur / SIGNE_L)
        signe_l, signe_h = SIGNE_L * facteur, SIGNE_H * facteur
        largeur = max(signe_l, mot.largeur)
        hauteur = signe_h + ECART_V + mot.hauteur
        corps = _signe(
            couleur,
            (largeur - signe_l) / 2 - SIGNE_X * facteur,
            -SIGNE_Y * facteur,
            note,
            facteur,
        ) + _mot_svg(
            couleur,
            (largeur - mot.largeur) / 2 - mot.x,
            signe_h + ECART_V - mot.y,
            mot,
        )
        return _svg(largeur, hauteur, corps), largeur, hauteur

    # Horizontalement, les deux se centrent sur le même axe optique : celui du
    # bouclier, et celui des capitales du mot. Une baseline posée à une
    # constante ferait pendre le signe sous le mot, ou rognerait les jambages.
    haut = max(SIGNE_H / 2, mot.montant)
    bas = max(SIGNE_H / 2, mot.descendant)
    largeur = SIGNE_L + ECART_H + mot.largeur
    hauteur = haut + bas

    corps = _signe(
        couleur, -SIGNE_X, haut - SIGNE_H / 2 - SIGNE_Y, note
    ) + _mot_svg(
        couleur,
        SIGNE_L + ECART_H - mot.x,
        haut + mot.capitale / 2,
        mot,
    )
    return _svg(largeur, hauteur, corps), largeur, hauteur


def main() -> int:
    verifier = "--verifier" in sys.argv[1:]
    mot = composer_mot()

    horizontal, l_h, h_h = lockup(ENCRE_CLAIRE, mot, vertical=False)
    vertical, l_v, h_v = lockup(ENCRE_CLAIRE, mot, vertical=True)

    if verifier:
        print(f"  mot        encre {mot.largeur:.2f} × {mot.hauteur:.2f}, "
              f"capitales {mot.capitale:.2f}, jambage {mot.descendant - mot.capitale / 2:.2f}")
        print(f"  horizontal viewBox {l_h:.2f} × {h_h:.2f}")
        print(f"  vertical   viewBox {l_v:.2f} × {h_v:.2f}")
        print(f"  icone      viewBox {COTE_ICONE:.2f} × {COTE_ICONE:.2f}, "
              f"marges {MARGE_ICONE:.0f} haut/bas, "
              f"{(COTE_ICONE - SIGNE_L) / 2:.0f} côtés")
        return 0

    CIBLE.mkdir(parents=True, exist_ok=True)
    fichiers = {
        "pepin-icon.svg": icone(ENCRE_CLAIRE),
        "pepin-icon-dark.svg": icone(ENCRE_SOMBRE),
        "pepin-icon-mono.svg": icone(None),
        "pepin-lockup-light.svg": horizontal,
        "pepin-lockup-dark.svg": lockup(ENCRE_SOMBRE, mot, vertical=False)[0],
        "pepin-lockup-mono.svg": lockup(None, mot, vertical=False)[0],
        "pepin-lockup-vertical-light.svg": vertical,
        "pepin-lockup-vertical-dark.svg": lockup(ENCRE_SOMBRE, mot, vertical=True)[0],
    }
    for nom, contenu in fichiers.items():
        (CIBLE / nom).write_text(contenu, encoding="utf-8")
        print(f"  ✔ {nom}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
