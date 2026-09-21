package objectstorage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/objectstorage"
)

// La branche STOCKAGE OBJET de `collect.Classify`, mesurée contre de vraies
// erreurs S3 — et non plus déduite.
//
// # La réserve que ce fichier lève
//
// `Classify` reconnaît les erreurs du SDK AWS par INTERFACE ANONYME
// (`HTTPStatusCode() int`, `ErrorCode() string`), pour ne pas importer les
// paquets du SDK dans le moteur. C'est un bon choix, mais il a un défaut propre :
// une interface anonyme compile toujours. Rien ne dit qu'un vrai type d'erreur du
// SDK la satisfait, ni lequel des deux chemins se déclenche en premier sur une
// réponse réelle. Le guide `docs/guides/tracing-api-calls.md` le consignait : la
// seule fois où cette branche avait vu le réseau, l'émulateur rendait un `404`
// sans corps d'erreur — donc sans code.
//
// Ici, un serveur rend le XML d'erreur EXACT que l'API S3 spécifie, le vrai client
// du SDK le décode, et l'erreur traverse `CollectBuckets` telle qu'elle remonterait
// d'un endpoint souverain. Ce qui est mesuré est la chaîne entière : décodage du
// SDK, reconnaissance par interface, classement.
//
// # Ce que cela n'établit pas
//
// Que tel fournisseur souverain ÉMETTE tel code dans tel cas. Le corps est celui
// de la spécification S3 ; ce qu'un OOS ou un SOS répond vraiment reste dû à un
// scan réel (ADR-0012). La distinction est la même que pour l'émulateur, et elle
// est écrite ici plutôt que supposée.

// erreurS3 rend le corps XML qu'une API S3 renvoie pour un code donné. La forme
// vient de la spécification S3 : `<Error><Code>…</Code><Message>…</Message></Error>`.
func erreurS3(code, message string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>`+
		`<Error><Code>%s</Code><Message>%s</Message>`+
		`<RequestId>PEPIN0000000096</RequestId></Error>`, code, message)
}

func serveurS3(t *testing.T, status int, corps string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(corps))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// classe rend ce que Pépin conclut d'une réponse S3 donnée, en passant par le
// VRAI client du SDK et par le collecteur, pas par une erreur construite.
func classe(t *testing.T, status int, corps string) (model.CollectionOutcome, string) {
	t.Helper()
	url := serveurS3(t, status, corps)
	_, err := objectstorage.CollectBuckets(context.Background(),
		"scaleway", url, "fr-par", "PEPINTESTACCESSKEY00", "pepin-test-secret-key", false, true)
	if err == nil {
		t.Fatalf("HTTP %d : le collecteur n'a pas rendu d'erreur", status)
	}
	return collect.Classify(err)
}

// Chaque classe de Classify, atteinte par un corps d'erreur S3 réel.
func TestEveryClassIsReachedByARealS3Error(t *testing.T) {
	cas := []struct {
		nom    string
		status int
		code   string
		veut   model.CollectionOutcome
		parce  string
	}{
		{
			nom: "droit manquant sur le bucket", status: 403, code: "AccessDenied",
			veut:  model.OutcomePermissionDenied,
			parce: "la clé est bonne, la politique du bucket refuse",
		},
		{
			nom: "clé d'accès inconnue", status: 403, code: "InvalidAccessKeyId",
			veut:  model.OutcomeUnauthenticated,
			parce: "élargir une politique ne rend pas connue une clé qui ne l'est pas",
		},
		{
			nom: "signature invalide", status: 403, code: "SignatureDoesNotMatch",
			veut:  model.OutcomeUnauthenticated,
			parce: "le secret est faux : c'est l'identifiant, pas le droit",
		},
		{
			nom: "débit plafonné", status: 503, code: "SlowDown",
			veut:  model.OutcomeRateLimited,
			parce: "l'inventaire est tronqué, pas vide, et il faut réessayer",
		},
		{
			nom: "endpoint absent de ce tenant", status: 404, code: "NoSuchBucket",
			veut:  model.OutcomeNotFound,
			parce: "rien à lire ici, et ce n'est pas une panne",
		},
		{
			nom: "panne du service", status: 500, code: "InternalError",
			veut:  model.OutcomeUnavailable,
			parce: "le seul cas où « service indisponible » est vrai",
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, detail := classe(t, c.status, erreurS3(c.code, c.nom))
			if got != c.veut {
				t.Errorf("HTTP %d · %s → classé %q, attendu %q\n  %s",
					c.status, c.code, got, c.veut, c.parce)
			}
			if detail == "" {
				t.Error("détail vide : la réponse du fournisseur doit être conservée")
			}
		})
	}
}
