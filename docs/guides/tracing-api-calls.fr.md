> [🇬🇧 English](tracing-api-calls.md) · 🇫🇷 Français

# Tracer les appels réels d'un collecteur

Un descripteur de fournisseur **déclare** des endpoints. Rien, en soi, ne prouve que le
collecteur les **émet**. Cet écart porte un nom dans ce projet : c'est la forme de l'incident
des politiques EIM inline, où la règle était juste, la donnée n'arrivait jamais, et aucun test
Rego ne pouvait le voir.

Cette page dit comment le combler en **mesurant** plutôt qu'en lisant.

## Ce qu'une trace établit, et ce qu'elle n'établit pas

À lire avant de citer une trace comme preuve. Tout ce qui figure en colonne de droite reste dû
à un scan réel contre un tenant réel, que ce dépôt ne fait jamais : il ne détient aucun
identifiant cloud, par construction.

| Un enregistrement contre l'émulateur local établit | Il n'établit **pas** |
|---|---|
| les endpoints que le collecteur émet réellement, face à ceux que `providers/<nom>.yaml` déclare | les noms et types de champs du contrat natif du fournisseur |
| qu'une jointure parent/enfant tire, sur un identifiant lu dans la réponse du parent | les bornes réelles de pagination du fournisseur |
| les paramètres de pagination réellement posés sur le réseau | son comportement de limitation de débit |
| la classe attribuée à un échec de collecte (`not_found`, `unavailable`…) | que le fournisseur réponde `403` plutôt que `200` avec un corps d'erreur |
| qu'aucun contrôle ne rende `pass` depuis une unité incomplète | quoi que ce soit de l'inventaire d'un vrai tenant |

**Un émulateur prouve ce que Pépin FAIT, pas ce que le cloud RÉPOND.** Ne jamais confondre les
deux : c'est exactement la fausse confiance que ce projet existe pour refuser.

Une conséquence mérite d'être dite en clair. L'émulateur **accepte n'importe quel identifiant**
et n'offre aucune injection de panne. C'est mesuré : sans en-tête d'authentification il rend
`200`, avec un jeton bidon il rend `200`, et aucune route de panne n'existe. Il **ne peut donc
pas produire de `403`**. La classification d'un refus est éprouvée ailleurs, par
`internal/collect/status_test.go`, contre une vraie socket qui refuse vraiment. Ce qui reste
non mesuré, c'est de savoir si tel fournisseur refuse bien avec ce statut.

## La procédure

```bash
mise run build
PROVIDER=scaleway mise run trace          # ou : ./scripts/trace-collector.sh scaleway .trace
```

C'est tout. Il faut [feint](https://github.com/stephrobert/feint) 0.10.0 ou plus dans le `PATH`
et `unshare` (util-linux). Il ne faut **aucun identifiant cloud, et aucune API réelle n'est
touchée**.

### Pourquoi la chaîne a deux étages de proxy

Aucune `base_url` de collecte n'est redirigeable : elles sont figées à la compilation, et seul
`--s3-endpoint` est une option. Ce qui est vrai, en revanche, et qui rend tout ceci possible,
c'est que le client de collecte n'installe pas de `Transport`. Il hérite de
`http.DefaultTransport`, donc il **honore `HTTPS_PROXY`**.

```
Pépin ──HTTPS_PROXY, CONNECT──▶ proxy AMONT (--forward, enregistre)
                                     │ redial vers l'hôte que le client a demandé
                                     ▼   (résolu sur 127.0.0.1 par /etc/hosts)
                                proxy AVAL (--intercept, termine le TLS)
                                     │ --upstream
                                     ▼
                                feint serve (l'émulateur)
```

feint 0.10.0 **refuse `--forward` et `--upstream` ensemble**, et il a raison : `--forward`
envoie chaque requête à l'hôte que le *client* a demandé, `--upstream` envoie toute requête à
l'hôte que *vous* avez choisi. Le second étage est ce qui fait que « l'hôte demandé » soit
l'émulateur, et non le vrai cloud.

**Rien de ce qui vous appartient n'est modifié.** Toute la chaîne tourne dans un espace de noms
(user, mount et net) : le `/etc/hosts` remplacé est celui de cet espace, jamais le vôtre, et le
port 443 lié vit dans une pile réseau privée qui disparaît avec le dernier processus.
`--vm off` interdit à l'émulateur de démarrer le moindre conteneur avec vos privilèges.

**Aucune ligne de Pépin n'est modifiée non plus**, et c'est le sujet, pas une commodité. Un
endpoint de collecte surchargeable a été identifié par l'audit de livraison comme un moyen
d'envoyer la clé secrète d'un tenant vers un hôte arbitraire. Chaque requête de collecte porte
cette clé en en-tête. La chaîne ci-dessus n'ajoute aucune surface de ce genre.

### Le stockage objet : la démonstration la moins coûteuse

`--s3-endpoint` est le seul endpoint qui se redirige déjà, donc le collecteur S3 n'a besoin ni
de `CONNECT` ni d'interception TLS :

```bash
feint proxy --addr 127.0.0.1:4601 --upstream http://127.0.0.1:4599 --record s3.jsonl
pepin scan scaleway --live --s3-endpoint http://127.0.0.1:4601
```

L'émulateur ne sert aucune surface de stockage objet : `ListBuckets` revient donc en `404`.
C'est tout de même une mesure, et une mesure utile. C'est la première fois que la branche S3
de `collect.Classify`, celle qui lit l'erreur du SDK AWS par une interface anonyme plutôt que
par un type, est éprouvée contre une vraie réponse HTTP plutôt que contre une erreur construite.

## Lire un enregistrement

La transcription est du JSON Lines, un objet par échange, l'opération amont nommée :

```json
{"seq":1,"method":"GET","path":"/iam/v1alpha1/api-keys","host":"api.scaleway.com",
 "status":501,"mounted":false,
 "req":{"headers":{"X-Auth-Token":"REDACTED"}},
 "res":{"body":{"type":"not_emulated"}}}
```

Ce qu'on y cherche, par ordre de valeur :

1. **Les endpoints émis face aux endpoints déclarés.** Un endpoint déclaré et jamais appelé est
   un contrôle qui ne mesure rien. Un endpoint **enfant** n'est atteint que si son parent a
   rendu au moins un item : un enfant muet n'est donc pas automatiquement un défaut, ni
   automatiquement acceptable. Il se lit.
2. **`"mounted": false`**, c'est-à-dire que l'émulateur n'a pas de route pour cet appel. Contre
   un vrai fournisseur, c'est là qu'un endpoint déplacé se verrait.
3. **Le statut, et la classe qu'il a produite.** À croiser avec la clé `collection` de
   `--format json` : chaque unité y dit ce qui lui est arrivé.
4. **Les appels répétés.** Deux ressources déclarées qui partagent un endpoint source
   l'appellent deux fois par scan. Ce n'est pas un défaut de justesse, mais c'est un coût et un
   fait de limitation de débit.

## Committer un enregistrement

> **Aucun enregistrement n'entre au dépôt sans une relecture valeur par valeur.**

La rédaction du proxy protège les **en-têtes**, et c'est une **liste blanche**. C'est ce qui lui
fait masquer aussi `X-Content-Type-Options`, qu'aucune liste noire n'aurait pensé à nommer.
L'asymétrie est délibérée : un contrôle par nom répond « est-ce que ça ressemble à un secret »,
jamais « est-ce que ça n'en est certainement pas un ».

**Les corps sont la mesure, donc ils sont conservés en entier.** Contre un vrai tenant, ils
portent ses identifiants de ressources, ses noms de buckets et ses adresses IP. L'assainissement
partiel est le piège : l'audit de livraison s'est ouvert sur un UUID d'instance réelle oublié
dans une fixture dont l'adresse IP avait pourtant été assainie.

Un enregistrement versé dans `internal/genprovider/testdata/transcripts/` porte un manifeste
(`<fournisseur>.yaml`) qui dit ce qu'il est, contre quoi il a été pris, avec quelles variables
de chemin, et, sous `non_observes`, chaque endpoint déclaré que la session n'a **pas** exercé,
avec sa raison. Deux portes le tiennent honnête, et elles échouent dans les deux sens :

- `TestTheRecordedCollectionStillHappens` rejoue l'enregistrement contre le collecteur du jour.
  Moins d'appels que ce que l'enregistrement a vu, c'est une donnée qui a cessé d'arriver ; plus
  d'appels, c'est un endpoint qu'aucune session n'a jamais observé.
- `TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved` tient le registre `non_observes`
  exact. Un registre qui surestime le trou est aussi faux que celui qui le sous-estime.

Le rejeu emploie les réponses **enregistrées**, jamais des réponses construites depuis la spec
qu'il éprouve. Un harnais qui répondrait « ce que la spec attend » mesurerait sa propre copie de
la spec : un `items:` faux lui ferait servir le tableau faux, et il resterait vert.

## À lire aussi

- [Terraform ou live](../concepts/terraform-vs-live.fr.md) : ce que chaque source sait conclure.
- [Limites connues](../known-limitations.fr.md) : ce qui reste non observable, et qui peut le lever.
- `CONTRIBUTING.fr.md` : les portes de qualité qu'un changement doit franchir.
