# La démo du README ANGLAIS (README.md), sous forme de SCRIPT et non d'un
# enregistrement que personne ne peut refaire.
#
# Sa jumelle française est tools/demo/quickstart.fr.tape. Les deux tapes exécutent
# EXACTEMENT les mêmes commandes ; seules la langue imposée (PEPIN_LANG) et les
# commentaires tapés à l'écran diffèrent. Le README anglais est la page d'accueil
# du projet : c'est celle dont la démo doit être lisible par tout le monde.
#
# Régénérer avec `mise run demo`. Tout ce qui s'affiche est réellement exécuté :
# le binaire est construit, il scanne un plan Terraform volontairement non
# conforme du dépôt, puis le même montage corrigé. Si une commande de ce tape
# casse, le GIF casse — c'est le but. Une démo qu'on ne peut pas rejouer est une
# affirmation, pas une preuve, et c'est exactement ce que Pépin reproche à un
# rapport de conformité.
#
# La version affichée n'est PAS écrite ici : le tape appelle `pepin version`, et
# la tâche mise injecte la version au build. Le GIF montre donc toujours la
# version réellement enregistrée, ce que docs/assets/quickstart.version consigne
# et que le preflight vérifie avant de laisser poser un tag.

Output docs/assets/quickstart.gif

Set Shell bash
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 14
Set TypingSpeed 35ms
Set Theme "Catppuccin Mocha"

Hide
# Le répertoire est capturé, jamais écrit en dur : un chemin figé publierait la
# disposition d'une machine et casserait sur toutes les autres.
Type "export REPO=$PWD && export PS1='$ ' && export PEPIN_LANG=en && clear" Enter
Sleep 1s
Show

Type "pepin version" Enter
Sleep 2s

Type "# A Terraform plan, no cloud account, nothing provisioned." Enter
Sleep 1500ms
Type "pepin scan scaleway --terraform examples/scaleway/terraform/plan.json" Enter
Sleep 6s

Type "# 1 = critical/high deviations. The CI gate is honest." Enter
Sleep 1500ms
Type "echo $?" Enter
Sleep 2s

Type "# The same setup, fixed:" Enter
Sleep 1500ms
Type "pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json" Enter
Sleep 6s

Type "echo $?" Enter
Sleep 3s
