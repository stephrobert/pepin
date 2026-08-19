package assess

// AssessmentSurfaceVersion dit quelle version de la FORME du document
// assessment le code livre.
//
// Cette forme (statuts typés, références normatives, provenance du run) est
// une surface publique : un consommateur la parse dans `--format assessment`,
// dans assessment.json d'un bundle scellé, et c'est d'elle que découle
// l'OSCAL. Elle vient du module scankit (épinglé dans go.mod) : une montée de
// version de scankit peut donc la déplacer sans qu'aucune ligne de Pépin ne
// change, et c'est exactement la dérive que le gel rend visible.
//
// La forme observée est gelée dans cmd/testdata/frozen/assessment.json :
//   - si la forme bouge sans que la fixture bouge,
//     TestTheFrozenSurfacesStillMatchTheirFixture échoue (changement subi) ;
//   - si la fixture bouge sans que cette constante bouge,
//     TestASurfaceChangeDemandsItsVersionBump échoue (régénéré sans décider).
//
// Un changement délibéré : changer le code (ou monter scankit), lancer
// `mise run frozen-update`, incrémenter cette constante, écrire la ligne de
// CHANGELOG dont l'incrément est le signal.
const AssessmentSurfaceVersion = 1
