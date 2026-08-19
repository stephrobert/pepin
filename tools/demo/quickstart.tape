# La démo du README, sous forme de SCRIPT et non d'un enregistrement que
# personne ne peut refaire.
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
Type "export REPO=$PWD && export PS1='$ ' && clear" Enter
Sleep 1s
Show

Type "pepin version" Enter
Sleep 2s

Type "# Un plan Terraform, aucun compte cloud, rien de provisionné." Enter
Sleep 1500ms
Type "pepin scan scaleway --terraform examples/scaleway/terraform/plan.json" Enter
Sleep 6s

Type "# 1 = des écarts critical/high. La porte de CI est honnête." Enter
Sleep 1500ms
Type "echo $?" Enter
Sleep 2s

Type "# Le même montage, corrigé :" Enter
Sleep 1500ms
Type "pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json" Enter
Sleep 6s

Type "echo $?" Enter
Sleep 3s
