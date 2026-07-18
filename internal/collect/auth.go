package collect

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// emptyPayloadHash est le SHA-256 d'un corps vide (requêtes GET sans body).
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// SigV4Auth signe les requêtes en AWS Signature V4. Couvre l'API Outscale (OAPI,
// service "oapi") et tout stockage objet S3-compatible (service "s3"). Le SDK
// natif Outscale (osc-sdk-go) signe en SigV4 ; on reproduit la même signature.
type SigV4Auth struct {
	Key, Secret, Service, Region string
}

// Apply signe la requête en place (en-têtes Authorization + X-Amz-Date).
func (a SigV4Auth) Apply(req *http.Request) error {
	hash, err := payloadHash(req)
	if err != nil {
		return err
	}
	creds := aws.Credentials{AccessKeyID: a.Key, SecretAccessKey: a.Secret}
	return v4.NewSigner().SignHTTP(req.Context(), creds, req, hash, a.Service, a.Region, time.Now())
}

// payloadHash retourne le SHA-256 hex du corps (vide si absent), en restaurant le
// corps pour qu'il reste lisible par le transport HTTP.
func payloadHash(req *http.Request) (string, error) {
	if req.Body == nil {
		return emptyPayloadHash, nil
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return "", fmt.Errorf("lecture du corps pour signature : %w", err)
	}
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(data))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ExoscaleAuth signe en EXO2-HMAC-SHA256 (schéma de l'API Exoscale v2). Réplique
// fidèle de egoscale/v3 api.go signRequest (non exporté) : chaîne signée =
// "<METHOD> <path>\n<body>\n<valeurs des query args triés>\n<headers vides>\n<expiration>".
type ExoscaleAuth struct {
	Key, Secret string
}

// Apply signe la requête en place (en-tête Authorization). L'expiration vaut
// l'instant + 10 minutes, comme le SDK Exoscale.
func (a ExoscaleAuth) Apply(req *http.Request) error {
	expiration := time.Now().UTC().Add(10 * time.Minute).Unix()

	var sigParts, headerParts []string
	sigParts = append(sigParts, fmt.Sprintf("%s %s", req.Method, req.URL.EscapedPath()))
	headerParts = append(headerParts, "EXO2-HMAC-SHA256 credential="+a.Key)

	body := ""
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("lecture du corps pour signature : %w", err)
		}
		_ = req.Body.Close()
		body = string(data)
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	sigParts = append(sigParts, body)

	names, values := exoSignedParams(req)
	sigParts = append(sigParts, values)
	if len(names) > 0 {
		headerParts = append(headerParts, "signed-query-args="+strings.Join(names, ";"))
	}

	sigParts = append(sigParts, "") // en-têtes signés : aucun
	sigParts = append(sigParts, fmt.Sprint(expiration))
	headerParts = append(headerParts, "expires="+fmt.Sprint(expiration))

	h := hmac.New(sha256.New, []byte(a.Secret))
	if _, err := h.Write([]byte(strings.Join(sigParts, "\n"))); err != nil {
		return fmt.Errorf("calcul HMAC : %w", err)
	}
	headerParts = append(headerParts, "signature="+base64.StdEncoding.EncodeToString(h.Sum(nil)))

	req.Header.Set("Authorization", strings.Join(headerParts, ","))
	return nil
}

// exoSignedParams retourne les noms (triés) des query args mono-valués et la
// concaténation de leurs valeurs (ordre alphabétique), comme l'exige Exoscale.
func exoSignedParams(req *http.Request) ([]string, string) {
	q := req.URL.Query()
	var names []string
	for p, v := range q {
		if len(v) == 1 {
			names = append(names, p)
		}
	}
	sort.Strings(names)
	values := ""
	for _, p := range names {
		values += q.Get(p)
	}
	return names, values
}
