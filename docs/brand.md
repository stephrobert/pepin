> 🇬🇧 English · [🇫🇷 Français](brand.fr.md)

# The mark

![Pépin](assets/brand/pepin-lockup-light.svg)

A shield, with a pip in it. Not in front of it, not beside it: **inside**, as a
hole, because that is where Pépin finds it. The name already carries both
meanings at once, the seed and the trouble you uncover, and the mark only holds
them together.

The pip is not a light shape sitting on the shield, it is a **void** through it.
On any background, that background shows through. It comes down to one
`fill-rule`, and that is what separates a logo from a picture of a logo.

## One reservation, written down rather than left unsaid

The shield is the single most used sign in all of information security. It says
"I protect", whereas Pépin protects nothing: **it establishes facts**. It reads
an effective configuration, checks it against a framework and returns a verdict
you can defend, with its normative references and a sealed proof. An auditor
does not protect, they look and they sign.

The chosen concept therefore accepts a familiar sign at the cost of a slightly
off promise. That is a defensible trade, familiarity being also what makes a
sign readable in one second, and it is recorded here so it stays a choice rather
than an oversight.

What redeems the shield is the hole: a pierced shield is not quite a shield any
more, and that is exactly what the tool says.

## Files

Everything is in [`assets/brand/`](assets/brand/). **The SVG files are the
source**; the PNG files derive from them and are regenerated, never edited.

| File | Use |
|---|---|
| `pepin-lockup-light.svg` | the default, on light backgrounds |
| `pepin-lockup-dark.svg` | on dark backgrounds |
| `pepin-lockup-mono.svg` | inherits `currentColor` |
| `pepin-lockup-vertical-light.svg` / `-dark.svg` | the stacked lockup, when width is scarce |
| `pepin-icon.svg` / `pepin-icon-dark.svg` | the mark alone, from 20px up |
| `pepin-icon-mono.svg` | the mark alone, single ink, for favicons and terminals |

### PNG, for what does not take vectors

[`assets/brand/png/`](assets/brand/png/) holds the rasterised versions: GitHub's
social preview, a slide, a thumbnail, an audit report exported by a tool that
ignores SVG.

| File | Size | Use |
|---|---|---|
| `pepin-icon-256/512/1024.png` | square, transparent | avatars, thumbnails, packaging |
| `pepin-icon-dark-512/1024.png` | square, transparent | the same, on dark backgrounds |
| `pepin-lockup-light-1024/2048.png` | horizontal, transparent | slides, articles |
| `pepin-lockup-dark-1024/2048.png` | idem | the same, on dark backgrounds |
| `pepin-lockup-vertical-light-1024.png` / `-dark-1024.png` | stacked, transparent | when width is scarce |
| `pepin-social-preview.png` | 1280×640 | GitHub Settings → Social preview |
| `pepin-social-preview-dark.png` | 1280×640 | the dark variant |

Regenerate them with `python3 scripts/generer-png-marque.py` rather than
exporting by hand. The script rasterises **at the target resolution**, computing
`-density` from each file's `viewBox`: `convert file.svg -resize 1024x` would
rasterise at the viewBox size, a few dozen pixels, and then enlarge that bitmap,
which is blurry and obvious at 100%. A constant density would be no better: it is
only right for the viewBox it was chosen for, and the square icon does not share
the lockup's.

## How it is embedded

GitHub switches on the reader's theme through `<picture>`:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/pepin-lockup-dark.svg">
  <img src="docs/assets/brand/pepin-lockup-light.svg" alt="Pépin" width="200">
</picture>
```

**The wordmark is outlines, not text**, and the acute accent is why. An SVG
carrying `<text font-family="Poppins">` renders in whatever the reader's browser
happens to have, because GitHub serves it no webfont; and a fallback font can
shift the accent, recompose it, or render it as a separate character. On a
five-letter name whose accent carries the identity, that shows.

The five glyphs are Poppins SemiBold converted to curves. Poppins is under the
SIL Open Font License: only these shapes travel in this repository, never the
font.

## Using it

- **Clear space**: the width of the shield on every side. Nothing enters it.
- **Smallest sizes**: 24px for the lockup, 20px for the mark alone. Below that
  the pip's hole closes up and only a solid shield is left.
- **The pip is a void**, not a white shape. Do not fill it, not even to "fix" a
  rendering on a coloured background: that is precisely the case it handles.
- **On dark backgrounds, use the dark file.**

Do not separate the shield from the wordmark in the lockup, do not set the
wordmark in another typeface, do not add a baseline or effects, and do not
square off the two top corners: their radius answers the point at the bottom,
and two sharp angles above a rounded point read as two drawings stitched
together.

## Editing it

**Nothing is edited by hand.** The SVG files come out of
`scripts/generer-marque.py`, the PNG files come out of the SVG: change the
script, then regenerate.

```bash
python3 -m pip install -r scripts/requirements.txt
python3 scripts/generer-marque.py --verifier   # the boxes, writing nothing
python3 scripts/generer-marque.py
python3 scripts/generer-png-marque.py
```

The mark is **a single path**: the shield then the pip, in the same `d`, with
`fill-rule="evenodd"`. Splitting them into two shapes, one filled white, would
break transparency and make the logo unusable on anything but white.

**The layout is computed from the font, not eyeballed**, and that is what caught
this drawing's most expensive defect: with the baseline set to a constant picked
by hand, the descender of the `p` fell outside the `viewBox` and was clipped.
The check is one line, the lockup being cropped tight to its ink: `identify
-format '%@' file.png` must return exactly the canvas dimensions, at origin
`+0+0`.

## What this repository cannot version

- **The social preview setting.** The image itself is versioned, at
  `assets/brand/png/pepin-social-preview.png`; what cannot be is the setting
  that points GitHub at it. Upload it by hand: Settings → Social preview.
- **The account avatar**, which GitHub takes from the owner, not the repository.

## Licence

**The name *Pépin* and the logo are not covered by the Apache 2.0 licence** that
covers the code. They may be used to refer to this project, in an article, a
talk, a comparison, a list of tools, without asking. They may not be used as the
mark of a fork, a product or a service, or in a way that suggests the project
endorses something it does not.

That distinction matters more here than elsewhere: Pépin issues compliance
verdicts, and a mark reused by a third party would suggest an endorsement the
project never gave.
