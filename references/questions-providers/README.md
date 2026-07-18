# Questions aux providers

Registre des points qu'on **n'a pas pu trancher avec la doc/SDK** et qui doivent
être confirmés auprès du provider (support, account team, issue tracker) AVANT
d'écrire un contrat `etat: verifie`, une règle ou de conclure « absent / non
applicable ». C'est le filet de la règle d'or §2 « ne rien affirmer sans valider » :
plutôt que de supposer, on **consigne la question**.

## Cycle de vie d'une question

1. **Ouverte** — détectée pendant l'ancrage, pas encore posée.
2. **En attente** — posée au provider, réponse attendue (noter où/quand).
3. **Résolue** — réponse obtenue : reporter la réponse + sa source, mettre à jour
   le contrat (`referentiel`/`providers/<nom>.yaml`) et **clore** la question ici.

## Convention

Un fichier par provider (`<provider>.md`). Chaque question :

```
### Q<n> — <titre court>   ·   contrôle: <CLD-* / code agnostique>   ·   statut: ouverte|en attente|résolue
- Contexte : pourquoi la question se pose (ce qu'on veut couvrir).
- Ce qu'on sait : faits déjà ancrés (avec source references/docs/… ou SDK).
- À confirmer : la question précise posée au provider.
- Source : liens doc/SDK consultés.
- Réponse : (si résolue) réponse + source + date + impact (contrat/règle mis à jour).
```

Voir aussi : le cache local de doc officielle sous `references/docs/<provider>/`
(généré par `scripts/fetch-docs.py`).
